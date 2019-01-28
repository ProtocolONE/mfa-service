package mfa

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"github.com/ProtocolONE/mfa-service/pkg/proto"
	"github.com/go-redis/redis"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"go.uber.org/zap"
	"image/png"
	"net/url"
	"regexp"
	"strings"
)

const (
	Version                   = "latest"
	ServiceName               = "p1mfa"
	qrUrlPattern              = "https://chart.googleapis.com/chart?cht=qr&chs=200x200&chl=%s"
	mfaRecoveryStoragePattern = "mfa_recovery_%s_%s"
	mfaSecretStoragePattern   = "mfa_secret_%s"

	ErrorSecretKeyNotExists      = "Secret key not exists"
	ErrorCodeInvalid             = "Invalid code"
	ErrorRequestPropertyRequired = "%s is required field"
)

type service struct {
	redis  *redis.Client
	logger *zap.Logger
}

func NewService(redis *redis.Client, logger *zap.Logger) *service {
	return &service{
		redis:  redis,
		logger: logger,
	}
}

func (s *service) Create(ctx context.Context, req *proto.MfaCreateDataRequest, res *proto.MfaCreateDataResponse) error {
	if err := s.validateCreateRequest(req); err != nil {
		s.logger.Error("Validate create request failed with error", zap.Error(err))

		return err
	}

	an := req.UserID
	if req.Email != "" {
		an = req.Email
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      req.AppName,
		AccountName: an,
	})
	if err != nil {
		s.logger.Error("Generate a new TOTP Key failed with error", zap.Error(err))
		return err
	}

	imageBased, err := s.generateBase64QrCode(key, int(req.QrSize))
	if err != nil {
		s.logger.Error("Generate base 64 qr code with error", zap.Error(err))

		return err
	}
	codes, err := s.generateRecoveryCodes(10)
	if err != nil {
		s.logger.Error("Generate recovery codes failed with error", zap.Error(err))

		return err
	}
	if err = s.redis.SAdd(s.GetRecoveryStorageKey(req.ProviderID, req.UserID), codes).Err(); err != nil {
		s.logger.Error("Add recovery codes to Redis failed with error", zap.Error(err))

		return err
	}
	if err = s.redis.HSet(s.GetSecretStorageKey(req.UserID), req.ProviderID, key.Secret()).Err(); err != nil {
		s.logger.Error("Add secret codes to Redis failed with error", zap.Error(err))

		return err
	}

	res.SecretKey = key.Secret()
	res.URL = key.URL()
	res.ImageBased = imageBased
	res.QrCodeURL = fmt.Sprintf(qrUrlPattern, url.QueryEscape(key.URL()))
	res.RecoveryCode = codes

	return nil
}

func (s *service) Check(ctx context.Context, req *proto.MfaCheckDataRequest, res *proto.MfaCheckDataResponse) error {
	if err := s.validateCheckRequest(req); err != nil {
		s.logger.Error("Validate check request failed with error", zap.Error(err))

		return err
	}

	res.Result = false
	secret := s.redis.HGet(s.GetSecretStorageKey(req.UserID), req.ProviderID)
	if secret.Err() != nil {
		s.logger.Error("Getting secret key from Redis failed with error", zap.Error(secret.Err()))

		res.Error = &proto.Error{
			Message: ErrorSecretKeyNotExists,
		}
		return secret.Err()
	}

	if len(regexp.MustCompile("[0-9]{6}").FindStringSubmatch(req.Code)) > 0 {
		res.Result = totp.Validate(req.Code, secret.Val())
		if res.Result != false {
			s.logger.Warn("Validating TOTP code format failed", zap.String("code", req.Code))

			res.Error = &proto.Error{
				Message: ErrorCodeInvalid,
			}
		}
	} else {
		r := s.redis.SRem(s.GetRecoveryStorageKey(req.UserID, req.ProviderID), req.Code)
		if r.Err() != nil || r.Val() < 1 {
			s.logger.Warn(
				"Removing recovery code from Redis failed",
				zap.Error(r.Err()),
				zap.String("userId", req.UserID),
				zap.String("providerId", req.ProviderID),
			)

			res.Error = &proto.Error{
				Message: ErrorCodeInvalid,
			}
		} else {
			res.Result = true
		}
	}

	return nil
}

func (s *service) generateRecoveryCodes(count int) (codes []string, err error) {
	secret := make([]byte, 10)
	for i := 0; i < count; i++ {
		if _, err := rand.Read(secret); err != nil {
			return nil, err
		}
		codes = append(codes, strings.TrimRight(base32.StdEncoding.EncodeToString(secret), "="))
	}
	return codes, nil
}

func (s *service) generateBase64QrCode(key *otp.Key, size int) (string, error) {
	if size == 0 {
		size = 200
	}

	var buf bytes.Buffer
	img, err := key.Image(size, size)
	if err != nil {
		return "", err
	}

	err = png.Encode(&buf, img)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func (s *service) validateCreateRequest(req *proto.MfaCreateDataRequest) error {
	if req.ProviderID == "" {
		return fmt.Errorf(ErrorRequestPropertyRequired, "ProviderID")
	}
	if req.AppName == "" {
		return fmt.Errorf(ErrorRequestPropertyRequired, "AppName")
	}
	if req.UserID == "" {
		return fmt.Errorf(ErrorRequestPropertyRequired, "UserID")
	}
	return nil
}

func (s *service) validateCheckRequest(req *proto.MfaCheckDataRequest) error {
	if req.ProviderID == "" {
		return fmt.Errorf(ErrorRequestPropertyRequired, "ProviderID")
	}
	if req.UserID == "" {
		return fmt.Errorf(ErrorRequestPropertyRequired, "UserID")
	}
	if req.Code == "" {
		return fmt.Errorf(ErrorRequestPropertyRequired, "Code")
	}
	return nil
}

func (s *service) GetRecoveryStorageKey(userId string, ProviderID string) string {
	return fmt.Sprintf(mfaRecoveryStoragePattern, userId, ProviderID)
}

func (s *service) GetSecretStorageKey(userId string) string {
	return fmt.Sprintf(mfaSecretStoragePattern, userId)
}
