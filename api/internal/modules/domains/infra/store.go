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

	"aether/internal/modules/domains/domain"
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

func (s *Store) CreateDomain(ctx context.Context, d *domain.Domain) (*domain.Domain, error) {
	row, err := s.q.CreateDomain(ctx, gen.CreateDomainParams{
		AppID: d.AppID, ServerID: nullUUIDPtr(d.ServerID), ServiceType: d.ServiceType, Host: d.Host, Https: d.HTTPS,
		Path: d.Path, InternalPath: d.InternalPath, StripPath: d.StripPath,
		ContainerPort: int32(d.ContainerPort), Status: d.Status, CertStatus: d.CertStatus,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return domainFromRow(row), nil
}

func (s *Store) ListDomains(ctx context.Context, appID uuid.UUID) ([]domain.Domain, error) {
	rows, err := s.q.ListDomains(ctx, appID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Domain, 0, len(rows))
	for _, r := range rows {
		out = append(out, *domainFromRow(gen.CreateDomainRow(r)))
	}
	return out, nil
}

func (s *Store) GetDomainByHost(ctx context.Context, appID uuid.UUID, host string) (*domain.Domain, error) {
	row, err := s.q.GetDomainByHost(ctx, gen.GetDomainByHostParams{AppID: appID, Host: host})
	if err != nil {
		return nil, mapErr(err)
	}
	return domainFromRow(gen.CreateDomainRow(row)), nil
}

func (s *Store) UpdateDomainStatus(ctx context.Context, id, appID uuid.UUID, status, certStatus string) error {
	return mapErr(s.q.UpdateDomainStatus(ctx, gen.UpdateDomainStatusParams{ID: id, AppID: appID, Status: status, CertStatus: certStatus}))
}

func (s *Store) UpdateDomainFields(ctx context.Context, id, appID uuid.UUID, host string, https bool, path, internalPath string, stripPath bool, containerPort int) error {
	return mapErr(s.q.UpdateDomainFields(ctx, gen.UpdateDomainFieldsParams{
		ID: id, AppID: appID, Host: host, Https: https, Path: path, InternalPath: internalPath,
		StripPath: stripPath, ContainerPort: int32(containerPort),
	}))
}

func (s *Store) UpdateDomainProvision(ctx context.Context, id, appID uuid.UUID, status, certStatus, lastError string, nextRetryAt *time.Time, retryCount int) error {
	return mapErr(s.q.UpdateDomainProvision(ctx, gen.UpdateDomainProvisionParams{
		ID: id, AppID: appID, Status: status, CertStatus: certStatus, LastError: lastError,
		NextRetryAt: nullTime(nextRetryAt), RetryCount: int32(retryCount),
	}))
}

func (s *Store) ListProvisioningDomains(ctx context.Context, now time.Time, maxRetries int) ([]domain.Domain, error) {
	rows, err := s.q.ListProvisioningDomains(ctx, gen.ListProvisioningDomainsParams{RetryCount: int32(maxRetries), NextRetryAt: sql.NullTime{Time: now, Valid: true}})
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Domain, 0, len(rows))
	for _, r := range rows {
		out = append(out, *domainFromRow(gen.CreateDomainRow(r)))
	}
	return out, nil
}

func (s *Store) GetDomainByID(ctx context.Context, id uuid.UUID) (*domain.Domain, error) {
	row, err := s.q.GetDomainByID(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return domainFromRow(gen.CreateDomainRow(row)), nil
}

func (s *Store) DeleteDomain(ctx context.Context, id, appID uuid.UUID) error {
	return mapErr(s.q.DeleteDomain(ctx, gen.DeleteDomainParams{ID: id, AppID: appID}))
}

func (s *Store) CreatePreview(ctx context.Context, preview *domain.Preview) (*domain.Preview, error) {
	row, err := s.q.CreatePreview(ctx, gen.CreatePreviewParams{
		AppID: preview.AppID, Branch: preview.Branch,
		DeploymentID: nullUUID(preview.DeploymentID), Domain: preview.Domain,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return previewFromRow(row), nil
}

func (s *Store) ListPreviews(ctx context.Context, appID uuid.UUID) ([]domain.Preview, error) {
	rows, err := s.q.ListPreviews(ctx, appID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Preview, 0, len(rows))
	for _, r := range rows {
		out = append(out, *previewFromRow(r))
	}
	return out, nil
}

func (s *Store) GetPreview(ctx context.Context, id, appID uuid.UUID) (*domain.Preview, error) {
	row, err := s.q.GetPreview(ctx, gen.GetPreviewParams{ID: id, AppID: appID})
	if err != nil {
		return nil, mapErr(err)
	}
	return previewFromRow(row), nil
}

func (s *Store) GetPreviewByID(ctx context.Context, id uuid.UUID) (*domain.Preview, error) {
	row, err := s.q.GetPreviewByID(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return previewFromRow(row), nil
}

func (s *Store) UpdatePreviewResult(ctx context.Context, id, appID uuid.UUID, deploymentID *uuid.UUID, containerID, status string) error {
	return mapErr(s.q.UpdatePreviewResult(ctx, gen.UpdatePreviewResultParams{
		ID: id, AppID: appID, DeploymentID: nullUUID(deploymentID), ContainerID: containerID, Status: status,
	}))
}

func (s *Store) DeletePreview(ctx context.Context, id, appID uuid.UUID) error {
	return mapErr(s.q.DeletePreview(ctx, gen.DeletePreviewParams{ID: id, AppID: appID}))
}

func (s *Store) ListCertificatesByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.Certificate, error) {
	rows, err := s.q.ListCertificatesByOrg(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Certificate, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Certificate{
			DomainID: r.ID, AppID: r.AppID, AppName: r.AppName, Host: r.Host,
			HTTPS: r.Https, CertState: r.CertStatus, CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

func domainFromRow(row gen.CreateDomainRow) *domain.Domain {
	var serverID uuid.UUID
	if row.ServerID.Valid {
		serverID = row.ServerID.UUID
	}
	var nextRetryAt *time.Time
	if row.NextRetryAt.Valid {
		t := row.NextRetryAt.Time
		nextRetryAt = &t
	}
	return &domain.Domain{
		ID: row.ID, AppID: row.AppID, ServiceType: row.ServiceType, ServerID: serverID, Host: row.Host, HTTPS: row.Https,
		Path: row.Path, InternalPath: row.InternalPath, StripPath: row.StripPath,
		ContainerPort: int(row.ContainerPort), Status: row.Status, CertStatus: row.CertStatus,
		RetryCount: int(row.RetryCount), LastError: row.LastError, NextRetryAt: nextRetryAt,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func previewFromRow(row gen.Preview) *domain.Preview {
	return &domain.Preview{
		ID: row.ID, AppID: row.AppID, Branch: row.Branch,
		DeploymentID: uuidPtr(row.DeploymentID), ContainerID: row.ContainerID,
		Domain: row.Domain, Status: row.Status, CreatedAt: row.CreatedAt,
	}
}

func nullUUID(v *uuid.UUID) uuid.NullUUID {
	if v == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *v, Valid: true}
}

func nullUUIDPtr(id uuid.UUID) uuid.NullUUID {
	if id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}

func nullTime(v *time.Time) sql.NullTime {
	if v == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *v, Valid: true}
}

func uuidPtr(v uuid.NullUUID) *uuid.UUID {
	if !v.Valid {
		return nil
	}
	return &v.UUID
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
		case "23502", "22P02":
			return domain.ErrValidation
		}
	}
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}
