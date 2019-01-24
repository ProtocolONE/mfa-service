package mfa_test

import (
	"context"
	"github.com/ProtocolONE/mfa-service/pkg"
	"github.com/ProtocolONE/mfa-service/pkg/proto"
	"github.com/go-redis/redis"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"math/rand"
	"regexp"
	"strconv"
	"testing"
	"time"
)

type ServiceTestSuite struct {
	suite.Suite
	service    *mfa.Service
	redis      *redis.Client
	userID     string
	ProviderID string
}

func Test_Service(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

func (suite *ServiceTestSuite) SetupTest() {
	r := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})

	suite.redis = r
	suite.service = &mfa.Service{Redis: r}
	suite.userID = strconv.Itoa(random(1000000, 9999999))
	suite.ProviderID = strconv.Itoa(random(1000000, 9999999))
}

func (suite *ServiceTestSuite) TearDownTest() {
	suite.deleteKeys()
	if err := suite.redis.Close(); err != nil {
		panic(err)
	}
}

func (suite *ServiceTestSuite) TestCreateToReturnErrorRequestData() {
	reqs := []proto.MfaCreateDataRequest{
		{ProviderID: "", AppName: "test", UserID: "1"},
		{ProviderID: "1", AppName: "", UserID: "1"},
		{ProviderID: "1", AppName: "test", UserID: ""},
	}
	for _, req := range reqs {
		err := suite.service.Create(context.TODO(), &req, &proto.MfaCreateDataResponse{})
		assert.Regexp(suite.T(), regexp.MustCompile("is required field"), err)
	}
}

func (suite *ServiceTestSuite) TestCreateToReturnSuccessResponse() {
	res := &proto.MfaCreateDataResponse{}
	req := proto.MfaCreateDataRequest{ProviderID: suite.ProviderID, AppName: "test", UserID: suite.userID}
	err := suite.service.Create(context.TODO(), &req, res)

	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), res.SecretKey)
	assert.NotEmpty(suite.T(), res.ImageBased)
	assert.Regexp(suite.T(), regexp.MustCompile("otpauth://totp/"), res.URL)
	assert.Regexp(suite.T(), regexp.MustCompile("https://chart.googleapis.com/chart\\?cht=qr"), res.QrCodeURL)
	assert.Equal(suite.T(), 10, len(res.RecoveryCode))
}

func (suite *ServiceTestSuite) TestCreateToReturnUserIdAsAccountName() {
	res := &proto.MfaCreateDataResponse{}
	req := proto.MfaCreateDataRequest{ProviderID: suite.ProviderID, AppName: "test", UserID: suite.userID}
	err := suite.service.Create(context.TODO(), &req, res)

	assert.NoError(suite.T(), err)
	assert.Regexp(suite.T(), regexp.MustCompile(suite.userID), res.URL)
}

func (suite *ServiceTestSuite) TestCreateToReturnEmailAsAccountName() {
	res := &proto.MfaCreateDataResponse{}
	req := proto.MfaCreateDataRequest{ProviderID: suite.ProviderID, AppName: "test", UserID: suite.userID, Email: "emailasaccount"}
	err := suite.service.Create(context.TODO(), &req, res)

	assert.NoError(suite.T(), err)
	assert.Regexp(suite.T(), regexp.MustCompile("emailasaccount"), res.URL)
}

func (suite *ServiceTestSuite) TestCheckToReturnErrorRequestData() {
	reqs := []proto.MfaCheckDataRequest{
		{ProviderID: "", UserID: "test", Code: "test"},
		{ProviderID: "test", UserID: "", Code: "test"},
		{ProviderID: "test", UserID: "test", Code: ""},
	}
	for _, req := range reqs {
		err := suite.service.Check(context.TODO(), &req, &proto.MfaCheckDataResponse{})
		assert.Regexp(suite.T(), regexp.MustCompile("is required field"), err)
	}
}

func (suite *ServiceTestSuite) TestCheckToReturnFalseWithoutSecretKey() {
	res := &proto.MfaCheckDataResponse{}
	req := &proto.MfaCheckDataRequest{ProviderID: suite.ProviderID, UserID: suite.userID, Code: "code"}
	err := suite.service.Check(context.TODO(), req, res)

	assert.Error(suite.T(), err)
	assert.False(suite.T(), res.Result)
	assert.Equal(suite.T(), mfa.ErrorSecretKeyNotExists, res.Error.Message)
}

func (suite *ServiceTestSuite) TestCheckToReturnFalseWithRecoveryKey() {
	res := &proto.MfaCheckDataResponse{}
	req := &proto.MfaCheckDataRequest{ProviderID: suite.ProviderID, UserID: suite.userID, Code: "invalidrecoverycode"}
	err := suite.service.Check(context.TODO(), req, res)

	assert.Error(suite.T(), err)
	assert.False(suite.T(), res.Result)
	assert.Equal(suite.T(), mfa.ErrorSecretKeyNotExists, res.Error.Message)
}

func (suite *ServiceTestSuite) TestCheckToReturnTrueWithRecoveryKey() {
	res1 := &proto.MfaCreateDataResponse{}
	req1 := proto.MfaCreateDataRequest{ProviderID: suite.ProviderID, AppName: "test", UserID: suite.userID}
	suite.service.Create(context.TODO(), &req1, res1)

	res2 := &proto.MfaCheckDataResponse{}
	req2 := &proto.MfaCheckDataRequest{ProviderID: suite.ProviderID, UserID: suite.userID, Code: res1.RecoveryCode[0]}
	err2 := suite.service.Check(context.TODO(), req2, res2)

	assert.NoError(suite.T(), err2)
	assert.True(suite.T(), res2.Result)
}

func (suite *ServiceTestSuite) TestCheckToReturnFalseIfRecoveryKeysEmpty() {
	res1 := &proto.MfaCreateDataResponse{}
	req1 := proto.MfaCreateDataRequest{ProviderID: suite.ProviderID, AppName: "test", UserID: suite.userID}
	suite.service.Create(context.TODO(), &req1, res1)

	for i := 0; i < len(res1.RecoveryCode); i++ {
		res2 := &proto.MfaCheckDataResponse{}
		req2 := &proto.MfaCheckDataRequest{ProviderID: suite.ProviderID, UserID: suite.userID, Code: res1.RecoveryCode[0]}
		suite.service.Check(context.TODO(), req2, res2)
	}

	res2 := &proto.MfaCheckDataResponse{}
	req2 := &proto.MfaCheckDataRequest{ProviderID: suite.ProviderID, UserID: suite.userID, Code: res1.RecoveryCode[0]}
	err2 := suite.service.Check(context.TODO(), req2, res2)

	assert.NoError(suite.T(), err2)
	assert.False(suite.T(), res2.Result)
	assert.Equal(suite.T(), mfa.ErrorCodeInvalid, res2.Error.Message)
}

func (suite *ServiceTestSuite) TestCheckToReturnFalseWithOtpKey() {
	res := &proto.MfaCheckDataResponse{}
	req := &proto.MfaCheckDataRequest{ProviderID: suite.ProviderID, UserID: suite.userID, Code: "123456"}
	err := suite.service.Check(context.TODO(), req, res)

	assert.Error(suite.T(), err)
	assert.False(suite.T(), res.Result)
	assert.Equal(suite.T(), mfa.ErrorSecretKeyNotExists, res.Error.Message)
}

func (suite *ServiceTestSuite) TestCheckToReturnTrueWithOtpKey() {
	res1 := &proto.MfaCreateDataResponse{}
	req1 := proto.MfaCreateDataRequest{ProviderID: suite.ProviderID, AppName: "test", UserID: suite.userID}
	suite.service.Create(context.TODO(), &req1, res1)
	code, _ := totp.GenerateCode(res1.GetSecretKey(), time.Now())

	res2 := &proto.MfaCheckDataResponse{}
	req2 := &proto.MfaCheckDataRequest{ProviderID: suite.ProviderID, UserID: suite.userID, Code: code}
	err2 := suite.service.Check(context.TODO(), req2, res2)

	assert.NoError(suite.T(), err2)
	assert.True(suite.T(), res2.Result)
}

func (suite *ServiceTestSuite) deleteKeys() error {
	return suite.redis.Del(
		suite.service.GetRecoveryStorageKey(suite.userID, suite.ProviderID),
		suite.service.GetSecretStorageKey(suite.userID),
	).Err()
}

func random(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}
