package infra

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/sourcecontrol/domain"
	gen "aether/internal/platform/infrastructure/pg/gen"
)

func (s *Store) UpsertConnection(ctx context.Context, connection *domain.Connection) (*domain.Connection, error) {
	row, err := s.q.UpsertSCMConnection(ctx, gen.UpsertSCMConnectionParams{
		OrganizationID: connection.OrganizationID, Provider: string(connection.Provider),
		ExternalAccountID: connection.ExternalAccountID, ExternalAccountName: connection.ExternalAccountName,
		InstallationID: connection.InstallationID, Status: connection.Status, Metadata: connection.Metadata,
		CredentialsEnc: connection.CredentialsEnc,
	})
	if err != nil {
		return nil, err
	}
	return connectionFromValues(row.ID, row.OrganizationID, row.Provider, row.ExternalAccountID, row.ExternalAccountName, row.InstallationID, row.Status, row.Metadata, row.CredentialsEnc, row.CreatedAt, row.UpdatedAt), nil
}

func (s *Store) GetConnectionByInstallation(ctx context.Context, provider domain.Provider, installationID string) (*domain.Connection, error) {
	row, err := s.q.GetSCMConnectionByInstallation(ctx, gen.GetSCMConnectionByInstallationParams{Provider: string(provider), InstallationID: installationID})
	if err != nil {
		return nil, err
	}
	return connectionFromValues(row.ID, row.OrganizationID, row.Provider, row.ExternalAccountID, row.ExternalAccountName, row.InstallationID, row.Status, row.Metadata, row.CredentialsEnc, row.CreatedAt, row.UpdatedAt), nil
}

func (s *Store) GetConnection(ctx context.Context, id uuid.UUID) (*domain.Connection, error) {
	row, err := s.q.GetSCMConnectionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return connectionFromValues(row.ID, row.OrganizationID, row.Provider, row.ExternalAccountID, row.ExternalAccountName, row.InstallationID, row.Status, row.Metadata, row.CredentialsEnc, row.CreatedAt, row.UpdatedAt), nil
}

func (s *Store) DeleteConnection(ctx context.Context, id, organizationID uuid.UUID) error {
	return s.q.DeleteSCMConnection(ctx, gen.DeleteSCMConnectionParams{ID: id, OrganizationID: organizationID})
}

func (s *Store) AttachInstallation(ctx context.Context, id uuid.UUID, installationID, accountID, accountName string) (*domain.Connection, error) {
	row, err := s.q.AttachSCMConnectionInstallation(ctx, gen.AttachSCMConnectionInstallationParams{
		ID: id, InstallationID: installationID, ExternalAccountID: accountID, ExternalAccountName: accountName,
	})
	if err != nil {
		return nil, err
	}
	return connectionFromValues(row.ID, row.OrganizationID, row.Provider, row.ExternalAccountID, row.ExternalAccountName, row.InstallationID, row.Status, row.Metadata, row.CredentialsEnc, row.CreatedAt, row.UpdatedAt), nil
}

func (s *Store) CreateManifestState(ctx context.Context, state string, organizationID, userID uuid.UUID, returnURL string, expiresAt time.Time) error {
	return s.q.CreateSCMManifestState(ctx, gen.CreateSCMManifestStateParams{State: state, OrganizationID: organizationID, UserID: userID, ReturnUrl: returnURL, ExpiresAt: expiresAt})
}

func (s *Store) ConsumeManifestState(ctx context.Context, state string) (uuid.UUID, uuid.UUID, string, error) {
	row, err := s.q.ConsumeSCMManifestState(ctx, state)
	if err == sql.ErrNoRows {
		return uuid.Nil, uuid.Nil, "", sql.ErrNoRows
	}
	if err != nil {
		return uuid.Nil, uuid.Nil, "", err
	}
	return row.OrganizationID, row.UserID, row.ReturnUrl, nil
}

func (s *Store) ListConnections(ctx context.Context, organizationID uuid.UUID, provider domain.Provider) ([]domain.Connection, error) {
	rows, err := s.q.ListSCMConnectionsByOrganization(ctx, gen.ListSCMConnectionsByOrganizationParams{
		OrganizationID: organizationID, Provider: string(provider),
	})
	if err != nil {
		return nil, err
	}
	result := make([]domain.Connection, 0, len(rows))
	for _, row := range rows {
		result = append(result, *connectionFromValues(row.ID, row.OrganizationID, row.Provider, row.ExternalAccountID, row.ExternalAccountName, row.InstallationID, row.Status, row.Metadata, row.CredentialsEnc, row.CreatedAt, row.UpdatedAt))
	}
	return result, nil
}

func connectionFromValues(id uuid.UUID, organizationID uuid.UUID, provider, externalAccountID, externalAccountName, installationID, status string, metadata []byte, credentialsEnc string, createdAt, updatedAt time.Time) *domain.Connection {
	return &domain.Connection{
		ID: id, OrganizationID: organizationID, Provider: domain.Provider(provider),
		ExternalAccountID: externalAccountID, ExternalAccountName: externalAccountName,
		InstallationID: installationID, Status: status, Metadata: metadata,
		CredentialsEnc: credentialsEnc, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}
