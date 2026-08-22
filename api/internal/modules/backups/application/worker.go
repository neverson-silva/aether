package application

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/backups/domain"
)

type BackupWorker struct {
	Service *DatabaseBackups
}

func (w *BackupWorker) Run(ctx context.Context) {
	if w.Service.Queue == nil {
		return
	}
	consumer, err := w.Service.Queue.NewConsumer(ctx, "backups", "backup-workers", "aether-backup")
	if err != nil {
		return
	}
	defer consumer.Close()

	var mu sync.Mutex
	inFlight := map[string]bool{}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		job, err := consumer.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}
		mu.Lock()
		if inFlight[job.ID] {
			mu.Unlock()
			_ = consumer.Nack(ctx, job)
			continue
		}
		inFlight[job.ID] = true
		mu.Unlock()

		orgID, _ := uuid.Parse(job.OrgID)
		jobUUID, _ := uuid.Parse(job.ID)
		switch job.Type {
		case "backup":
			w.Service.runBackup(ctx, orgID, jobUUID)
		case "backup.cancel":
			w.Service.cancelRunning(ctx, orgID, jobUUID)
		case "restore":
			w.Service.runRestore(ctx, orgID, jobUUID)
		}

		mu.Lock()
		delete(inFlight, job.ID)
		mu.Unlock()
		_ = consumer.Ack(ctx, job)
	}
}

func (s *DatabaseBackups) cancelRunning(ctx context.Context, orgID, jobID uuid.UUID) {
	job, err := s.Store.GetJob(ctx, jobID)
	if err != nil {
		return
	}
	if job.Status != domain.BackupCancelling {
		return
	}
	now := time.Now()
	job.CompletedAt = &now
	if err := job.Transition(domain.BackupCancelled); err != nil {
		return
	}
	_, _ = s.Store.UpdateJob(ctx, job)
	s.notifyBackup(ctx, orgID, job.DatabaseID, job.ID, string(job.Status))
	s.Audit.Record(ctx, orgID, "backup.cancelled", "database", job.DatabaseID.String(), job.ID.String())
}
