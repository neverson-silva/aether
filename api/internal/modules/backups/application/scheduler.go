package application

import (
	"context"
	"time"

	"aether/internal/modules/backups/domain"
	"aether/internal/platform/druntime/queue"
)

type BackupScheduler struct {
	Service *DatabaseBackups
}

func (sc *BackupScheduler) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	sc.Service.reconcileNow(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sc.Service.reconcileNow(ctx)
		}
	}
}

func (s *DatabaseBackups) reconcileNow(ctx context.Context) {
	configs, err := s.Store.ListEnabledConfigurations(ctx)
	if err != nil {
		return
	}
	now := time.Now()
	for _, cfg := range configs {
		if cfg.NextRunAt == nil || !cfg.NextRunAt.Before(now) {
			continue
		}
		active, err := s.Store.ListActiveJobsByDatabase(ctx, cfg.DatabaseID)
		if err != nil || len(active) > 0 {
			s.advanceNextRun(ctx, &cfg)
			continue
		}
		job, err := s.Store.CreateJob(ctx, &domain.BackupJob{
			DatabaseID: cfg.DatabaseID, Configuration: &cfg.ID, Trigger: domain.TriggerScheduled,
			Status: domain.BackupQueued, DestinationID: cfg.DestinationID,
		})
		if err != nil {
			continue
		}
		_ = s.enqueue(ctx, queue.Job{ID: job.ID.String(), Type: "backup", OrgID: cfg.OrgID.String(), Payload: []byte(job.ID.String())})
		s.advanceNextRun(ctx, &cfg)
	}
}

func (s *DatabaseBackups) advanceNextRun(ctx context.Context, cfg *domain.BackupConfiguration) {
	next, err := NextRun(cfg.Schedule, time.Now())
	if err != nil {
		next = time.Now().Add(24 * time.Hour)
	}
	_ = s.Store.SetConfigurationNextRun(ctx, cfg.ID, &next)
}
