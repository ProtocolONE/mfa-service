package mfa

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/pquerna/otp/totp"
	"image/png"
	"net/url"
	"p1mfa/pkg/proto"
	"regexp"
	"strings"
)

const (
	Version                   = "latest"
	ServiceName               = "p1mfa"
	qrUrlPattern              = "https://chart.googleapis.com/chart?cht=qr&chs=%dx%d&chl=%s"
	mfaRecoveryStoragePattern = "mfa_recovery_%s"
	mfaSecretStoragePattern   = "mfa_secret"
	ErrorSecretKeyNotExists   = "Secret key not exists"
	ErrorCodeInvalid          = "Invalid code"
)

type Service struct {
	Redis      *redis.Client
	OpsCounter func()
}

func (s *Service) Create(ctx context.Context, req *proto.MfaCreateDataRequest, res *proto.MfaCreateDataResponse) error {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      req.AppName,
		AccountName: req.Email,
	})
	if err != nil {
		return err
	}

	size := 200
	if req.QrSize != 0 {
		size = int(req.QrSize)
	}

	var buf bytes.Buffer
	img, err := key.Image(size, size)
	if err != nil {
		return err
	}
	png.Encode(&buf, img)

	codes, err := s.generateRecoveryCodes(10)
	if err != nil {
		return err
	}
	if err = s.Redis.SAdd(fmt.Sprintf(mfaRecoveryStoragePattern, req.UserID), codes).Err(); err != nil {
		return err
	}
	if err = s.Redis.HSet(mfaSecretStoragePattern, req.UserID, key.Secret()).Err(); err != nil {
		return err
	}

	res.SecretKey = key.Secret()
	res.URL = key.URL()
	res.ImageBased = base64.StdEncoding.EncodeToString(buf.Bytes())
	res.QrCodeURL = fmt.Sprintf(qrUrlPattern, size, size, url.QueryEscape(key.URL()))
	res.RecoveryCode = codes

	s.OpsCounter()
	return nil
}

func (s *Service) Check(ctx context.Context, req *proto.MfaCheckDataRequest, res *proto.MfaCheckDataResponse) error {
	res.Result = false
	secret := s.Redis.HGet(mfaSecretStoragePattern, req.UserID)
	if secret.Err() != nil {
		res.Error.Message = ErrorSecretKeyNotExists
		return secret.Err()
	}

	if len(regexp.MustCompile("[0-9]{6}").FindStringSubmatch(req.Code)) > 0 {
		res.Result = totp.Validate(req.Code, secret.String())

		if res.Result != false {
			res.Error.Message = ErrorCodeInvalid
		}
	} else {
		if s.Redis.SRem(fmt.Sprintf(mfaRecoveryStoragePattern, req.UserID), req.Code).Err() != nil {
			res.Error.Message = ErrorCodeInvalid
		} else {
			res.Result = true
		}
	}

	s.OpsCounter()
	return nil
}

func (s *Service) generateRecoveryCodes(count int) (codes []string, err error) {
	secret := make([]byte, 10)
	for i := 0; i < count; i++ {
		if _, err = rand.Read(secret); err != nil {
			return nil, err
		}
		codes = append(codes, strings.TrimRight(base32.StdEncoding.EncodeToString(secret), "="))
	}
	return codes, nil
}
