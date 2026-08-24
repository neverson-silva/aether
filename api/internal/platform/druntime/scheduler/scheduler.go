package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"aether/internal/platform/druntime/locks"
)

type Scheduler interface {
	ScheduleAt(ctx context.Context, key string, at time.Time, payload []byte) error
	ScheduleJobAt(ctx context.Context, subject, key, jobType string, at time.Time, payload []byte) error
}

type RecurringScheduler interface {
	ScheduleJobCron(ctx context.Context, subject, key, jobType, expression, timezone string, payload []byte) error
	ScheduleJobEvery(ctx context.Context, subject, key, jobType string, interval time.Duration, payload []byte) error
	ReconcileRecurring(ctx context.Context, namespace string, activeKeys []string) error
}

type ReconcileTask struct {
	Name string
	Run  func(context.Context) error
}

const ReconcileInterval = 30 * time.Second

func Reconcile(ctx context.Context, manager locks.LockManager, tasks ...ReconcileTask) error {
	return WithLeadership(ctx, manager, "leader", func(ctx context.Context) error {
		var firstErr error
		for _, task := range tasks {
			if task.Run == nil {
				continue
			}
			if err := task.Run(ctx); err != nil {
				if task.Name == "" {
					if firstErr == nil {
						firstErr = err
					}
					continue
				}
				if firstErr == nil {
					firstErr = fmt.Errorf("%s: %w", task.Name, err)
				}
			}
		}
		return firstErr
	})
}

func Run(ctx context.Context, manager locks.LockManager, tasks ...ReconcileTask) {
	ticker := time.NewTicker(ReconcileInterval)
	defer ticker.Stop()
	for {
		if err := Reconcile(ctx, manager, tasks...); err != nil {
			slog.Default().Error("scheduler reconciliation failed", "error", err)
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}
