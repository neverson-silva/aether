package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"aether/internal/backups/adapters/container"
	"aether/internal/backups/domain"
	databasedomain "aether/internal/databases/domain"
	"aether/internal/druntime/locks"
	"aether/internal/druntime/queue"
	"aether/internal/storage"
)

type DatabaseBackups struct {
	Store        domain.DatabaseBackupStore
	Databases    DatabasesReader
	Passwords    PasswordCipher
	Destinations DestinationProvider
	Exec         container.Executor
	Queue        queue.Queue
	Locks        locks.LockManager
	Audit        AuditRecorder
	Timeout      time.Duration
	Batch        int
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

func (s *DatabaseBackups) timeout() time.Duration {
	if s.Timeout <= 0 {
		return 30 * time.Minute
	}
	return s.Timeout
}

func (s *DatabaseBackups) batch() int {
	if s.Batch <= 0 {
		return 20
	}
	return s.Batch
}