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

	"aether/internal/modules/snapshots/domain"
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

func (s *Store) CreateSnapshot(ctx context.Context, snapshot *domain.Snapshot) (*domain.Snapshot, error) {
	row, err := s.q.CreateSnapshot(ctx, gen.CreateSnapshotParams{
		OrgID: snapshot.OrgID, AppID: nullUUID(snapshot.AppID), Volume: snapshot.Volume,
		Name: snapshot.Name, Size: snapshot.Size, Chunks: int32(snapshot.Chunks),
		DedupSaved: snapshot.DedupSaved,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return snapshotFromRow(row), nil
}

func (s *Store) GetSnapshot(ctx context.Context, id uuid.UUID) (*domain.Snapshot, error) {
	row, err := s.q.GetSnapshot(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return snapshotFromRow(row), nil
}

func (s *Store) CreateSnapshotForService(ctx context.Context, snapshot *domain.Snapshot) (*domain.Snapshot, error) {
	row, err := s.q.CreateSnapshotForService(ctx, gen.CreateSnapshotForServiceParams{OrgID: snapshot.OrgID, ServiceID: nullUUID(snapshot.ServiceID), Volume: snapshot.Volume, Name: snapshot.Name, Size: snapshot.Size, Chunks: int32(snapshot.Chunks), DedupSaved: snapshot.DedupSaved})
	if err != nil {
		return nil, mapErr(err)
	}
	return snapshotFromRow(row), nil
}

func (s *Store) ListSnapshotsByService(ctx context.Context, orgID, serviceID uuid.UUID, limit int) ([]domain.Snapshot, error) {
	rows, err := s.q.ListSnapshotsByService(ctx, gen.ListSnapshotsByServiceParams{OrgID: orgID, ServiceID: nullUUID(&serviceID), Limit: int32(limit)})
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Snapshot, 0, len(rows))
	for _, row := range rows {
		out = append(out, *snapshotFromRow(row))
	}
	return out, nil
}

func (s *Store) ListSnapshotsByOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]domain.Snapshot, error) {
	rows, err := s.q.ListSnapshotsByOrg(ctx, gen.ListSnapshotsByOrgParams{OrgID: orgID, Limit: int32(limit)})
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Snapshot, 0, len(rows))
	for _, r := range rows {
		out = append(out, *snapshotFromRow(r))
	}
	return out, nil
}

func (s *Store) DeleteSnapshot(ctx context.Context, id, orgID uuid.UUID) error {
	return mapErr(s.q.DeleteSnapshot(ctx, gen.DeleteSnapshotParams{ID: id, OrgID: orgID}))
}

func (s *Store) CreateSchedule(ctx context.Context, schedule *domain.Schedule) (*domain.Schedule, error) {
	row, err := s.q.CreateSnapshotSchedule(ctx, gen.CreateSnapshotScheduleParams{
		OrgID: schedule.OrgID, AppID: nullUUID(schedule.AppID), Volume: schedule.Volume,
		NamePrefix: schedule.NamePrefix, Cron: schedule.Cron, Retention: int32(schedule.Retention),
		Enabled: schedule.Enabled,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return scheduleFromRow(row), nil
}

func (s *Store) GetSchedule(ctx context.Context, id uuid.UUID) (*domain.Schedule, error) {
	row, err := s.q.GetSnapshotSchedule(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return scheduleFromRow(row), nil
}

func (s *Store) CreateScheduleForService(ctx context.Context, schedule *domain.Schedule) (*domain.Schedule, error) {
	row, err := s.q.CreateSnapshotScheduleForService(ctx, gen.CreateSnapshotScheduleForServiceParams{OrgID: schedule.OrgID, ServiceID: nullUUID(schedule.ServiceID), Volume: schedule.Volume, NamePrefix: schedule.NamePrefix, Cron: schedule.Cron, Retention: int32(schedule.Retention), Enabled: schedule.Enabled})
	if err != nil {
		return nil, mapErr(err)
	}
	return scheduleFromRow(row), nil
}

func (s *Store) ListSchedulesByService(ctx context.Context, orgID, serviceID uuid.UUID) ([]domain.Schedule, error) {
	rows, err := s.q.ListSchedulesByService(ctx, gen.ListSchedulesByServiceParams{OrgID: orgID, ServiceID: nullUUID(&serviceID)})
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Schedule, 0, len(rows))
	for _, row := range rows {
		out = append(out, *scheduleFromRow(row))
	}
	return out, nil
}

func (s *Store) ListSchedulesByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.Schedule, error) {
	rows, err := s.q.ListSchedulesByOrg(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Schedule, 0, len(rows))
	for _, r := range rows {
		out = append(out, *scheduleFromRow(r))
	}
	return out, nil
}

func (s *Store) DeleteSchedule(ctx context.Context, id, orgID uuid.UUID) error {
	return mapErr(s.q.DeleteSnapshotSchedule(ctx, gen.DeleteSnapshotScheduleParams{ID: id, OrgID: orgID}))
}

func (s *Store) ListEnabledSchedules(ctx context.Context) ([]domain.Schedule, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, org_id, app_id, service_id, volume, name_prefix, cron, retention, enabled, last_run, next_run, created_at FROM snapshot_schedules WHERE enabled ORDER BY next_run NULLS FIRST, created_at`)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out := make([]domain.Schedule, 0)
	for rows.Next() {
		var schedule domain.Schedule
		if err := rows.Scan(&schedule.ID, &schedule.OrgID, &schedule.AppID, &schedule.ServiceID, &schedule.Volume, &schedule.NamePrefix, &schedule.Cron, &schedule.Retention, &schedule.Enabled, &schedule.LastRun, &schedule.NextRun, &schedule.CreatedAt); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, schedule)
	}
	return out, mapErr(rows.Err())
}

func (s *Store) SetScheduleRun(ctx context.Context, id uuid.UUID, last, next time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE snapshot_schedules SET last_run = $2, next_run = $3 WHERE id = $1`, id, last, next)
	return mapErr(err)
}

func snapshotFromRow(row gen.Snapshot) *domain.Snapshot {
	return &domain.Snapshot{
		ID: row.ID, OrgID: row.OrgID, AppID: uuidPtr(row.AppID), Volume: row.Volume,
		ServiceID: uuidPtr(row.ServiceID),
		Name:      row.Name, Size: row.Size, Chunks: int(row.Chunks), DedupSaved: row.DedupSaved,
		CreatedAt: row.CreatedAt,
	}
}

func scheduleFromRow(row gen.SnapshotSchedule) *domain.Schedule {
	return &domain.Schedule{
		ID: row.ID, OrgID: row.OrgID, AppID: uuidPtr(row.AppID), Volume: row.Volume,
		ServiceID:  uuidPtr(row.ServiceID),
		NamePrefix: row.NamePrefix, Cron: row.Cron, Retention: int(row.Retention),
		Enabled: row.Enabled, LastRun: nullTimePtr(row.LastRun), NextRun: nullTimePtr(row.NextRun),
		CreatedAt: row.CreatedAt,
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
