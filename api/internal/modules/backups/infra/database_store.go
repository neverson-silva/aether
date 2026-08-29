package infra

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"aether/internal/modules/backups/domain"
	"aether/internal/platform/druntime/queue"
	gen "aether/internal/platform/infrastructure/pg/gen"
)

type DatabaseStore struct {
	q  gen.Querier
	db *sql.DB
}

func NewDatabaseStore(pool *pgxpool.Pool) *DatabaseStore {
	db := stdlib.OpenDBFromPool(pool)
	return &DatabaseStore{q: gen.New(db), db: db}
}

func (s *DatabaseStore) Close() error { return s.db.Close() }

func (s *DatabaseStore) GetConfiguration(ctx context.Context, id uuid.UUID) (*domain.BackupConfiguration, error) {
	row, err := s.q.GetBackupConfiguration(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	cfg := configFromRow(row)
	cfg.ServiceID = s.serviceID(ctx, "backup_configurations", id)
	return cfg, nil
}

func (s *DatabaseStore) ListConfigurationsByDatabase(ctx context.Context, databaseID uuid.UUID) ([]domain.BackupConfiguration, error) {
	rows, err := s.q.ListBackupConfigurationsByDatabase(ctx, databaseID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.BackupConfiguration, 0, len(rows))
	for _, row := range rows {
		cfg := configFromRow(row)
		cfg.ServiceID = s.serviceID(ctx, "backup_configurations", cfg.ID)
		out = append(out, *cfg)
	}
	return out, nil
}

func (s *DatabaseStore) CreateConfiguration(ctx context.Context, cfg *domain.BackupConfiguration) (*domain.BackupConfiguration, error) {
	row, err := s.q.CreateBackupConfiguration(ctx, gen.CreateBackupConfigurationParams{
		DatabaseID:     cfg.DatabaseID,
		Enabled:        cfg.Enabled,
		DestinationID:  cfg.DestinationID,
		PathPrefix:     cfg.PathPrefix,
		ScheduleType:   string(cfg.Schedule.Type),
		ScheduleMinute: int32(cfg.Schedule.Minute),
		ScheduleAt:     cfg.Schedule.At,
		ScheduleDay:    cfg.Schedule.DayOfWeek,
		ScheduleStart:  cfg.Schedule.StartDate,
		ScheduleCron:   cfg.Schedule.Cron,
		Timezone:       cfg.Schedule.Timezone,
		RetentionType:  string(cfg.Retention.Type),
		NextRunAt:      nullTime(cfg.NextRunAt),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	savedCfg := configFromRow(row)
	savedCfg.ServiceID = s.serviceID(ctx, "backup_configurations", savedCfg.ID)
	return savedCfg, nil
}

func (s *DatabaseStore) UpdateConfiguration(ctx context.Context, cfg *domain.BackupConfiguration) (*domain.BackupConfiguration, error) {
	row, err := s.q.UpdateBackupConfiguration(ctx, gen.UpdateBackupConfigurationParams{
		ID: cfg.ID, Enabled: cfg.Enabled, DestinationID: cfg.DestinationID, PathPrefix: cfg.PathPrefix,
		ScheduleType: string(cfg.Schedule.Type), ScheduleMinute: int32(cfg.Schedule.Minute), ScheduleAt: cfg.Schedule.At,
		ScheduleDay: cfg.Schedule.DayOfWeek, ScheduleStart: cfg.Schedule.StartDate, ScheduleCron: cfg.Schedule.Cron,
		Timezone: cfg.Schedule.Timezone, RetentionType: string(cfg.Retention.Type), NextRunAt: nullTime(cfg.NextRunAt),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	savedCfg := configFromRow(row)
	savedCfg.ServiceID = s.serviceID(ctx, "backup_configurations", savedCfg.ID)
	return savedCfg, nil
}

func (s *DatabaseStore) DeleteConfiguration(ctx context.Context, id uuid.UUID) error {
	return mapErr(s.q.DeleteBackupConfiguration(ctx, id))
}

func (s *DatabaseStore) ListEnabledConfigurations(ctx context.Context) ([]domain.BackupConfiguration, error) {
	rows, err := s.q.ListEnabledBackupConfigurations(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.BackupConfiguration, 0, len(rows))
	for _, r := range rows {
		cfg := configFromEnabledRow(r)
		cfg.ServiceID = s.serviceID(ctx, "backup_configurations", cfg.ID)
		out = append(out, cfg)
	}
	return out, nil
}

func (s *DatabaseStore) SetConfigurationNextRun(ctx context.Context, id uuid.UUID, next *time.Time) error {
	return mapErr(s.q.SetBackupConfigurationNextRun(ctx, gen.SetBackupConfigurationNextRunParams{ID: id, NextRunAt: nullTime(next)}))
}

func (s *DatabaseStore) CreateJob(ctx context.Context, job *domain.BackupJob) (*domain.BackupJob, error) {
	row, err := s.q.CreateBackupJob(ctx, gen.CreateBackupJobParams{
		DatabaseID:      job.DatabaseID,
		ConfigurationID: nullUUID(job.Configuration),
		TriggerType:     string(job.Trigger),
		Status:          string(job.Status),
		Engine:          job.Engine,
		EngineVersion:   job.EngineVersion,
		Format:          job.Format,
		DestinationID:   nullUUIDP(job.DestinationID),
		StorageKey:      job.StorageKey,
		SizeBytes:       job.SizeBytes,
		Checksum:        job.Checksum,
		ErrorCode:       job.ErrorCode,
		ErrorMessage:    job.ErrorMessage,
		StartedAt:       nullTime(job.StartedAt),
		CompletedAt:     nullTime(job.CompletedAt),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	savedJob := jobFromRow(row)
	savedJob.ServiceID = s.serviceID(ctx, "backup_jobs", savedJob.ID)
	return savedJob, nil
}

func (s *DatabaseStore) GetJob(ctx context.Context, id uuid.UUID) (*domain.BackupJob, error) {
	row, err := s.q.GetBackupJob(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	savedJob := jobFromRow(row)
	savedJob.ServiceID = s.serviceID(ctx, "backup_jobs", savedJob.ID)
	return savedJob, nil
}

func (s *DatabaseStore) UpdateJob(ctx context.Context, job *domain.BackupJob) (*domain.BackupJob, error) {
	row, err := s.q.UpdateBackupJob(ctx, gen.UpdateBackupJobParams{
		ID:            job.ID,
		Status:        string(job.Status),
		Engine:        job.Engine,
		EngineVersion: job.EngineVersion,
		Format:        job.Format,
		StorageKey:    job.StorageKey,
		SizeBytes:     job.SizeBytes,
		Checksum:      job.Checksum,
		ErrorCode:     job.ErrorCode,
		ErrorMessage:  job.ErrorMessage,
		StartedAt:     nullTime(job.StartedAt),
		CompletedAt:   nullTime(job.CompletedAt),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	savedJob := jobFromRow(row)
	savedJob.ServiceID = s.serviceID(ctx, "backup_jobs", savedJob.ID)
	return savedJob, nil
}

func (s *DatabaseStore) ListJobsByDatabase(ctx context.Context, databaseID uuid.UUID, limit int) ([]domain.BackupJob, error) {
	rows, err := s.q.ListBackupJobsByDatabase(ctx, gen.ListBackupJobsByDatabaseParams{ID: databaseID, Limit: int32(limit)})
	if err != nil {
		return nil, mapErr(err)
	}
	out := jobsFromRows(rows)
	for i := range out {
		out[i].ServiceID = s.serviceID(ctx, "backup_jobs", out[i].ID)
	}
	return out, nil
}

func (s *DatabaseStore) ListActiveJobsByDatabase(ctx context.Context, databaseID uuid.UUID) ([]domain.BackupJob, error) {
	rows, err := s.q.ListActiveBackupJobsByDatabase(ctx, databaseID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := jobsFromRows(rows)
	for i := range out {
		out[i].ServiceID = s.serviceID(ctx, "backup_jobs", out[i].ID)
	}
	return out, nil
}

func (s *DatabaseStore) ListQueuedJobs(ctx context.Context, limit int) ([]domain.BackupJob, error) {
	rows, err := s.q.ListBackupJobsDue(ctx, int32(limit))
	if err != nil {
		return nil, mapErr(err)
	}
	out := jobsFromRows(rows)
	for i := range out {
		out[i].ServiceID = s.serviceID(ctx, "backup_jobs", out[i].ID)
	}
	return out, nil
}

func (s *DatabaseStore) RecoverInterrupted(ctx context.Context, startedAt time.Time) ([]queue.Job, error) {
	if err := s.q.RecoverInterruptedBackupJobs(ctx, sql.NullTime{Time: startedAt, Valid: true}); err != nil {
		return nil, mapErr(err)
	}
	if err := s.q.RecoverInterruptedRestoreJobs(ctx, sql.NullTime{Time: startedAt, Valid: true}); err != nil {
		return nil, mapErr(err)
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT bj.id, d.org_id, 'backup'
		FROM backup_jobs bj JOIN databases d ON d.id = bj.database_id
		WHERE bj.status = 'queued'
		UNION ALL
		SELECT rj.id, d.org_id, 'restore'
		FROM restore_jobs rj JOIN databases d ON d.id = rj.target_database_id
		WHERE rj.status = 'queued'`)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	jobs := make([]queue.Job, 0)
	for rows.Next() {
		var id, orgID uuid.UUID
		var jobType string
		if err := rows.Scan(&id, &orgID, &jobType); err != nil {
			return nil, mapErr(err)
		}
		jobs = append(jobs, queue.Job{ID: id.String(), Type: jobType, OrgID: orgID.String(), Payload: []byte(id.String())})
	}
	return jobs, mapErr(rows.Err())
}

func (s *DatabaseStore) CreateRestoreJob(ctx context.Context, job *domain.RestoreJob) (*domain.RestoreJob, error) {
	row, err := s.q.CreateRestoreJob(ctx, gen.CreateRestoreJobParams{
		BackupID:         nullUUID(job.BackupID),
		TargetDatabaseID: job.TargetDatabaseID,
		Status:           string(job.Status),
		ErrorCode:        job.ErrorCode,
		ErrorMessage:     job.ErrorMessage,
		StartedAt:        nullTime(job.StartedAt),
		CompletedAt:      nullTime(job.CompletedAt),
		SourceType:       string(job.SourceType),
		SourceFilename:   job.SourceFilename,
		SourceSize:       job.SourceSize,
		SourceChecksum:   job.SourceChecksum,
		SourceFormat:     job.SourceFormat,
		UploadedBytes:    job.UploadedBytes,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	savedJob := restoreFromRow(row)
	savedJob.ServiceID = s.serviceID(ctx, "restore_jobs", savedJob.ID)
	return savedJob, nil
}

func (s *DatabaseStore) GetRestoreJob(ctx context.Context, id uuid.UUID) (*domain.RestoreJob, error) {
	row, err := s.q.GetRestoreJob(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	savedJob := restoreFromRow(row)
	savedJob.ServiceID = s.serviceID(ctx, "restore_jobs", savedJob.ID)
	return savedJob, nil
}

func (s *DatabaseStore) UpdateRestoreJob(ctx context.Context, job *domain.RestoreJob) (*domain.RestoreJob, error) {
	row, err := s.q.UpdateRestoreJob(ctx, gen.UpdateRestoreJobParams{
		ID:             job.ID,
		Status:         string(job.Status),
		ErrorCode:      job.ErrorCode,
		ErrorMessage:   job.ErrorMessage,
		StartedAt:      nullTime(job.StartedAt),
		CompletedAt:    nullTime(job.CompletedAt),
		SourceFilename: job.SourceFilename,
		SourceSize:     job.SourceSize,
		SourceChecksum: job.SourceChecksum,
		SourceFormat:   job.SourceFormat,
		UploadedBytes:  job.UploadedBytes,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	savedJob := restoreFromRow(row)
	savedJob.ServiceID = s.serviceID(ctx, "restore_jobs", savedJob.ID)
	return savedJob, nil
}

func (s *DatabaseStore) FailAbandonedUploads(ctx context.Context, olderThan time.Time) error {
	return mapErr(s.q.FailAbandonedUploadRestores(ctx, olderThan))
}

func (s *DatabaseStore) ListRestoreJobsByTarget(ctx context.Context, targetID uuid.UUID, limit int) ([]domain.RestoreJob, error) {
	rows, err := s.q.ListRestoreJobsByTarget(ctx, gen.ListRestoreJobsByTargetParams{ID: targetID, Limit: int32(limit)})
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.RestoreJob, 0, len(rows))
	for _, r := range rows {
		job := restoreFromRow(r)
		job.ServiceID = s.serviceID(ctx, "restore_jobs", job.ID)
		out = append(out, *job)
	}
	return out, nil
}

func (s *DatabaseStore) ListQueuedRestoreJobs(ctx context.Context, limit int) ([]domain.RestoreJob, error) {
	rows, err := s.q.ListRestoreJobsDue(ctx, int32(limit))
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.RestoreJob, 0, len(rows))
	for _, r := range rows {
		job := restoreFromRow(r)
		job.ServiceID = s.serviceID(ctx, "restore_jobs", job.ID)
		out = append(out, *job)
	}
	return out, nil
}

func (s *DatabaseStore) serviceID(ctx context.Context, table string, id uuid.UUID) uuid.UUID {
	query := "SELECT service_id FROM " + table + " WHERE id = $1"
	var serviceID uuid.UUID
	if err := s.db.QueryRowContext(ctx, query, id).Scan(&serviceID); err != nil {
		return uuid.Nil
	}
	return serviceID
}

func configFromEnabledRow(r gen.ListEnabledBackupConfigurationsRow) domain.BackupConfiguration {
	var next *time.Time
	if r.NextRunAt.Valid {
		t := r.NextRunAt.Time
		next = &t
	}
	return domain.BackupConfiguration{
		ID: r.ID, DatabaseID: r.DatabaseID, OrgID: r.OrgID, Enabled: r.Enabled, DestinationID: r.DestinationID,
		PathPrefix: r.PathPrefix,
		Schedule: domain.Schedule{
			Type: domain.ScheduleType(r.ScheduleType), Minute: int(r.ScheduleMinute), At: r.ScheduleAt,
			DayOfWeek: r.ScheduleDay, StartDate: r.ScheduleStart, Cron: r.ScheduleCron, Timezone: r.Timezone,
		},
		Retention: domain.Retention{Type: domain.RetentionType(r.RetentionType)},
		NextRunAt: next, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func configFromRow(r gen.BackupConfiguration) *domain.BackupConfiguration {
	var next *time.Time
	if r.NextRunAt.Valid {
		t := r.NextRunAt.Time
		next = &t
	}
	return &domain.BackupConfiguration{
		ID: r.ID, DatabaseID: r.DatabaseID, Enabled: r.Enabled, DestinationID: r.DestinationID,
		PathPrefix: r.PathPrefix,
		Schedule: domain.Schedule{
			Type: domain.ScheduleType(r.ScheduleType), Minute: int(r.ScheduleMinute), At: r.ScheduleAt,
			DayOfWeek: r.ScheduleDay, StartDate: r.ScheduleStart, Cron: r.ScheduleCron, Timezone: r.Timezone,
		},
		Retention: domain.Retention{Type: domain.RetentionType(r.RetentionType)},
		NextRunAt: next, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func jobFromRow(r gen.BackupJob) *domain.BackupJob {
	return &domain.BackupJob{
		ID: r.ID, DatabaseID: r.DatabaseID, Configuration: uuidPtr(r.ConfigurationID),
		Trigger: domain.TriggerType(r.TriggerType), Status: domain.BackupStatus(r.Status),
		Engine: r.Engine, EngineVersion: r.EngineVersion, Format: r.Format,
		DestinationID: r.DestinationID.UUID, StorageKey: r.StorageKey, SizeBytes: r.SizeBytes,
		Checksum: r.Checksum, ErrorCode: r.ErrorCode, ErrorMessage: r.ErrorMessage,
		StartedAt: timePtr(r.StartedAt), CompletedAt: timePtr(r.CompletedAt), CreatedAt: r.CreatedAt,
	}
}

func jobsFromRows(rows []gen.BackupJob) []domain.BackupJob {
	out := make([]domain.BackupJob, 0, len(rows))
	for _, r := range rows {
		out = append(out, *jobFromRow(r))
	}
	return out
}

func restoreFromRow(r gen.RestoreJob) *domain.RestoreJob {
	return &domain.RestoreJob{
		ID: r.ID, BackupID: uuidFromNull(r.BackupID), TargetDatabaseID: r.TargetDatabaseID,
		Status: domain.RestoreStatus(r.Status), ErrorCode: r.ErrorCode, ErrorMessage: r.ErrorMessage,
		StartedAt: timePtr(r.StartedAt), CompletedAt: timePtr(r.CompletedAt), CreatedAt: r.CreatedAt,
		SourceType: domain.RestoreSourceType(r.SourceType), SourceFilename: r.SourceFilename,
		SourceSize: r.SourceSize, SourceChecksum: r.SourceChecksum, SourceFormat: r.SourceFormat,
		UploadedBytes: r.UploadedBytes,
	}
}

func uuidFromNull(v uuid.NullUUID) *uuid.UUID {
	if !v.Valid {
		return nil
	}
	return &v.UUID
}

func nullTime(v *time.Time) sql.NullTime {
	if v == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *v, Valid: true}
}

func timePtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}

func nullUUIDP(v uuid.UUID) uuid.NullUUID {
	if v == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: v, Valid: true}
}
