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

type Service struct {
	Redis *redis.Client
}

func (s *Service) Create(ctx context.Context, req *proto.MfaCreateDataRequest, res *proto.MfaCreateDataResponse) error {
	if err := s.validateCreateRequest(req); err != nil {
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
		return err
	}

	imageBased, err := s.generateBase64QrCode(key, int(req.QrSize))
	if err != nil {
		return err
	}
	codes, err := s.generateRecoveryCodes(10)
	if err != nil {
		return err
	}
	if err = s.Redis.SAdd(s.GetRecoveryStorageKey(req.ProviderID, req.UserID), codes).Err(); err != nil {
		return err
	}
	if err = s.Redis.HSet(s.GetSecretStorageKey(req.UserID), req.ProviderID, key.Secret()).Err(); err != nil {
		return err
	}

	res.SecretKey = key.Secret()
	res.URL = key.URL()
	res.ImageBased = imageBased
	res.QrCodeURL = fmt.Sprintf(qrUrlPattern, url.QueryEscape(key.URL()))
	res.RecoveryCode = codes

	return nil
}

func (s *Service) Check(ctx context.Context, req *proto.MfaCheckDataRequest, res *proto.MfaCheckDataResponse) error {
	if err := s.validateCheckRequest(req); err != nil {
		return err
	}

	res.Result = false
	secret := s.Redis.HGet(s.GetSecretStorageKey(req.UserID), req.ProviderID)
	if secret.Err() != nil {
		res.Error = &proto.Error{
			Message: ErrorSecretKeyNotExists,
		}
		return secret.Err()
	}

	if len(regexp.MustCompile("[0-9]{6}").FindStringSubmatch(req.Code)) > 0 {
		res.Result = totp.Validate(req.Code, secret.Val())
		if res.Result != false {
			res.Error = &proto.Error{
				Message: ErrorCodeInvalid,
			}
		}
	} else {
		r := s.Redis.SRem(s.GetRecoveryStorageKey(req.UserID, req.ProviderID), req.Code)
		if r.Err() != nil || r.Val() < 1 {
			res.Error = &proto.Error{
				Message: ErrorCodeInvalid,
			}
		} else {
			res.Result = true
		}
	}

	return nil
}

func (s *Service) generateRecoveryCodes(count int) (codes []string, err error) {
	secret := make([]byte, 10)
	for i := 0; i < count; i++ {
		if _, err := rand.Read(secret); err != nil {
			return nil, err
		}
		codes = append(codes, strings.TrimRight(base32.StdEncoding.EncodeToString(secret), "="))
	}
	return codes, nil
}

func (s *Service) generateBase64QrCode(key *otp.Key, size int) (string, error) {
	if size == 0 {
		size = 200
	}

	var buf bytes.Buffer
	img, err := key.Image(size, size)
	if err != nil {
		return "", err
	}
	png.Encode(&buf, img)

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func (s *Service) validateCreateRequest(req *proto.MfaCreateDataRequest) error {
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

func (s *Service) validateCheckRequest(req *proto.MfaCheckDataRequest) error {
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

func (s *Service) GetRecoveryStorageKey(userId string, ProviderID string) string {
	return fmt.Sprintf(mfaRecoveryStoragePattern, userId, ProviderID)
}

func (s *Service) GetSecretStorageKey(userId string) string {
	return fmt.Sprintf(mfaSecretStoragePattern, userId)
}
