package locks

import (
	"context"
	"errors"
	"time"
)

var ErrLockNotOwned = errors.New("lock does not belong to the token")

type Lock struct {
	Name  string
	Token string
	TTL   time.Duration
}

type LockManager interface {
	Acquire(ctx context.Context, name string, ttl time.Duration) (Lock, bool, error)
	Renew(ctx context.Context, l Lock, ttl time.Duration) error
	Release(ctx context.Context, l Lock) error
	Locked(ctx context.Context, name string) (bool, error)
}
