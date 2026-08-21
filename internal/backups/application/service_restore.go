package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"aether/internal/backups/domain"
	"aether/internal/druntime/queue"
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
	if err := s.enqueue(ctx, queue.Job{ID: job.ID.String(), Type: "restore", OrgID: orgID.String(), Payload: []byte(job.ID.String())}); err != nil {
		now := time.Now()
		job.ErrorCode = "RESTORE_ENQUEUE_FAILED"
		job.ErrorMessage = err.Error()
		job.CompletedAt = &now
		_ = job.Transition(domain.RestoreFailed)
		_, _ = s.Store.UpdateRestoreJob(ctx, job)
		return nil, err
	}
	s.Audit.Record(ctx, orgID, "restore.started", "database", targetDBID.String(), backupID.String())
	return job, nil
}

func (s *DatabaseBackups) ListRestoreJobs(ctx context.Context, targetDBID, orgID uuid.UUID, limit int) ([]domain.RestoreJob, error) {
	if _, err := s.Databases.Get(ctx, targetDBID, orgID); err != nil {
		return nil, err
	}
	return s.Store.ListRestoreJobsByTarget(ctx, targetDBID, limit)
}
