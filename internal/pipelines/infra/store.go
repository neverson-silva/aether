package infra

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	gen "aether/internal/infrastructure/pg/gen"
	"aether/internal/pipelines/domain"
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

func (s *Store) CreatePipeline(ctx context.Context, pipeline *domain.Pipeline) (*domain.Pipeline, error) {
	stages, _ := json.Marshal(pipeline.Stages)
	row, err := s.q.CreatePipeline(ctx, gen.CreatePipelineParams{
		OrgID: pipeline.OrgID, AppID: nullUUID(pipeline.AppID), Name: pipeline.Name,
		Trigger: pipeline.Trigger, Stages: stages, Enabled: pipeline.Enabled,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return pipelineFromRow(row), nil
}

func (s *Store) GetPipeline(ctx context.Context, id uuid.UUID) (*domain.Pipeline, error) {
	row, err := s.q.GetPipeline(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return pipelineFromRow(row), nil
}

func (s *Store) ListPipelinesByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.Pipeline, error) {
	rows, err := s.q.ListPipelinesByOrg(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Pipeline, 0, len(rows))
	for _, r := range rows {
		out = append(out, *pipelineFromRow(r))
	}
	return out, nil
}

func (s *Store) DeletePipeline(ctx context.Context, id, orgID uuid.UUID) error {
	return mapErr(s.q.DeletePipeline(ctx, gen.DeletePipelineParams{ID: id, OrgID: orgID}))
}

func (s *Store) CreateRun(ctx context.Context, run *domain.Run) (*domain.Run, error) {
	row, err := s.q.CreatePipelineRun(ctx, gen.CreatePipelineRunParams{
		PipelineID: run.PipelineID, Status: run.Status, Trigger: run.Trigger,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return runFromRow(row), nil
}

func (s *Store) FinishRun(ctx context.Context, id uuid.UUID, status, log string) error {
	return mapErr(s.q.FinishPipelineRun(ctx, gen.FinishPipelineRunParams{ID: id, Status: status, Log: log}))
}

func (s *Store) ListRuns(ctx context.Context, pipelineID uuid.UUID, limit int) ([]domain.Run, error) {
	rows, err := s.q.ListPipelineRuns(ctx, gen.ListPipelineRunsParams{PipelineID: pipelineID, Limit: int32(limit)})
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Run, 0, len(rows))
	for _, r := range rows {
		out = append(out, *runFromRow(r))
	}
	return out, nil
}

func pipelineFromRow(row gen.Pipeline) *domain.Pipeline {
	var stages []domain.Stage
	_ = json.Unmarshal(row.Stages, &stages)
	if stages == nil {
		stages = []domain.Stage{}
	}
	return &domain.Pipeline{
		ID: row.ID, OrgID: row.OrgID, AppID: uuidPtr(row.AppID), Name: row.Name,
		Trigger: row.Trigger, Stages: stages, Enabled: row.Enabled, CreatedAt: row.CreatedAt,
	}
}

func runFromRow(row gen.PipelineRun) *domain.Run {
	return &domain.Run{
		ID: row.ID, PipelineID: row.PipelineID, Status: row.Status, Trigger: row.Trigger,
		Log: row.Log, StartedAt: row.StartedAt, FinishedAt: nullTimePtr(row.FinishedAt),
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
