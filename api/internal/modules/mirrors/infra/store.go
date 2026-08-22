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

	"aether/internal/modules/mirrors/domain"
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

func (s *Store) CreateMirror(ctx context.Context, mirror *domain.Mirror) (*domain.Mirror, error) {
	row, err := s.q.CreateMirror(ctx, gen.CreateMirrorParams{
		Name: mirror.Name, Source: mirror.Source, Dest: mirror.Dest,
		DestTlsVerify: mirror.DestTLSVerify, TagsFilter: mirror.TagsFilter, Schedule: mirror.Schedule,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return mirrorFromRow(row), nil
}

func (s *Store) GetMirror(ctx context.Context, id uuid.UUID) (*domain.Mirror, error) {
	row, err := s.q.GetMirror(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return mirrorFromRow(row), nil
}

func (s *Store) ListMirrors(ctx context.Context) ([]domain.Mirror, error) {
	rows, err := s.q.ListMirrors(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Mirror, 0, len(rows))
	for _, r := range rows {
		out = append(out, *mirrorFromRow(r))
	}
	return out, nil
}

func (s *Store) SetStatus(ctx context.Context, id uuid.UUID, status string) error {
	return mapErr(s.q.SetMirrorStatus(ctx, gen.SetMirrorStatusParams{ID: id, Status: status}))
}

func (s *Store) DeleteMirror(ctx context.Context, id uuid.UUID) error {
	return mapErr(s.q.DeleteMirror(ctx, id))
}

func mirrorFromRow(row gen.RegistryMirror) *domain.Mirror {
	return &domain.Mirror{
		ID: row.ID, Name: row.Name, Source: row.Source, Dest: row.Dest,
		DestTLSVerify: row.DestTlsVerify, TagsFilter: row.TagsFilter, Schedule: row.Schedule,
		LastRun: nullTimePtr(row.LastRun), Status: row.Status, CreatedAt: row.CreatedAt,
	}
}

func nullTimePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
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
