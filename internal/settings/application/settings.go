package application

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"aether/internal/settings/domain"
)

type Settings struct {
	Store     domain.Store
	Passwords domain.PasswordCipher
	OIDC      domain.OIDCDiscoverer
}

func (s *Settings) GetBranding(ctx context.Context, orgID uuid.UUID) (*domain.Branding, error) {
	branding, err := s.Store.GetBranding(ctx, orgID)
	if errors.Is(err, domain.ErrNotFound) {
		return &domain.Branding{OrgID: orgID}, nil
	}
	return branding, err
}

func (s *Settings) SaveBranding(ctx context.Context, orgID uuid.UUID, branding *domain.Branding) (*domain.Branding, error) {
	branding.OrgID = orgID
	return s.Store.SaveBranding(ctx, branding)
}

func (s *Settings) CreateS3(ctx context.Context, orgID uuid.UUID, name, endpoint, bucket, region, accessKey, secretKey string) (*domain.S3Destination, error) {
	name = strings.TrimSpace(name)
	if name == "" || endpoint == "" || bucket == "" {
		return nil, domain.ErrValidation
	}
	if region == "" {
		region = "us-east-1"
	}
	accessEnc, err := s.Passwords.Encrypt(accessKey)
	if err != nil {
		return nil, err
	}
	secretEnc, err := s.Passwords.Encrypt(secretKey)
	if err != nil {
		return nil, err
	}
	return s.Store.CreateS3(ctx, &domain.S3Destination{
		OrgID: orgID, Name: name, Endpoint: endpoint, Bucket: bucket, Region: region,
		AccessKeyEnc: accessEnc, SecretKeyEnc: secretEnc,
	})
}

func (s *Settings) ListS3(ctx context.Context, orgID uuid.UUID) ([]domain.S3Destination, error) {
	return s.Store.ListS3ByOrg(ctx, orgID)
}

func (s *Settings) DeleteS3(ctx context.Context, id, orgID uuid.UUID) error {
	return s.Store.DeleteS3(ctx, id, orgID)
}

func (s *Settings) CreateOIDC(ctx context.Context, orgID uuid.UUID, name, issuer, clientID, clientSecret, scopes string) (*domain.OIDCProvider, error) {
	name = strings.TrimSpace(name)
	if name == "" || issuer == "" || clientID == "" {
		return nil, domain.ErrValidation
	}
	if scopes == "" {
		scopes = "openid email profile"
	}
	secretEnc, err := s.Passwords.Encrypt(clientSecret)
	if err != nil {
		return nil, err
	}
	return s.Store.CreateOIDC(ctx, &domain.OIDCProvider{
		OrgID: orgID, Name: name, Issuer: issuer, ClientID: clientID,
		ClientSecretEnc: secretEnc, Scopes: scopes, Enabled: true,
	})
}

func (s *Settings) ListOIDC(ctx context.Context, orgID uuid.UUID) ([]domain.OIDCProvider, error) {
	return s.Store.ListOIDCByOrg(ctx, orgID)
}

func (s *Settings) DeleteOIDC(ctx context.Context, id, orgID uuid.UUID) error {
	return s.Store.DeleteOIDC(ctx, id, orgID)
}

func (s *Settings) PublicOIDC(ctx context.Context) ([]domain.OIDCProvider, error) {
	return s.Store.ListEnabledOIDC(ctx)
}

func (s *Settings) OIDCAuthURL(ctx context.Context, id uuid.UUID) (string, error) {
	provider, err := s.Store.GetOIDC(ctx, id)
	if err != nil {
		return "", err
	}
	if !provider.Enabled {
		return "", domain.ErrForbidden
	}
	return s.OIDC.AuthURL(ctx, provider.Issuer, provider.ClientID, provider.Scopes, provider.ID.String())
}

func (s *Settings) OIDCCallback(ctx context.Context, id uuid.UUID, code string) (*domain.OIDCUser, error) {
	provider, err := s.Store.GetOIDC(ctx, id)
	if err != nil {
		return nil, err
	}
	if !provider.Enabled {
		return nil, domain.ErrForbidden
	}
	secret, err := s.Passwords.Decrypt(provider.ClientSecretEnc)
	if err != nil {
		return nil, err
	}
	return s.OIDC.Exchange(ctx, provider.Issuer, provider.ClientID, secret, provider.ID.String(), code)
}
