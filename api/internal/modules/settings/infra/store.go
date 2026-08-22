package infra

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"aether/internal/modules/settings/domain"
	gen "aether/internal/platform/infrastructure/pg/gen"
)

type Store struct {
	q  gen.Querier
	db *sql.DB
}

func NewStore(pool *pgxpool.Pool) *Store {
	db := stdlib.OpenDBFromPool(pool)
	return &Store{q: gen.New(db), db: db}
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) GetBranding(ctx context.Context, orgID uuid.UUID) (*domain.Branding, error) {
	row, err := s.q.GetBranding(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	return brandingFromRow(row), nil
}

func (s *Store) SaveBranding(ctx context.Context, branding *domain.Branding) (*domain.Branding, error) {
	row, err := s.q.UpsertBranding(ctx, gen.UpsertBrandingParams{
		OrgID: branding.OrgID, Name: branding.Name, LogoUrl: branding.LogoURL,
		PrimaryColor: branding.PrimaryColor, AccentColor: branding.AccentColor, DarkMode: branding.DarkMode,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return brandingFromRow(row), nil
}

func (s *Store) CreateS3(ctx context.Context, dest *domain.S3Destination) (*domain.S3Destination, error) {
	row, err := s.q.CreateS3Destination(ctx, gen.CreateS3DestinationParams{
		OrgID: dest.OrgID, Name: dest.Name, Type: string(dest.Type), Endpoint: dest.Endpoint,
		Bucket: dest.Bucket, Region: dest.Region, AccountID: dest.AccountID,
		AccessKeyEnc: dest.AccessKeyEnc, SecretKeyEnc: dest.SecretKeyEnc,
		GoogleClientID: dest.GoogleClientID, GoogleClientSecretEnc: dest.GoogleClientSecretEnc,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return s3Model(row.ID, row.OrgID, row.Name, row.Type, row.Endpoint, row.Bucket, row.Region, row.AccountID,
		row.AccessKeyEnc, row.SecretKeyEnc, row.OauthStatus, row.OauthEmail, row.AccessTokenEnc, row.RefreshTokenEnc,
		row.GoogleClientID, row.GoogleClientSecretEnc, row.CreatedAt, row.UpdatedAt), nil
}

func (s *Store) ListS3ByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.S3Destination, error) {
	rows, err := s.q.ListS3Destinations(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.S3Destination, 0, len(rows))
	for _, r := range rows {
		out = append(out, *s3Model(r.ID, r.OrgID, r.Name, r.Type, r.Endpoint, r.Bucket, r.Region, r.AccountID,
			r.AccessKeyEnc, r.SecretKeyEnc, r.OauthStatus, r.OauthEmail, r.AccessTokenEnc, r.RefreshTokenEnc,
			r.GoogleClientID, r.GoogleClientSecretEnc, r.CreatedAt, r.UpdatedAt))
	}
	return out, nil
}

func (s *Store) GetS3(ctx context.Context, id, orgID uuid.UUID) (*domain.S3Destination, error) {
	row, err := s.q.GetS3Destination(ctx, gen.GetS3DestinationParams{ID: id, OrgID: orgID})
	if err != nil {
		return nil, mapErr(err)
	}
	return s3Model(row.ID, row.OrgID, row.Name, row.Type, row.Endpoint, row.Bucket, row.Region, row.AccountID,
		row.AccessKeyEnc, row.SecretKeyEnc, row.OauthStatus, row.OauthEmail, row.AccessTokenEnc, row.RefreshTokenEnc,
		row.GoogleClientID, row.GoogleClientSecretEnc, row.CreatedAt, row.UpdatedAt), nil
}

func (s *Store) UpdateS3(ctx context.Context, dest *domain.S3Destination) (*domain.S3Destination, error) {
	row, err := s.q.UpdateS3Destination(ctx, gen.UpdateS3DestinationParams{
		ID: dest.ID, OrgID: dest.OrgID, Name: dest.Name, Type: string(dest.Type),
		Endpoint: dest.Endpoint, Bucket: dest.Bucket, Region: dest.Region, AccountID: dest.AccountID,
		AccessKeyEnc: dest.AccessKeyEnc, SecretKeyEnc: dest.SecretKeyEnc,
		GoogleClientID: dest.GoogleClientID, GoogleClientSecretEnc: dest.GoogleClientSecretEnc,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return s3Model(row.ID, row.OrgID, row.Name, row.Type, row.Endpoint, row.Bucket, row.Region, row.AccountID,
		row.AccessKeyEnc, row.SecretKeyEnc, row.OauthStatus, row.OauthEmail, row.AccessTokenEnc, row.RefreshTokenEnc,
		row.GoogleClientID, row.GoogleClientSecretEnc, row.CreatedAt, row.UpdatedAt), nil
}

func (s *Store) UpdateS3OAuth(ctx context.Context, id, orgID uuid.UUID, status domain.OAuthStatus, email, accessTokenEnc, refreshTokenEnc string) error {
	return mapErr(s.q.UpdateS3DestinationOAuth(ctx, gen.UpdateS3DestinationOAuthParams{
		ID: id, OrgID: orgID, OauthStatus: string(status), OauthEmail: email,
		AccessTokenEnc: accessTokenEnc, RefreshTokenEnc: refreshTokenEnc,
	}))
}

func (s *Store) DeleteS3(ctx context.Context, id, orgID uuid.UUID) error {
	return mapErr(s.q.DeleteS3Destination(ctx, gen.DeleteS3DestinationParams{ID: id, OrgID: orgID}))
}

func (s *Store) CreateOIDC(ctx context.Context, provider *domain.OIDCProvider) (*domain.OIDCProvider, error) {
	row, err := s.q.CreateOIDCProvider(ctx, gen.CreateOIDCProviderParams{
		OrgID: provider.OrgID, Name: provider.Name, Issuer: provider.Issuer,
		ClientID: provider.ClientID, ClientSecretEnc: provider.ClientSecretEnc,
		Scopes: provider.Scopes, Enabled: provider.Enabled,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return oidcFromRow(row), nil
}

func (s *Store) ListOIDCByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.OIDCProvider, error) {
	rows, err := s.q.ListOIDCProviders(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.OIDCProvider, 0, len(rows))
	for _, r := range rows {
		out = append(out, *oidcFromRow(r))
	}
	return out, nil
}

func (s *Store) ListEnabledOIDC(ctx context.Context) ([]domain.OIDCProvider, error) {
	rows, err := s.q.ListEnabledOIDCProviders(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.OIDCProvider, 0, len(rows))
	for _, r := range rows {
		out = append(out, *oidcFromRow(r))
	}
	return out, nil
}

func (s *Store) CountEnabledOIDC(ctx context.Context) (int, error) {
	count, err := s.q.CountEnabledOIDCProviders(ctx)
	if err != nil {
		return 0, mapErr(err)
	}
	return int(count), nil
}

func (s *Store) GetOIDC(ctx context.Context, id uuid.UUID) (*domain.OIDCProvider, error) {
	row, err := s.q.GetOIDCProvider(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return oidcFromRow(row), nil
}

func (s *Store) DeleteOIDC(ctx context.Context, id, orgID uuid.UUID) error {
	return mapErr(s.q.DeleteOIDCProvider(ctx, gen.DeleteOIDCProviderParams{ID: id, OrgID: orgID}))
}

func brandingFromRow(row gen.Branding) *domain.Branding {
	return &domain.Branding{
		OrgID: row.OrgID, Name: row.Name, LogoURL: row.LogoUrl, PrimaryColor: row.PrimaryColor,
		AccentColor: row.AccentColor, DarkMode: row.DarkMode, UpdatedAt: row.UpdatedAt,
	}
}

func s3FromRow(row gen.S3Destination) *domain.S3Destination {
	return &domain.S3Destination{
		ID: row.ID, OrgID: row.OrgID, Name: row.Name,
		Type: domain.DestinationType(row.Type), Endpoint: row.Endpoint,
		Bucket: row.Bucket, Region: row.Region, AccountID: row.AccountID,
		AccessKeyEnc: row.AccessKeyEnc, SecretKeyEnc: row.SecretKeyEnc,
		OAuthStatus: domain.OAuthStatus(row.OauthStatus), OAuthEmail: row.OauthEmail,
		AccessTokenEnc: row.AccessTokenEnc, RefreshTokenEnc: row.RefreshTokenEnc,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func s3Model(id, orgID uuid.UUID, name, typ, endpoint, bucket, region, accountID, ak, sk, status, email, atEnc, rtEnc, gClientID, gSecretEnc string, createdAt, updatedAt time.Time) *domain.S3Destination {
	return &domain.S3Destination{
		ID: id, OrgID: orgID, Name: name, Type: domain.DestinationType(typ),
		Endpoint: endpoint, Bucket: bucket, Region: region, AccountID: accountID,
		AccessKeyEnc: ak, SecretKeyEnc: sk, OAuthStatus: domain.OAuthStatus(status),
		OAuthEmail: email, AccessTokenEnc: atEnc, RefreshTokenEnc: rtEnc,
		GoogleClientID: gClientID, GoogleClientSecretEnc: gSecretEnc,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

func oidcFromRow(row gen.OidcProvider) *domain.OIDCProvider {
	return &domain.OIDCProvider{
		ID: row.ID, OrgID: row.OrgID, Name: row.Name, Issuer: row.Issuer,
		ClientID: row.ClientID, ClientSecretEnc: row.ClientSecretEnc, Scopes: row.Scopes,
		Enabled: row.Enabled, CreatedAt: row.CreatedAt,
	}
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return domain.ErrConflict
		case "23503":
			return domain.ErrConflict
		case "23502", "22P02", "23514":
			return domain.ErrValidation
		}
	}
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}
