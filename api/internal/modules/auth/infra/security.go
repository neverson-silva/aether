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
	Kind string `json:"kind"`
	Sid  string `json:"sid,omitempty"`
	Jti  string `json:"jti,omitempty"`
}

const (
	accessTokenMaxTTL  = 10 * time.Minute
	refreshTokenMaxTTL = 30 * 24 * time.Hour
)

func (s *Signer) Sign(ctx context.Context, subject, orgID uuid.UUID, role domain.Role, global string, ttl time.Duration) (string, error) {
	return s.sign(ctx, uuid.Nil, subject, orgID, role, global, ttl, "access", time.Time{})
}

func (s *Signer) SignWithSession(ctx context.Context, sessionID, subject, orgID uuid.UUID, role domain.Role, global string, ttl time.Duration) (string, error) {
	return s.sign(ctx, sessionID, subject, orgID, role, global, ttl, "access", time.Time{})
}

func (s *Signer) sign(ctx context.Context, sessionID, subject, orgID uuid.UUID, role domain.Role, global string, ttl time.Duration, kind string, expiresAt time.Time) (string, error) {
	if ttl > accessTokenMaxTTL {
		ttl = accessTokenMaxTTL
	}
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(ttl)
	}
	if sessionID != uuid.Nil {
		if kind == "access" {
			return s.signPayload(payload{Sub: subject.String(), Org: orgID.String(), Role: string(role), Glob: global, Exp: expiresAt.Unix(), Kind: kind, Sid: sessionID.String()})
		}
		return s.signPayload(payload{Sub: subject.String(), Org: orgID.String(), Role: string(role), Glob: global, Exp: expiresAt.Unix(), Kind: kind, Sid: sessionID.String()})
	}
	return s.signPayload(payload{Sub: subject.String(), Org: orgID.String(), Role: string(role), Glob: global, Exp: expiresAt.Unix(), Kind: kind})
}

func (s *Signer) signPayload(p payload) (string, error) {
	body, err := json.Marshal(payload{
		Sub: p.Sub, Org: p.Org, Role: p.Role, Glob: p.Glob, Exp: p.Exp, Kind: p.Kind, Sid: p.Sid, Jti: p.Jti,
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

func (s *Signer) SignRefresh(ctx context.Context, subject, orgID uuid.UUID, role domain.Role, global string, ttl time.Duration) (string, error) {
	return s.signRefresh(ctx, uuid.Nil, subject, orgID, role, global, ttl)
}

func (s *Signer) SignRefreshWithSession(ctx context.Context, sessionID, subject, orgID uuid.UUID, role domain.Role, global string, ttl time.Duration) (string, error) {
	return s.signRefresh(ctx, sessionID, subject, orgID, role, global, ttl)
}

func (s *Signer) signRefresh(ctx context.Context, sessionID, subject, orgID uuid.UUID, role domain.Role, global string, ttl time.Duration) (string, error) {
	if ttl > refreshTokenMaxTTL {
		ttl = refreshTokenMaxTTL
	}
	return s.signRefreshUntil(ctx, sessionID, subject, orgID, role, global, time.Now().Add(ttl))
}

func (s *Signer) SignRefreshUntil(ctx context.Context, subject, orgID uuid.UUID, role domain.Role, global string, expiresAt time.Time) (string, error) {
	return s.signRefreshUntil(ctx, uuid.Nil, subject, orgID, role, global, expiresAt)
}

func (s *Signer) SignRefreshUntilWithSession(ctx context.Context, sessionID, subject, orgID uuid.UUID, role domain.Role, global string, expiresAt time.Time) (string, error) {
	return s.signRefreshUntil(ctx, sessionID, subject, orgID, role, global, expiresAt)
}

func (s *Signer) IssueRefreshWithSession(ctx context.Context, sessionID, subject, orgID uuid.UUID, role domain.Role, global string, expiresAt time.Time) (string, uuid.UUID, error) {
	tokenID := uuid.New()
	if expiresAt.After(time.Now().Add(refreshTokenMaxTTL)) {
		expiresAt = time.Now().Add(refreshTokenMaxTTL)
	}
	token, err := s.signPayload(payload{Sub: subject.String(), Org: orgID.String(), Role: string(role), Glob: global, Exp: expiresAt.Unix(), Kind: "refresh", Sid: sessionString(sessionID), Jti: tokenID.String()})
	return token, tokenID, err
}

func (s *Signer) signRefreshUntil(ctx context.Context, sessionID, subject, orgID uuid.UUID, role domain.Role, global string, expiresAt time.Time) (string, error) {
	maxExpiry := time.Now().Add(refreshTokenMaxTTL)
	if expiresAt.After(maxExpiry) {
		expiresAt = maxExpiry
	}
	return s.signPayload(payload{Sub: subject.String(), Org: orgID.String(), Role: string(role), Glob: global, Exp: expiresAt.Unix(), Kind: "refresh", Sid: sessionString(sessionID)})
}

func sessionString(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
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
	sessionID := uuid.Nil
	if p.Sid != "" {
		sessionID, err = uuid.Parse(p.Sid)
		if err != nil {
			return nil, domain.ErrUnauthorized
		}
	}
	tokenID := uuid.Nil
	if p.Jti != "" {
		tokenID, err = uuid.Parse(p.Jti)
		if err != nil {
			return nil, domain.ErrUnauthorized
		}
	}
	return &domain.AuthToken{
		Subject: sub, OrgID: org, Role: domain.Role(p.Role), Global: p.Glob,
		Expires: time.Unix(p.Exp, 0), Kind: p.Kind, SessionID: sessionID, TokenID: tokenID,
	}, nil
}
