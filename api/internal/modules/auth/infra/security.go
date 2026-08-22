package infra

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"

	"aether/internal/modules/auth/domain"
)

type Hasher struct{}

func NewHasher() *Hasher { return &Hasher{} }

func (h *Hasher) Hash(ctx context.Context, password string) (string, error) {
	raw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(raw), err
}

func (h *Hasher) Verify(ctx context.Context, password, hash string) bool {
	if strings.HasPrefix(hash, "$argon2id$") {
		return verifyArgon2ID(password, hash)
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func verifyArgon2ID(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) < 6 {
		return false
	}
	var params struct {
		Memory      uint32
		Iterations  uint32
		Parallelism uint8
	}
	for _, p := range strings.Split(parts[3], ",") {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "m":
			v, _ := strconv.ParseUint(kv[1], 10, 32)
			params.Memory = uint32(v)
		case "t":
			v, _ := strconv.ParseUint(kv[1], 10, 32)
			params.Iterations = uint32(v)
		case "p":
			v, _ := strconv.ParseUint(kv[1], 10, 8)
			params.Parallelism = uint8(v)
		}
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, uint32(len(expected)))
	return hmac.Equal(actual, expected)
}

type Signer struct {
	secret []byte
}

func NewSigner(secret string) *Signer {
	return &Signer{secret: []byte(secret)}
}

type payload struct {
	Sub  string `json:"sub"`
	Org  string `json:"org"`
	Role string `json:"role"`
	Glob string `json:"glob"`
	Exp  int64  `json:"exp"`
}

func (s *Signer) Sign(ctx context.Context, subject, orgID uuid.UUID, role domain.Role, global string, ttl time.Duration) (string, error) {
	body, err := json.Marshal(payload{
		Sub: subject.String(), Org: orgID.String(), Role: string(role), Glob: global,
		Exp: time.Now().Add(ttl).Unix(),
	})
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(encoded))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encoded + "." + sig, nil
}

func (s *Signer) Verify(ctx context.Context, token string) (*domain.AuthToken, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, domain.ErrUnauthorized
	}
	expected := hmac.New(sha256.New, s.secret)
	expected.Write([]byte(parts[0]))
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(sig, expected.Sum(nil)) {
		return nil, domain.ErrUnauthorized
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	var p payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, domain.ErrUnauthorized
	}
	if p.Exp < time.Now().Unix() {
		return nil, domain.ErrUnauthorized
	}
	sub, err := uuid.Parse(p.Sub)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	org, err := uuid.Parse(p.Org)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	if !domain.Role(p.Role).Valid() {
		return nil, domain.ErrUnauthorized
	}
	return &domain.AuthToken{
		Subject: sub, OrgID: org, Role: domain.Role(p.Role), Global: p.Glob,
		Expires: time.Unix(p.Exp, 0),
	}, nil
}
