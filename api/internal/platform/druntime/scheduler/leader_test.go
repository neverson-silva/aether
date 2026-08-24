package scheduler

import (
	"context"
	"testing"
	"time"

	"aether/internal/platform/druntime/locks"
)

type leaderLockManager struct {
	acquired bool
	released bool
}

func (m *leaderLockManager) Acquire(context.Context, string, time.Duration) (locks.Lock, bool, error) {
	if m.acquired {
		return locks.Lock{}, false, nil
	}
	m.acquired = true
	return locks.Lock{Name: "scheduler/test", Token: "1:token"}, true, nil
}

func (m *leaderLockManager) Renew(context.Context, locks.Lock, time.Duration) error { return nil }

func (m *leaderLockManager) Release(context.Context, locks.Lock) error {
	m.released = true
	return nil
}

func (m *leaderLockManager) Locked(context.Context, string) (bool, error) { return m.acquired, nil }

func TestWithLeadershipRunsOnlyWhenAcquired(t *testing.T) {
	manager := &leaderLockManager{}
	runs := 0
	if err := WithLeadership(context.Background(), manager, "test", func(context.Context) error {
		runs++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := WithLeadership(context.Background(), manager, "test", func(context.Context) error {
		runs++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if runs != 1 || !manager.released {
		t.Fatalf("unexpected leadership state: runs=%d released=%v", runs, manager.released)
	}
}
