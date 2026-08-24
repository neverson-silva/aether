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

func (s *DatabaseBackups) RequestRestore(ctx context.Context, backupID, targetDBID, orgID uuid.UUID) (*domain.RestoreJob, error) {
	backup, err := s.GetBackup(ctx, backupID, orgID)
	if err != nil {
		return nil, err
	}
	if backup.Status != domain.BackupCompleted {
		return nil, domain.ErrValidation
	}
	target, err := s.Databases.Get(ctx, targetDBID, orgID)
	if err != nil {
		return nil, err
	}
	if string(target.Engine) != backup.Engine {
		return nil, domain.ErrValidation
	}
	active, err := s.Store.ListActiveJobsByDatabase(ctx, targetDBID)
	if err != nil {
		return nil, err
	}
	if len(active) > 0 {
		return nil, domain.ErrConflict
	}
	job, err := s.Store.CreateRestoreJob(ctx, &domain.RestoreJob{
		BackupID: backupID, TargetDatabaseID: targetDBID, Status: domain.RestoreQueued,
	})
	if err != nil {
		return nil, err
	}
	s.Audit.Record(ctx, orgID, "restore.started", "database", targetDBID.String(), backupID.String())
	jobPayload := queue.Job{ID: job.ID.String(), Type: "restore", OrgID: orgID.String(), Payload: []byte(job.ID.String())}
	var enqueueErr error
	if s.Outbox != nil {
		payload, marshalErr := json.Marshal(jobPayload)
		if marshalErr != nil {
			enqueueErr = marshalErr
		} else {
			enqueueErr = s.Outbox.Enqueue(ctx, events.Event{ID: uuid.NewString(), Type: "restore.queued", AggregateType: "restore", AggregateID: job.ID.String(), Payload: payload, TS: time.Now().UTC()}, "backups")
		}
	} else if s.Queue != nil {
		enqueueErr = s.Queue.Enqueue(ctx, "backups", jobPayload)
	} else {
		enqueueErr = domain.ErrValidation
	}
	if enqueueErr != nil {
		_ = s.failRestore(ctx, job, "RESTORE_ENQUEUE_FAILED", enqueueErr.Error())
		return job, enqueueErr
	}
	return job, nil
}

func (s *DatabaseBackups) ListRestoreJobs(ctx context.Context, targetDBID, orgID uuid.UUID, limit int) ([]domain.RestoreJob, error) {
	if _, err := s.Databases.Get(ctx, targetDBID, orgID); err != nil {
		return nil, err
	}
	return s.Store.ListRestoreJobsByTarget(ctx, targetDBID, limit)
}
