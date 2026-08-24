package scheduler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"aether/internal/platform/druntime/locks"
)

type testLockManager struct {
	active atomic.Bool
}

func (m *testLockManager) Acquire(_ context.Context, name string, ttl time.Duration) (locks.Lock, bool, error) {
	if !m.active.CompareAndSwap(false, true) {
		return locks.Lock{}, false, nil
	}
	return locks.Lock{Name: name, Token: "test", TTL: ttl}, true, nil
}

func (m *testLockManager) Renew(context.Context, locks.Lock, time.Duration) error {
	return nil
}

func (m *testLockManager) Release(context.Context, locks.Lock) error {
	m.active.Store(false)
	return nil
}

func (m *testLockManager) Locked(context.Context, string) (bool, error) {
	return m.active.Load(), nil
}

func TestReconcileRunsAllTasksAndReturnsFirstError(t *testing.T) {
	manager := &testLockManager{}
	ctx := context.Background()
	var calls atomic.Int32
	firstErr := errors.New("first task failed")

	err := Reconcile(ctx, manager,
		ReconcileTask{Name: "first", Run: func(context.Context) error {
			calls.Add(1)
			return firstErr
		}},
		ReconcileTask{Name: "second", Run: func(context.Context) error {
			calls.Add(1)
			return errors.New("second task failed")
		}},
		ReconcileTask{},
	)

	if !errors.Is(err, firstErr) {
		t.Fatalf("expected first error, got %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected both tasks to run, got %d calls", calls.Load())
	}
}

func TestReconcileUsesOneGlobalLeadershipLock(t *testing.T) {
	manager := &testLockManager{}
	ctx := context.Background()
	var active atomic.Int32
	var maximum atomic.Int32
	task := ReconcileTask{Name: "serialized", Run: func(context.Context) error {
		current := active.Add(1)
		for {
			seen := maximum.Load()
			if current <= seen || maximum.CompareAndSwap(seen, current) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		active.Add(-1)
		return nil
	}}

	done := make(chan struct{}, 2)
	for range 2 {
		go func() {
			_ = Reconcile(ctx, manager, task)
			done <- struct{}{}
		}()
	}
	for range 2 {
		<-done
	}

	if maximum.Load() != 1 {
		t.Fatalf("expected one active scheduler task, got maximum %d", maximum.Load())
	}
}
