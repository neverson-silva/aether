package application

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/backups/domain"
	"aether/internal/platform/druntime/events"
	"aether/internal/platform/druntime/queue"
)

func (s *DatabaseBackups) StartManualBackup(ctx context.Context, dbID, orgID uuid.UUID, configIDs ...uuid.UUID) (*domain.BackupJob, error) {
	db, err := s.Databases.Get(ctx, dbID, orgID)
	if err != nil {
		return nil, err
	}
	var cfg *domain.BackupConfiguration
	if len(configIDs) > 0 && configIDs[0] != uuid.Nil {
		cfg, err = s.Store.GetConfiguration(ctx, configIDs[0])
		if err != nil || cfg == nil || cfg.DatabaseID != dbID {
			return nil, domain.ErrValidation
		}
	} else {
		configs, listErr := s.Store.ListConfigurationsByDatabase(ctx, dbID)
		if listErr != nil || len(configs) != 1 {
			return nil, domain.ErrValidation
		}
		cfg = &configs[0]
	}
	active, err := s.Store.ListActiveJobsByDatabase(ctx, dbID)
	if err != nil {
		return nil, err
	}
	if len(active) > 0 {
		return nil, domain.ErrConflict
	}
	job, err := s.Store.CreateJob(ctx, &domain.BackupJob{
		DatabaseID: dbID, Configuration: &cfg.ID, DestinationID: cfg.DestinationID,
		Trigger: domain.TriggerManual, Status: domain.BackupQueued, Engine: string(db.Engine),
	})
	if err != nil {
		return nil, err
	}
	if err := s.enqueue(ctx, queue.Job{ID: job.ID.String(), Type: "backup", OrgID: orgID.String(), Payload: []byte(job.ID.String())}); err != nil {
		_ = s.failJob(ctx, orgID, job, "BACKUP_ENQUEUE_FAILED", err.Error())
		return nil, err
	}
	s.Audit.Record(ctx, orgID, "backup.started", "database", dbID.String(), job.ID.String())
	s.notifyBackup(ctx, orgID, dbID, job.ID, string(job.Status))
	return job, nil
}

func (s *DatabaseBackups) ListBackups(ctx context.Context, dbID, orgID uuid.UUID, limit int) ([]domain.BackupJob, error) {
	if _, err := s.Databases.Get(ctx, dbID, orgID); err != nil {
		return nil, err
	}
	return s.Store.ListJobsByDatabase(ctx, dbID, limit)
}

func (s *DatabaseBackups) GetBackup(ctx context.Context, backupID, orgID uuid.UUID) (*domain.BackupJob, error) {
	job, err := s.Store.GetJob(ctx, backupID)
	if err != nil {
		return nil, err
	}
	if _, err := s.Databases.Get(ctx, job.DatabaseID, orgID); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *DatabaseBackups) CancelBackup(ctx context.Context, backupID, orgID uuid.UUID) error {
	job, err := s.GetBackup(ctx, backupID, orgID)
	if err != nil {
		return err
	}
	switch job.Status {
	case domain.BackupQueued:
		if err := job.Transition(domain.BackupCancelled); err != nil {
			return err
		}
		_, err = s.Store.UpdateJob(ctx, job)
		if err == nil {
			s.notifyBackup(ctx, orgID, job.DatabaseID, job.ID, string(job.Status))
		}
		return err
	case domain.BackupPreparing, domain.BackupRunning, domain.BackupUploading, domain.BackupVerifying:
		if err := job.Transition(domain.BackupCancelling); err != nil {
			return err
		}
		if _, err := s.Store.UpdateJob(ctx, job); err != nil {
			return err
		}
		s.notifyBackup(ctx, orgID, job.DatabaseID, job.ID, string(job.Status))
		return s.enqueue(ctx, queue.Job{ID: "cancel:" + job.ID.String(), Type: "backup.cancel", OrgID: orgID.String(), Payload: []byte(job.ID.String())})
	default:
		return domain.ErrValidation
	}
}

func (s *DatabaseBackups) enqueue(ctx context.Context, job queue.Job) error {
	if s.Outbox != nil {
		payload, err := json.Marshal(job)
		if err != nil {
			return err
		}
		event := events.Event{
			ID: uuid.NewString(), Type: jobEventType(job.Type), AggregateType: "job",
			AggregateID: job.ID, Payload: payload, TS: time.Now().UTC(),
		}
		return s.Outbox.Enqueue(ctx, event, "backups")
	}
	if s.Queue == nil {
		return nil
	}
	return s.Queue.Enqueue(ctx, "backups", job)
}

func jobEventType(jobType string) string {
	return jobType + ".queued"
}

func (s *DatabaseBackups) failJob(ctx context.Context, orgID uuid.UUID, job *domain.BackupJob, code, msg string) error {
	now := time.Now()
	job.ErrorCode = code
	job.ErrorMessage = msg
	job.CompletedAt = &now
	if err := job.Transition(domain.BackupFailed); err != nil {
		return err
	}
	_, err := s.Store.UpdateJob(ctx, job)
	s.notifyBackup(ctx, orgID, job.DatabaseID, job.ID, string(job.Status))
	return err
}

func (s *DatabaseBackups) notifyBackup(ctx context.Context, orgID, databaseID, backupID uuid.UUID, status string) {
	if s.Notifier != nil {
		s.Notifier.NotifyBackup(ctx, orgID, databaseID, backupID, status)
	}
}
