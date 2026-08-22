package infra

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"aether/internal/modules/sourcecontrol/domain"
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

func (s *Store) ListByRepository(ctx context.Context, provider domain.Provider, installationID, repositoryID string) ([]domain.ServiceSource, error) {
	rows, err := s.q.ListServiceSourcesByRepository(ctx, gen.ListServiceSourcesByRepositoryParams{
		Provider: string(provider), InstallationID: installationID, RepositoryID: repositoryID,
	})
	if err != nil {
		return nil, err
	}
	result := make([]domain.ServiceSource, 0, len(rows))
	for _, row := range rows {
		result = append(result, sourceFromListRow(row))
	}
	return result, nil
}

func (s *Store) GetByService(ctx context.Context, serviceID, organizationID uuid.UUID) (*domain.ServiceSource, error) {
	row, err := s.q.GetServiceSource(ctx, gen.GetServiceSourceParams{ServiceID: serviceID, OrgID: organizationID})
	if err != nil {
		return nil, err
	}
	return &domain.ServiceSource{
		ID: row.ID, ServiceID: row.ServiceID, ConnectionID: row.ConnectionID, OrganizationID: row.OrganizationID,
		RepositoryID: row.RepositoryID, RepositoryOwner: row.RepositoryOwner, RepositoryName: row.RepositoryName,
		RepositoryFullName: row.RepositoryFullName, DefaultBranch: row.DefaultBranch, Branch: row.Branch,
		AutoDeploy: row.AutoDeploy, RootDirectory: row.RootDirectory, WatchPaths: row.WatchPaths,
		IgnorePaths: row.IgnorePaths, WatchRootFiles: row.WatchRootFiles, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (s *Store) Upsert(ctx context.Context, source *domain.ServiceSource) (*domain.ServiceSource, error) {
	row, err := s.q.UpsertServiceSource(ctx, gen.UpsertServiceSourceParams{
		ServiceID: source.ServiceID, ConnectionID: source.ConnectionID, RepositoryID: source.RepositoryID,
		RepositoryOwner: source.RepositoryOwner, RepositoryName: source.RepositoryName,
		RepositoryFullName: source.RepositoryFullName, DefaultBranch: source.DefaultBranch, Branch: source.Branch,
		AutoDeploy: source.AutoDeploy, RootDirectory: source.RootDirectory, WatchPaths: source.WatchPaths,
		IgnorePaths: source.IgnorePaths, WatchRootFiles: source.WatchRootFiles, OrgID: source.OrganizationID,
	})
	if err != nil {
		return nil, err
	}
	return &domain.ServiceSource{
		ID: row.ID, ServiceID: row.ServiceID, ConnectionID: row.ConnectionID, OrganizationID: source.OrganizationID,
		RepositoryID: row.RepositoryID, RepositoryOwner: row.RepositoryOwner, RepositoryName: row.RepositoryName,
		RepositoryFullName: row.RepositoryFullName, DefaultBranch: row.DefaultBranch, Branch: row.Branch,
		AutoDeploy: row.AutoDeploy, RootDirectory: row.RootDirectory, WatchPaths: row.WatchPaths,
		IgnorePaths: row.IgnorePaths, WatchRootFiles: row.WatchRootFiles, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (s *Store) DeleteByService(ctx context.Context, serviceID, organizationID uuid.UUID) error {
	return s.q.DeleteServiceSource(ctx, gen.DeleteServiceSourceParams{ServiceID: serviceID, OrgID: organizationID})
}

func (s *Store) Claim(ctx context.Context, delivery domain.WebhookDelivery) (*domain.WebhookDelivery, bool, error) {
	row, err := s.q.ClaimSCMWebhookDelivery(ctx, gen.ClaimSCMWebhookDeliveryParams{
		Provider: string(delivery.Provider), DeliveryID: delivery.DeliveryID,
		EventType: delivery.EventType, InstallationID: delivery.InstallationID,
		RepositoryID: delivery.RepositoryID,
	})
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	claimed := deliveryFromRow(row)
	return &claimed, true, nil
}

func (s *Store) Complete(ctx context.Context, id uuid.UUID, status, message string) error {
	return s.q.CompleteSCMWebhookDelivery(ctx, gen.CompleteSCMWebhookDeliveryParams{ID: id, Status: status, Error: message})
}

func sourceFromListRow(row gen.ListServiceSourcesByRepositoryRow) domain.ServiceSource {
	return domain.ServiceSource{
		ID: row.ID, ServiceID: row.ServiceID, ConnectionID: row.ConnectionID, OrganizationID: row.OrganizationID,
		RepositoryID: row.RepositoryID, RepositoryOwner: row.RepositoryOwner,
		RepositoryName: row.RepositoryName, RepositoryFullName: row.RepositoryFullName,
		DefaultBranch: row.DefaultBranch, Branch: row.Branch, AutoDeploy: row.AutoDeploy,
		RootDirectory: row.RootDirectory, WatchPaths: row.WatchPaths,
		IgnorePaths: row.IgnorePaths, WatchRootFiles: row.WatchRootFiles,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func deliveryFromRow(row gen.ScmWebhookDelivery) domain.WebhookDelivery {
	var processedAt *time.Time
	if row.ProcessedAt.Valid {
		processedAt = &row.ProcessedAt.Time
	}
	return domain.WebhookDelivery{
		ID: row.ID, Provider: domain.Provider(row.Provider), DeliveryID: row.DeliveryID,
		EventType: row.EventType, InstallationID: row.InstallationID,
		RepositoryID: row.RepositoryID, Status: row.Status, Error: row.Error,
		ReceivedAt: row.ReceivedAt, ProcessedAt: processedAt,
	}
}
