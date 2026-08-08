package core

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

type TOTP struct {
	Secret string
}

func (c *Core) EnrollTOTP(userID string) (*TOTP, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	secret := strings.TrimRight(base32.StdEncoding.EncodeToString(raw), "=")
	enc, err := c.Secrets.EncryptString(secret)
	if err != nil {
		return nil, err
	}
	if err := c.Store.SetTOTP(userID, enc, false); err != nil {
		return nil, err
	}
	return &TOTP{Secret: secret}, nil
}

func (c *Core) VerifyTOTP(userID, code string) (bool, error) {
	enc, enabled, err := c.Store.GetTOTP(userID)
	if err != nil || !enabled {
		return false, err
	}
	secret, err := c.Secrets.DecryptString(enc)
	if err != nil {
		return false, err
	}
	code = strings.TrimSpace(code)
	now := time.Now().Unix()
	for i := int64(-1); i <= 1; i++ {
		if totpCode(secret, now+i*30) == code {
			return true, nil
		}
	}
	return false, nil
}

func (c *Core) EnableTOTP(userID, code string) error {
	enc, _, err := c.Store.GetTOTP(userID)
	if err != nil {
		return err
	}
	secret, err := c.Secrets.DecryptString(enc)
	if err != nil {
		return err
	}
	if totpCode(secret, time.Now().Unix()) != strings.TrimSpace(code) {
		return fmt.Errorf("código inválido")
	}
	return c.Store.SetTOTP(userID, enc, true)
}

func (c *Core) DisableTOTP(userID string) error {
	return c.Store.SetTOTP(userID, "", false)
}

func (c *Core) TOTPEnabled(userID string) (bool, error) {
	_, enabled, err := c.Store.GetTOTP(userID)
	return enabled, err
}

func (c *Core) TOTPProvisioningURI(userID, email string) (string, error) {
	enc, _, err := c.Store.GetTOTP(userID)
	if err != nil {
		return "", err
	}
	secret, err := c.Secrets.DecryptString(enc)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("otpauth://totp/Aether:%s?secret=%s&issuer=Aether&algorithm=SHA1&digits=6&period=30",
		email, secret), nil
}

func totpCode(secret string, at int64) string {
	key, err := base32.StdEncoding.DecodeString(strings.ToUpper(secret) + strings.Repeat("=", (8-len(secret)%8)%8))
	if err != nil {
		return ""
	}
	counter := make([]byte, 8)
	binary.BigEndian.PutUint64(counter, uint64(at/30))
	mac := hmac.New(sha1.New, key)
	mac.Write(counter)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	code := (uint32(sum[offset])&0x7f)<<24 | uint32(sum[offset+1])<<16 | uint32(sum[offset+2])<<8 | uint32(sum[offset+3])
	return fmt.Sprintf("%06d", code%1000000)
}
