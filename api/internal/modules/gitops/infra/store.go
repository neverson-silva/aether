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

	"aether/internal/modules/gitops/domain"
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

func (s *Store) CreateGitOps(ctx context.Context, g *domain.GitOps) (*domain.GitOps, error) {
	row, err := s.q.CreateGitOps(ctx, gen.CreateGitOpsParams{
		OrgID: g.OrgID, Name: g.Name, RepoUrl: g.RepoURL, Branch: g.Branch,
		Path: g.Path, ApplyMode: g.ApplyMode,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return gitopsFromRow(row), nil
}

func (s *Store) GetGitOps(ctx context.Context, id uuid.UUID) (*domain.GitOps, error) {
	row, err := s.q.GetGitOps(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return gitopsFromRow(row), nil
}

func (s *Store) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.GitOps, error) {
	rows, err := s.q.ListGitOpsByOrg(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.GitOps, 0, len(rows))
	for _, r := range rows {
		out = append(out, *gitopsFromRow(r))
	}
	return out, nil
}

func (s *Store) UpdateSync(ctx context.Context, id uuid.UUID, result domain.SyncResult) error {
	return mapErr(s.q.UpdateGitOpsSync(ctx, gen.UpdateGitOpsSyncParams{
		ID: id, LastSha: result.SHA, LastStatus: "synced",
		DriftAdded: int32(result.Added), DriftChanged: int32(result.Changed), DriftRemoved: int32(result.Removed),
	}))
}

func (s *Store) DeleteGitOps(ctx context.Context, id, orgID uuid.UUID) error {
	return mapErr(s.q.DeleteGitOps(ctx, gen.DeleteGitOpsParams{ID: id, OrgID: orgID}))
}

func gitopsFromRow(row gen.Gitop) *domain.GitOps {
	var target *uuid.UUID
	if row.TargetOrgID.Valid {
		target = &row.TargetOrgID.UUID
	}
	return &domain.GitOps{
		ID: row.ID, OrgID: row.OrgID, Name: row.Name, RepoURL: row.RepoUrl,
		Branch: row.Branch, Path: row.Path, TargetOrgID: target, ApplyMode: row.ApplyMode,
		LastSHA: row.LastSha, LastStatus: row.LastStatus, DriftAdded: int(row.DriftAdded),
		DriftChanged: int(row.DriftChanged), DriftRemoved: int(row.DriftRemoved),
		LastSync: nullTimePtr(row.LastSync), CreatedAt: row.CreatedAt,
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
