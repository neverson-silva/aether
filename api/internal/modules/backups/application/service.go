package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/backups/adapters/container"
	"aether/internal/modules/backups/domain"
	databasedomain "aether/internal/modules/databases/domain"
	"aether/internal/platform/druntime/cache"
	"aether/internal/platform/druntime/events"
	"aether/internal/platform/druntime/locks"
	"aether/internal/platform/druntime/queue"
	"aether/internal/platform/druntime/scheduler"
	"aether/internal/platform/storage"
)

type DatabaseBackups struct {
	Store        domain.DatabaseBackupStore
	Databases    DatabasesReader
	Passwords    PasswordCipher
	Destinations DestinationProvider
	Exec         container.Executor
	Queue        queue.Queue
	Scheduler    scheduler.Scheduler
	Locks        locks.LockManager
	Audit        AuditRecorder
	Notifier     BackupNotifier
	Outbox       interface {
		Enqueue(context.Context, events.Event, string) error
	}
	Cache             cache.Cache
	UploadRoot        string
	MaxUploadBytes    int64
	RestoreQuotaBytes int64
	MaxBackupBytes    int64
	Timeout           time.Duration
	Batch             int
}

type RestoreNotifier interface {
	NotifyRestore(ctx context.Context, orgID, targetDBID, restoreID uuid.UUID, status string)
}

type DatabasesReader interface {
	Get(ctx context.Context, id, orgID uuid.UUID) (*databasedomain.Database, error)
}

type PasswordCipher interface {
	Decrypt(ciphertext string) (string, error)
}

type DestinationProvider interface {
	GetProvider(ctx context.Context, destID, orgID uuid.UUID) (storage.Provider, error)
}

type AuditRecorder interface {
	Record(ctx context.Context, orgID uuid.UUID, action, resourceType, resourceID, details string)
}

type BackupNotifier interface {
	NotifyBackup(ctx context.Context, orgID, databaseID, backupID uuid.UUID, status string)
}

func (s *DatabaseBackups) timeout() time.Duration {
	if s.Timeout <= 0 {
		return 30 * time.Minute
	}
	return s.Timeout
}

func (s *DatabaseBackups) maxBackupBytes() int64 {
	if s.MaxBackupBytes <= 0 {
		return 8 << 30
	}
	return s.MaxBackupBytes
}

func (s *DatabaseBackups) ensureOrganizationCapacity(ctx context.Context, orgID uuid.UUID) error {
	counter, ok := s.Store.(domain.OrganizationJobCounter)
	if !ok {
		return nil
	}
	active, err := counter.CountActiveByOrg(ctx, orgID)
	if err != nil {
		return err
	}
	if active >= 8 {
		return domain.ErrConflict
	}
	return nil
}

func (s *DatabaseBackups) ensureUploadCapacity(ctx context.Context, orgID uuid.UUID, additional int64) error {
	counter, ok := s.Store.(domain.OrganizationJobCounter)
	if !ok {
		return nil
	}
	used, err := counter.CountActiveUploadBytesByOrg(ctx, orgID)
	if err != nil {
		return err
	}
	quota := s.RestoreQuotaBytes
	if quota <= 0 {
		quota = 4 << 30
	}
	if additional < 0 || used > quota-additional {
		return domain.ErrConflict
	}
	return nil
}

func (s *DatabaseBackups) batch() int {
	if s.Batch <= 0 {
		return 20
	}
	return s.Batch
}

func (s *DatabaseBackups) notifyRestore(ctx context.Context, orgID, targetDBID, restoreID uuid.UUID, status string) {
	if notifier, ok := s.Notifier.(RestoreNotifier); ok {
		notifier.NotifyRestore(ctx, orgID, targetDBID, restoreID, status)
	}
}
