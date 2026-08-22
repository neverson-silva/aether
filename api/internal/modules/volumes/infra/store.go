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

	"aether/internal/modules/volumes/domain"
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

func (s *Store) GetVolumeByApp(ctx context.Context, appID uuid.UUID, name string) (*domain.Volume, error) {
	row, err := s.q.GetVolumeByApp(ctx, gen.GetVolumeByAppParams{AppID: appID, Name: name})
	if err != nil {
		return nil, mapErr(err)
	}
	return volumeFromRow(row), nil
}

func (s *Store) ListVolumesByApp(ctx context.Context, appID uuid.UUID) ([]domain.Volume, error) {
	rows, err := s.q.ListVolumesByApp(ctx, appID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Volume, 0, len(rows))
	for _, r := range rows {
		out = append(out, *volumeFromRow(r))
	}
	return out, nil
}

func (s *Store) CreateVolume(ctx context.Context, volume *domain.Volume) (*domain.Volume, error) {
	row, err := s.q.CreateVolume(ctx, gen.CreateVolumeParams{
		AppID: volume.AppID, Name: volume.Name, MountPath: volume.MountPath,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return volumeFromRow(row), nil
}

func (s *Store) CreateBackup(ctx context.Context, backup *domain.Backup) (*domain.Backup, error) {
	row, err := s.q.CreateBackup(ctx, gen.CreateBackupParams{
		OrgID: backup.OrgID, AppID: nullUUID(backup.AppID), Path: backup.Path,
		Size: backup.Size, Kind: backup.Kind, Dest: backup.Dest,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return &domain.Backup{
		ID: row.ID, OrgID: row.OrgID, AppID: uuidPtr(row.AppID), Path: row.Path,
		Size: row.Size, Kind: row.Kind, Dest: row.Dest, CreatedAt: row.CreatedAt,
	}, nil
}

func volumeFromRow(row gen.AppVolume) *domain.Volume {
	return &domain.Volume{
		ID: row.ID, AppID: row.AppID, Name: row.Name, MountPath: row.MountPath,
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
