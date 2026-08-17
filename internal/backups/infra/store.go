package infra

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"aether/internal/backups/domain"
	gen "aether/internal/infrastructure/pg/gen"
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

func (s *Store) CreateBackup(ctx context.Context, backup *domain.Backup) (*domain.Backup, error) {
	row, err := s.q.CreateBackup(ctx, gen.CreateBackupParams{
		OrgID: backup.OrgID, DatabaseID: nullUUID(backup.DatabaseID), AppID: nullUUID(backup.AppID), Path: backup.Path,
		Size: backup.Size, Kind: backup.Kind, Dest: backup.Dest,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return backupFromRow(row), nil
}

func (s *Store) GetBackup(ctx context.Context, id uuid.UUID) (*domain.Backup, error) {
	row, err := s.q.GetBackup(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return backupFromRow(row), nil
}

func (s *Store) ListByOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]domain.Backup, error) {
	rows, err := s.q.ListBackupsByOrg(ctx, gen.ListBackupsByOrgParams{OrgID: orgID, Limit: int32(limit)})
	if err != nil {
		return nil, mapErr(err)
	}
	return backupsFromRows(rows), nil
}

func (s *Store) ListByDatabase(ctx context.Context, databaseID uuid.UUID, limit int) ([]domain.Backup, error) {
	rows, err := s.q.ListBackupsByDatabase(ctx, gen.ListBackupsByDatabaseParams{DatabaseID: nullUUID(&databaseID), Limit: int32(limit)})
	if err != nil {
		return nil, mapErr(err)
	}
	return backupsFromRows(rows), nil
}

func backupsFromRows(rows []gen.Backup) []domain.Backup {
	out := make([]domain.Backup, 0, len(rows))
	for _, r := range rows {
		out = append(out, *backupFromRow(r))
	}
	return out
}

func backupFromRow(row gen.Backup) *domain.Backup {
	return &domain.Backup{
		ID: row.ID, OrgID: row.OrgID, DatabaseID: uuidPtr(row.DatabaseID),
		Path: row.Path, Size: row.Size, Kind: row.Kind, Dest: row.Dest, CreatedAt: row.CreatedAt,
	}
}

func nullUUID(v *uuid.UUID) uuid.NullUUID {
	if v == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *v, Valid: true}
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
		case "23502", "22P02", "23514":
			return domain.ErrValidation
		}
	}
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}
