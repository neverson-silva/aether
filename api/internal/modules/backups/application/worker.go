package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/backups/domain"
	"aether/internal/platform/druntime/queue"
	"aether/internal/platform/observability"
)

type BackupWorker struct {
	Service     *DatabaseBackups
	Metrics     *observability.Metrics
	Concurrency int
}

func (w *BackupWorker) concurrency() int {
	if w.Concurrency < 1 {
		return 1
	}
	return w.Concurrency
}

type interruptedJobStore interface {
	RecoverInterrupted(context.Context, time.Time) ([]queue.Job, error)
}

func (w *BackupWorker) RecoverInterrupted(ctx context.Context, cutoff time.Time) error {
	store, ok := w.Service.Store.(interruptedJobStore)
	if !ok || w.Service.Queue == nil {
		return nil
	}
	jobs, err := store.RecoverInterrupted(ctx, cutoff)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if err := w.Service.Queue.Enqueue(ctx, "backups", job); err != nil {
			return err
		}
	}
	return nil
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
	for range w.concurrency() {
		go w.consumeLoop(ctx, consumer, inFlight, &mu)
	}
	<-ctx.Done()
}

func (w *BackupWorker) consumeLoop(ctx context.Context, consumer queue.Consumer, inFlight map[string]bool, mu *sync.Mutex) {
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

		stopProgress := queue.StartProgress(ctx, consumer, job)
		finish := func(bool) {}
		if w.Metrics != nil {
			finish = w.Metrics.StartJob(job.Type)
		}
		processErr := w.process(ctx, job)
		stopProgress()
		if w.Metrics != nil {
			finish(processErr != nil)
		}

		mu.Lock()
		delete(inFlight, job.ID)
		mu.Unlock()
		if processErr != nil && !queue.IsPermanent(processErr) {
			_ = consumer.Nack(ctx, job)
		} else {
			_ = consumer.Ack(ctx, job)
		}
	}
}

func (w *BackupWorker) process(ctx context.Context, job *queue.Job) error {
	orgID, err := uuid.Parse(job.OrgID)
	if err != nil {
		return queue.Permanent(fmt.Errorf("invalid organization id: %w", err))
	}
	switch job.Type {
	case "backup.schedule":
		return w.Service.RunScheduled(ctx, job.Payload)
	case "backup":
		jobID, parseErr := uuid.Parse(job.ID)
		if parseErr != nil {
			return queue.Permanent(fmt.Errorf("invalid backup job id: %w", parseErr))
		}
		w.Service.runBackup(ctx, orgID, jobID)
		completed, getErr := w.Service.Store.GetJob(ctx, jobID)
		if getErr != nil {
			return getErr
		}
		if completed == nil || !completed.Terminal() {
			return errors.New("backup did not reach a terminal state")
		}
		return nil
	case "backup.cancel":
		cancelID, parseErr := uuid.Parse(string(job.Payload))
		if parseErr != nil {
			return queue.Permanent(fmt.Errorf("invalid cancellation job id: %w", parseErr))
		}
		return w.Service.cancelRunning(ctx, orgID, cancelID)
	case "restore":
		jobID, parseErr := uuid.Parse(job.ID)
		if parseErr != nil {
			return queue.Permanent(fmt.Errorf("invalid restore job id: %w", parseErr))
		}
		w.Service.runRestore(ctx, orgID, jobID)
		completed, getErr := w.Service.Store.GetRestoreJob(ctx, jobID)
		if getErr != nil {
			return getErr
		}
		if completed == nil || !completed.Terminal() {
			return errors.New("restore did not reach a terminal state")
		}
		return nil
	default:
		return queue.Permanent(fmt.Errorf("unsupported backup job type %q", job.Type))
	}
}

func (s *DatabaseBackups) cancelRunning(ctx context.Context, orgID, jobID uuid.UUID) error {
	job, err := s.Store.GetJob(ctx, jobID)
	if err != nil {
		return err
	}
	if job.Status != domain.BackupCancelling {
		if job.Terminal() {
			return nil
		}
		return fmt.Errorf("backup cancellation is not pending")
	}
	now := time.Now()
	job.CompletedAt = &now
	if err := job.Transition(domain.BackupCancelled); err != nil {
		return err
	}
	if _, err := s.Store.UpdateJob(ctx, job); err != nil {
		return err
	}
	s.notifyBackup(ctx, orgID, job.DatabaseID, job.ID, string(job.Status))
	s.Audit.Record(ctx, orgID, "backup.cancelled", "database", job.DatabaseID.String(), job.ID.String())
	return nil
}
