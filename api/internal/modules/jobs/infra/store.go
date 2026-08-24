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

	"aether/internal/modules/jobs/domain"
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

func (s *Store) CreateCronJob(ctx context.Context, job *domain.CronJob) (*domain.CronJob, error) {
	row, err := s.q.CreateCronJob(ctx, gen.CreateCronJobParams{
		AppID: job.AppID, Name: job.Name, Schedule: job.Schedule, Command: job.Command, Enabled: job.Enabled,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return cronFromRow(row), nil
}

func (s *Store) GetCronJob(ctx context.Context, id uuid.UUID) (*domain.CronJob, error) {
	row, err := s.q.GetCronJob(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return cronFromRow(row), nil
}

func (s *Store) ListCronJobsByApp(ctx context.Context, appID uuid.UUID) ([]domain.CronJob, error) {
	rows, err := s.q.ListCronJobsByApp(ctx, appID)
	if err != nil {
		return nil, mapErr(err)
	}
	return cronsFromRows(rows), nil
}

func (s *Store) ListCronJobsByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.CronJob, error) {
	rows, err := s.q.ListCronJobsByOrg(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	return cronsFromRows(rows), nil
}

func (s *Store) ListEnabledCronJobs(ctx context.Context) ([]domain.CronJob, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT c.id, a.org_id, c.app_id, c.name, c.schedule, c.command, c.enabled, c.last_run, c.next_run, c.created_at FROM cron_jobs c JOIN apps a ON a.id = c.app_id WHERE c.enabled ORDER BY c.next_run NULLS FIRST, c.created_at`)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	jobs := make([]domain.CronJob, 0)
	for rows.Next() {
		var job domain.CronJob
		if err := rows.Scan(&job.ID, &job.OrgID, &job.AppID, &job.Name, &job.Schedule, &job.Command, &job.Enabled, &job.LastRun, &job.NextRun, &job.CreatedAt); err != nil {
			return nil, mapErr(err)
		}
		jobs = append(jobs, job)
	}
	return jobs, mapErr(rows.Err())
}

func (s *Store) SetCronRun(ctx context.Context, id uuid.UUID, last, next time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE cron_jobs SET last_run = $2, next_run = $3 WHERE id = $1`, id, last, next)
	return mapErr(err)
}

func (s *Store) UpdateCronJob(ctx context.Context, job *domain.CronJob) (*domain.CronJob, error) {
	row, err := s.q.UpdateCronJob(ctx, gen.UpdateCronJobParams{
		ID: job.ID, AppID: job.AppID, Schedule: job.Schedule, Command: job.Command, Enabled: job.Enabled,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return cronFromRow(row), nil
}

func (s *Store) DeleteCronJob(ctx context.Context, id uuid.UUID) error {
	return mapErr(s.q.DeleteCronJob(ctx, id))
}

func (s *Store) CreateWorker(ctx context.Context, worker *domain.Worker) (*domain.Worker, error) {
	row, err := s.q.CreateWorker(ctx, gen.CreateWorkerParams{
		AppID: worker.AppID, Name: worker.Name, Command: worker.Command,
		Replicas: int32(worker.Replicas), Enabled: worker.Enabled,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return workerFromRow(row), nil
}

func (s *Store) GetWorker(ctx context.Context, id uuid.UUID) (*domain.Worker, error) {
	row, err := s.q.GetWorker(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return workerFromRow(row), nil
}

func (s *Store) ListWorkersByApp(ctx context.Context, appID uuid.UUID) ([]domain.Worker, error) {
	rows, err := s.q.ListWorkersByApp(ctx, appID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Worker, 0, len(rows))
	for _, r := range rows {
		out = append(out, *workerFromRow(r))
	}
	return out, nil
}

func (s *Store) SetWorkerState(ctx context.Context, id, appID uuid.UUID, status, containerID string) error {
	return mapErr(s.q.SetWorkerState(ctx, gen.SetWorkerStateParams{ID: id, AppID: appID, Status: status, ContainerID: containerID}))
}

func (s *Store) DeleteWorker(ctx context.Context, id uuid.UUID) error {
	return mapErr(s.q.DeleteWorker(ctx, id))
}

func (s *Store) GetPolicy(ctx context.Context, appID uuid.UUID) (*domain.Policy, error) {
	row, err := s.q.GetAppPolicy(ctx, appID)
	if err != nil {
		return nil, mapErr(err)
	}
	return policyFromRow(row), nil
}

func (s *Store) SavePolicy(ctx context.Context, policy *domain.Policy) (*domain.Policy, error) {
	row, err := s.q.UpsertAppPolicy(ctx, gen.UpsertAppPolicyParams{
		AppID: policy.AppID, Enabled: policy.Enabled, CpuMin: float32(policy.CPUMin),
		CpuMax: float32(policy.CPUMax), MemMinMb: int32(policy.MemMinMB), MemMaxMb: int32(policy.MemMaxMB),
		ScaleUpPct: int32(policy.ScaleUpPct), ScaleDownPct: int32(policy.ScaleDownPct), CooldownMin: int32(policy.CooldownMin),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return policyFromRow(row), nil
}

func (s *Store) CreateAutopilotEvent(ctx context.Context, appID uuid.UUID, action, detail string) error {
	return mapErr(s.q.CreateAutopilotEvent(ctx, gen.CreateAutopilotEventParams{AppID: appID, Action: action, Detail: detail}))
}

func (s *Store) ListAutopilotEvents(ctx context.Context, appID uuid.UUID, limit int) ([]domain.AutopilotEvent, error) {
	rows, err := s.q.ListAutopilotEvents(ctx, gen.ListAutopilotEventsParams{AppID: appID, Limit: int32(limit)})
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.AutopilotEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.AutopilotEvent{
			ID: r.ID, AppID: r.AppID, Action: r.Action, Detail: r.Detail, CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

func cronsFromRows(rows []gen.CronJob) []domain.CronJob {
	out := make([]domain.CronJob, 0, len(rows))
	for _, r := range rows {
		out = append(out, *cronFromRow(r))
	}
	return out
}

func cronFromRow(row gen.CronJob) *domain.CronJob {
	return &domain.CronJob{
		ID: row.ID, AppID: row.AppID, Name: row.Name, Schedule: row.Schedule,
		Command: row.Command, Enabled: row.Enabled, LastRun: nullTimePtr(row.LastRun),
		NextRun: nullTimePtr(row.NextRun), CreatedAt: row.CreatedAt,
	}
}

func workerFromRow(row gen.Worker) *domain.Worker {
	return &domain.Worker{
		ID: row.ID, AppID: row.AppID, Name: row.Name, Command: row.Command,
		Replicas: int(row.Replicas), Enabled: row.Enabled, Status: row.Status,
		ContainerID: row.ContainerID, CreatedAt: row.CreatedAt,
	}
}

func policyFromRow(row gen.AppPolicy) *domain.Policy {
	return &domain.Policy{
		AppID: row.AppID, Enabled: row.Enabled, CPUMin: float64(row.CpuMin), CPUMax: float64(row.CpuMax),
		MemMinMB: int(row.MemMinMb), MemMaxMB: int(row.MemMaxMb), ScaleUpPct: int(row.ScaleUpPct),
		ScaleDownPct: int(row.ScaleDownPct), CooldownMin: int(row.CooldownMin), UpdatedAt: row.UpdatedAt,
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
