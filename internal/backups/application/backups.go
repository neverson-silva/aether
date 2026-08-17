package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"aether/internal/backups/domain"
	databasedomain "aether/internal/databases/domain"
)

type Backups struct {
	Store     domain.Store
	Databases DatabaseStore
}

type DatabaseStore interface {
	GetDatabase(ctx context.Context, id uuid.UUID) (*databasedomain.Database, error)
}

func (b *Backups) CreateState(ctx context.Context, orgID uuid.UUID) (*domain.Backup, error) {
	return b.Store.CreateBackup(ctx, &domain.Backup{
		OrgID: orgID, Kind: "state", Dest: "local",
		Path: fmt.Sprintf("state/%s/%d.tar", orgID, time.Now().Unix()),
	})
}

func (b *Backups) CreateDatabase(ctx context.Context, dbID, orgID uuid.UUID) (*domain.Backup, error) {
	db, err := b.Databases.GetDatabase(ctx, dbID)
	if err != nil {
		return nil, err
	}
	if db.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	return b.Store.CreateBackup(ctx, &domain.Backup{
		OrgID: orgID, DatabaseID: &db.ID, Kind: "db", Dest: "local",
		Path: fmt.Sprintf("db-backups/%s/%s-%d.dump", orgID, db.Name, time.Now().Unix()),
	})
}

func (b *Backups) List(ctx context.Context, orgID uuid.UUID, limit int) ([]domain.Backup, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return b.Store.ListByOrg(ctx, orgID, limit)
}

func (b *Backups) RestoreState(ctx context.Context, backupID, orgID uuid.UUID) error {
	backup, err := b.Store.GetBackup(ctx, backupID)
	if err != nil {
		return err
	}
	if backup.OrgID != orgID {
		return domain.ErrNotFound
	}
	return nil
}

func (b *Backups) RestoreDatabase(ctx context.Context, dbID, orgID, backupID uuid.UUID) error {
	backup, err := b.Store.GetBackup(ctx, backupID)
	if err != nil {
		return err
	}
	if backup.OrgID != orgID {
		return domain.ErrNotFound
	}
	db, err := b.Databases.GetDatabase(ctx, dbID)
	if err != nil {
		return err
	}
	if backup.DatabaseID == nil || *backup.DatabaseID != db.ID {
		return domain.ErrValidation
	}
	return nil
}
