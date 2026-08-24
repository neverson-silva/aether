package scheduler

import (
	"context"
	"time"

	"aether/internal/platform/druntime/locks"
)

const leaderTTL = 30 * time.Second

func WithLeadership(ctx context.Context, manager locks.LockManager, name string, fn func(context.Context) error) error {
	if manager == nil {
		return fn(ctx)
	}
	lock, acquired, err := manager.Acquire(ctx, "scheduler/"+name, leaderTTL)
	if err != nil || !acquired {
		return err
	}
	defer manager.Release(context.Background(), lock)
	return fn(ctx)
}
