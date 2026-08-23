package ports

import (
	"context"
	"time"

	"aether/internal/modules/sourcecontrol/domain"
	"github.com/google/uuid"
)

type SourceStore interface {
	ListByRepository(ctx context.Context, provider domain.Provider, installationID, repositoryID string) ([]domain.ServiceSource, error)
	GetByService(ctx context.Context, serviceID, organizationID uuid.UUID) (*domain.ServiceSource, error)
	Upsert(ctx context.Context, source *domain.ServiceSource) (*domain.ServiceSource, error)
	DeleteByService(ctx context.Context, serviceID, organizationID uuid.UUID) error
}

type ConnectionStore interface {
	UpsertConnection(ctx context.Context, connection *domain.Connection) (*domain.Connection, error)
	ListConnections(ctx context.Context, organizationID uuid.UUID, provider domain.Provider) ([]domain.Connection, error)
	GetConnectionByInstallation(ctx context.Context, provider domain.Provider, installationID string) (*domain.Connection, error)
	GetConnection(ctx context.Context, id uuid.UUID) (*domain.Connection, error)
	AttachInstallation(ctx context.Context, id uuid.UUID, installationID, accountID, accountName string) (*domain.Connection, error)
	CreateManifestState(ctx context.Context, state string, organizationID, userID uuid.UUID, returnURL string, expiresAt time.Time) error
	ConsumeManifestState(ctx context.Context, state string) (uuid.UUID, uuid.UUID, string, error)
}

type SecretCipher interface {
	Encrypt(plain string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type InstallationProvider interface {
	GetInstallation(ctx context.Context, installationID string) (domain.Installation, error)
	ListRepositories(ctx context.Context, installationID string) ([]domain.Repository, error)
	GetBranches(ctx context.Context, repositoryID string) ([]domain.Branch, error)
	GetFile(ctx context.Context, repositoryID, path, ref string) (string, error)
}

type DeliveryStore interface {
	Claim(ctx context.Context, delivery domain.WebhookDelivery) (*domain.WebhookDelivery, bool, error)
	Complete(ctx context.Context, id uuid.UUID, status, message string) error
}

type DeploymentTrigger interface {
	Deploy(ctx context.Context, appID, orgID uuid.UUID, trigger, commit string) (any, error)
}

type ChangedFilesResolver interface {
	GetChangedFiles(ctx context.Context, repositoryID, before, after string) ([]string, error)
}

type TemplateFileReader interface {
	GetServiceFile(ctx context.Context, connectionID uuid.UUID, repositoryID, path, ref string) (string, error)
}
