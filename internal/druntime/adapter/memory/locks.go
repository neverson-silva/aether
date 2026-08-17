package memory

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"aether/internal/druntime/locks"
)

type heldLock struct {
	token   string
	expires time.Time
}

type LockManager struct {
	mu    sync.Mutex
	held  map[string]heldLock
	seq   atomic.Int64
	nowFn func() time.Time
}

func NewLockManager() *LockManager {
	return &LockManager{held: map[string]heldLock{}, nowFn: time.Now}
}

func (l *LockManager) nextToken() string {
	n := l.seq.Add(1)
	return time.Now().UTC().Format("20060102T150405.000000000") + "-" + itoa(n)
}

func (l *LockManager) Acquire(_ context.Context, name string, ttl time.Duration) (locks.Lock, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.nowFn()
	if h, ok := l.held[name]; ok && now.Before(h.expires) {
		return locks.Lock{}, false, nil
	}
	tok := l.nextToken()
	l.held[name] = heldLock{token: tok, expires: now.Add(ttl)}
	return locks.Lock{Name: name, Token: tok, TTL: ttl}, true, nil
}

func (l *LockManager) Renew(_ context.Context, lk locks.Lock, ttl time.Duration) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	h, ok := l.held[lk.Name]
	if !ok || h.token != lk.Token {
		return locks.ErrLockNotOwned
	}
	h.expires = l.nowFn().Add(ttl)
	l.held[lk.Name] = h
	return nil
}

func (l *LockManager) Release(_ context.Context, lk locks.Lock) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	h, ok := l.held[lk.Name]
	if !ok || h.token != lk.Token {
		return locks.ErrLockNotOwned
	}
	delete(l.held, lk.Name)
	return nil
}

func (l *LockManager) Locked(_ context.Context, name string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	h, ok := l.held[name]
	if !ok {
		return false, nil
	}
	if l.nowFn().After(h.expires) {
		delete(l.held, name)
		return false, nil
	}
	return true, nil
}
