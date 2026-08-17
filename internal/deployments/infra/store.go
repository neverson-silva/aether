package infra

import (
	"bytes"
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
	"github.com/sqlc-dev/pqtype"

	"aether/internal/deployments/domain"
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

func (s *Store) CreateDeployment(ctx context.Context, dep *domain.Deployment) (*domain.Deployment, error) {
	snapshot := dep.EnvSnapshot
	if len(snapshot) == 0 {
		snapshot = []byte("{}")
	}
	var spec pqtype.NullRawMessage
	if len(dep.DeploySpec) > 0 {
		spec = pqtype.NullRawMessage{RawMessage: dep.DeploySpec, Valid: true}
	}
	row, err := s.q.CreateDeployment(ctx, gen.CreateDeploymentParams{
		AppID: dep.AppID, Number: int32(dep.Number), Status: string(dep.Status),
		Trigger: dep.Trigger, TriggeredBy: dep.TriggeredBy, CommitSha: dep.CommitSHA,
		ImageRef: dep.ImageRef, ServerID: dep.ServerID, Error: dep.Error,
		EnvSnapshot: snapshot, ComposeYaml: dep.ComposeYAML, DeploySpec: spec,
		ComposeHash: dep.ComposeHash,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return deploymentFromRow(row), nil
}

func (s *Store) GetDeployment(ctx context.Context, id uuid.UUID) (*domain.Deployment, error) {
	row, err := s.q.GetDeployment(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return deploymentFromRow(row), nil
}

func (s *Store) GetByApp(ctx context.Context, appID uuid.UUID, number int) (*domain.Deployment, error) {
	row, err := s.q.GetDeploymentByApp(ctx, gen.GetDeploymentByAppParams{AppID: appID, Number: int32(number)})
	if err != nil {
		return nil, mapErr(err)
	}
	return deploymentFromRow(row), nil
}

func (s *Store) ListByApp(ctx context.Context, appID uuid.UUID, limit int) ([]domain.Deployment, error) {
	rows, err := s.q.ListDeployments(ctx, gen.ListDeploymentsParams{AppID: appID, Limit: int32(limit)})
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Deployment, 0, len(rows))
	for _, r := range rows {
		out = append(out, *deploymentFromRow(r))
	}
	return out, nil
}

func (s *Store) LatestByApps(ctx context.Context, appIDs []uuid.UUID) (map[uuid.UUID]domain.Deployment, error) {
	out := make(map[uuid.UUID]domain.Deployment, len(appIDs))
	if len(appIDs) == 0 {
		return out, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT ON (app_id) id, app_id, number, status, trigger, triggered_by, commit_sha, image_ref,
    container_id, server_id, error, env_snapshot, compose_yaml, deploy_spec, compose_hash,
    created_at, started_at, finished_at
FROM deployments
WHERE app_id = ANY($1)
ORDER BY app_id, number DESC`, appIDs)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	for rows.Next() {
		var i gen.Deployment
		if err := rows.Scan(
			&i.ID, &i.AppID, &i.Number, &i.Status, &i.Trigger, &i.TriggeredBy,
			&i.CommitSha, &i.ImageRef, &i.ContainerID, &i.ServerID, &i.Error,
			&i.EnvSnapshot, &i.ComposeYaml, &i.DeploySpec, &i.ComposeHash,
			&i.CreatedAt, &i.StartedAt, &i.FinishedAt,
		); err != nil {
			return nil, mapErr(err)
		}
		dep := deploymentFromRow(i)
		out[dep.AppID] = *dep
	}
	if err := rows.Err(); err != nil {
		return nil, mapErr(err)
	}
	return out, nil
}

func (s *Store) GetDeploymentCompose(ctx context.Context, depID uuid.UUID) (string, error) {
	compose, err := s.q.GetDeploymentCompose(ctx, depID)
	if err != nil {
		return "", mapErr(err)
	}
	return compose, nil
}

func (s *Store) ListQueued(ctx context.Context) ([]domain.Deployment, error) {
	rows, err := s.q.ListQueuedDeployments(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Deployment, 0, len(rows))
	for _, r := range rows {
		out = append(out, *deploymentFromRow(r))
	}
	return out, nil
}

func (s *Store) ListReady(ctx context.Context) ([]domain.Deployment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, app_id, number, status, trigger, triggered_by, commit_sha, image_ref,
    container_id, server_id, error, env_snapshot, compose_yaml, deploy_spec, compose_hash,
    created_at, started_at, finished_at
FROM deployments
WHERE status = 'ready'
ORDER BY number DESC
LIMIT 500`)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out := make([]domain.Deployment, 0, 32)
	for rows.Next() {
		var i gen.Deployment
		if err := rows.Scan(
			&i.ID, &i.AppID, &i.Number, &i.Status, &i.Trigger, &i.TriggeredBy,
			&i.CommitSha, &i.ImageRef, &i.ContainerID, &i.ServerID, &i.Error,
			&i.EnvSnapshot, &i.ComposeYaml, &i.DeploySpec, &i.ComposeHash,
			&i.CreatedAt, &i.StartedAt, &i.FinishedAt,
		); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, *deploymentFromRow(i))
	}
	if err := rows.Err(); err != nil {
		return nil, mapErr(err)
	}
	return out, nil
}

func (s *Store) NextNumber(ctx context.Context, appID uuid.UUID) (int, error) {
	n, err := s.q.NextDeploymentNumber(ctx, appID)
	if err != nil {
		return 0, mapErr(err)
	}
	return int(n), nil
}

func (s *Store) LastReady(ctx context.Context, appID uuid.UUID) (*domain.Deployment, error) {
	row, err := s.q.LastReadyDeployment(ctx, appID)
	if err != nil {
		return nil, mapErr(err)
	}
	return deploymentFromRow(row), nil
}

func (s *Store) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.Status, errMsg, imageRef, containerID string, startedAt, finishedAt *time.Time) error {
	return mapErr(s.q.UpdateDeploymentStatus(ctx, gen.UpdateDeploymentStatusParams{
		ID: id, Status: string(status), Error: errMsg, ImageRef: imageRef,
		ContainerID: containerID, StartedAt: timePtr(startedAt), FinishedAt: timePtr(finishedAt),
	}))
}

func (s *Store) MarkRolledBack(ctx context.Context, id uuid.UUID) error {
	return mapErr(s.q.MarkDeploymentRolledBack(ctx, id))
}

func (s *Store) CreateRollback(ctx context.Context, newDep *domain.Deployment, rolledBackID uuid.UUID) (*domain.Deployment, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	q := gen.New(tx)
	snapshot := newDep.EnvSnapshot
	if len(snapshot) == 0 {
		snapshot = []byte("{}")
	}
	var spec pqtype.NullRawMessage
	if len(newDep.DeploySpec) > 0 {
		spec = pqtype.NullRawMessage{RawMessage: newDep.DeploySpec, Valid: true}
	}
	row, err := q.CreateDeployment(ctx, gen.CreateDeploymentParams{
		AppID: newDep.AppID, Number: int32(newDep.Number), Status: string(newDep.Status),
		Trigger: newDep.Trigger, TriggeredBy: newDep.TriggeredBy, CommitSha: newDep.CommitSHA,
		ImageRef: newDep.ImageRef, ServerID: newDep.ServerID, Error: newDep.Error,
		EnvSnapshot: snapshot, ComposeYaml: newDep.ComposeYAML, DeploySpec: spec,
		ComposeHash: newDep.ComposeHash,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	if err := q.MarkDeploymentRolledBack(ctx, rolledBackID); err != nil {
		return nil, mapErr(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return deploymentFromRow(row), nil
}

func deploymentFromRow(row gen.Deployment) *domain.Deployment {
	started := nullTimePtr(row.StartedAt)
	finished := nullTimePtr(row.FinishedAt)
	return &domain.Deployment{
		ID: row.ID, AppID: row.AppID, Number: int(row.Number), Status: domain.Status(row.Status),
		Trigger: row.Trigger, TriggeredBy: row.TriggeredBy, CommitSHA: row.CommitSha,
		ImageRef: row.ImageRef, ContainerID: row.ContainerID, ServerID: row.ServerID,
		Error: row.Error, EnvSnapshot: compactJSON(row.EnvSnapshot), ComposeYAML: row.ComposeYaml,
		DeploySpec: row.DeploySpec.RawMessage, ComposeHash: row.ComposeHash,
		CreatedAt: row.CreatedAt, StartedAt: started, FinishedAt: finished,
	}
}

func nullTimePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func timePtr(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func compactJSON(raw []byte) []byte {
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return raw
	}
	return buf.Bytes()
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
		case "23502", "22P02":
			return domain.ErrValidation
		}
	}
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}
