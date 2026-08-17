package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict")
	ErrValidation = errors.New("invalid input")
	ErrForbidden  = errors.New("access denied")
)

type Branding struct {
	OrgID        uuid.UUID
	Name         string
	LogoURL      string
	PrimaryColor string
	AccentColor  string
	DarkMode     bool
	UpdatedAt    time.Time
}

type S3Destination struct {
	ID           uuid.UUID
	OrgID        uuid.UUID
	Name         string
	Endpoint     string
	Bucket       string
	Region       string
	AccessKeyEnc string
	SecretKeyEnc string
	CreatedAt    time.Time
}

type OIDCProvider struct {
	ID              uuid.UUID
	OrgID           uuid.UUID
	Name            string
	Issuer          string
	ClientID        string
	ClientSecretEnc string
	Scopes          string
	Enabled         bool
	CreatedAt       time.Time
}

type PasswordCipher interface {
	Encrypt(plain string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type OIDCDiscoverer interface {
	AuthURL(ctx context.Context, issuer, clientID, scopes, providerID string) (string, error)
	Exchange(ctx context.Context, issuer, clientID, clientSecret, providerID, code string) (user *OIDCUser, err error)
}

type OIDCUser struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Sub           string `json:"sub"`
}

type Store interface {
	GetBranding(ctx context.Context, orgID uuid.UUID) (*Branding, error)
	SaveBranding(ctx context.Context, branding *Branding) (*Branding, error)

	CreateS3(ctx context.Context, dest *S3Destination) (*S3Destination, error)
	ListS3ByOrg(ctx context.Context, orgID uuid.UUID) ([]S3Destination, error)
	DeleteS3(ctx context.Context, id, orgID uuid.UUID) error

	CreateOIDC(ctx context.Context, provider *OIDCProvider) (*OIDCProvider, error)
	ListOIDCByOrg(ctx context.Context, orgID uuid.UUID) ([]OIDCProvider, error)
	ListEnabledOIDC(ctx context.Context) ([]OIDCProvider, error)
	CountEnabledOIDC(ctx context.Context) (int, error)
	GetOIDC(ctx context.Context, id uuid.UUID) (*OIDCProvider, error)
	DeleteOIDC(ctx context.Context, id, orgID uuid.UUID) error
}
