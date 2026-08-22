package application

import (
	"context"

	"github.com/google/uuid"

	"aether/internal/modules/backups/domain"
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
	s.runRestore(ctx, orgID, job.ID)
	finished, err := s.Store.GetRestoreJob(ctx, job.ID)
	if err != nil {
		return nil, err
	}
	if finished.Status == domain.RestoreFailed {
		return finished, domain.ErrValidation
	}
	if !finished.Terminal() {
		return finished, domain.ErrValidation
	}
	return finished, nil
}

func (s *DatabaseBackups) ListRestoreJobs(ctx context.Context, targetDBID, orgID uuid.UUID, limit int) ([]domain.RestoreJob, error) {
	if _, err := s.Databases.Get(ctx, targetDBID, orgID); err != nil {
		return nil, err
	}
	return s.Store.ListRestoreJobsByTarget(ctx, targetDBID, limit)
}
