package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type DatabaseBackupStore interface {
	GetConfiguration(ctx context.Context, databaseID uuid.UUID) (*BackupConfiguration, error)
	UpsertConfiguration(ctx context.Context, cfg *BackupConfiguration) (*BackupConfiguration, error)
	DeleteConfiguration(ctx context.Context, databaseID uuid.UUID) error
	ListEnabledConfigurations(ctx context.Context) ([]BackupConfiguration, error)
	SetConfigurationNextRun(ctx context.Context, id uuid.UUID, next *time.Time) error

	CreateJob(ctx context.Context, job *BackupJob) (*BackupJob, error)
	GetJob(ctx context.Context, id uuid.UUID) (*BackupJob, error)
	UpdateJob(ctx context.Context, job *BackupJob) (*BackupJob, error)
	ListJobsByDatabase(ctx context.Context, databaseID uuid.UUID, limit int) ([]BackupJob, error)
	ListActiveJobsByDatabase(ctx context.Context, databaseID uuid.UUID) ([]BackupJob, error)
	ListQueuedJobs(ctx context.Context, limit int) ([]BackupJob, error)

	CreateRestoreJob(ctx context.Context, job *RestoreJob) (*RestoreJob, error)
	GetRestoreJob(ctx context.Context, id uuid.UUID) (*RestoreJob, error)
	UpdateRestoreJob(ctx context.Context, job *RestoreJob) (*RestoreJob, error)
	ListRestoreJobsByTarget(ctx context.Context, targetID uuid.UUID, limit int) ([]RestoreJob, error)
	ListQueuedRestoreJobs(ctx context.Context, limit int) ([]RestoreJob, error)
}