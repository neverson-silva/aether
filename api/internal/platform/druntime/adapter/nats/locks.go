package nats

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"aether/internal/platform/druntime/locks"
)

type lockValue struct {
	Owner   string    `json:"owner"`
	Expires time.Time `json:"expires"`
}

type LockManager struct {
	rt *Runtime
}

func (l *LockManager) Acquire(ctx context.Context, name string, ttl time.Duration) (locks.Lock, bool, error) {
	key := lockKey(name)
	owner := randomToken()
	raw, err := json.Marshal(lockValue{Owner: owner, Expires: time.Now().Add(ttl)})
	if err != nil {
		return locks.Lock{}, false, err
	}
	revision, err := l.rt.locks.Create(ctx, key, raw)
	if err == nil {
		return locks.Lock{Name: name, Token: fmt.Sprintf("%d:%s", revision, owner), TTL: ttl}, true, nil
	}
	if !errors.Is(err, jetstream.ErrKeyExists) {
		return locks.Lock{}, false, err
	}
	entry, err := l.rt.locks.Get(ctx, key)
	if err != nil {
		return locks.Lock{}, false, err
	}
	var current lockValue
	if json.Unmarshal(entry.Value(), &current) != nil || time.Now().Before(current.Expires) {
		return locks.Lock{}, false, nil
	}
	revision, err = l.rt.locks.Update(ctx, key, raw, entry.Revision())
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyRevisionMismatch) {
			return locks.Lock{}, false, nil
		}
		return locks.Lock{}, false, err
	}
	return locks.Lock{Name: name, Token: fmt.Sprintf("%d:%s", revision, owner), TTL: ttl}, true, nil
}

func (l *LockManager) Renew(ctx context.Context, lock locks.Lock, ttl time.Duration) error {
	owner, err := parseLockToken(lock.Token)
	if err != nil {
		return err
	}
	entry, err := l.rt.locks.Get(ctx, lockKey(lock.Name))
	if err != nil {
		return err
	}
	var value lockValue
	if err := json.Unmarshal(entry.Value(), &value); err != nil || value.Owner != owner {
		return locks.ErrLockNotOwned
	}
	raw, _ := json.Marshal(lockValue{Owner: owner, Expires: time.Now().Add(ttl)})
	_, err = l.rt.locks.Update(ctx, lockKey(lock.Name), raw, entry.Revision())
	if errors.Is(err, jetstream.ErrKeyRevisionMismatch) {
		return locks.ErrLockNotOwned
	}
	return err
}

func (l *LockManager) Release(ctx context.Context, lock locks.Lock) error {
	owner, err := parseLockToken(lock.Token)
	if err != nil {
		return err
	}
	entry, err := l.rt.locks.Get(ctx, lockKey(lock.Name))
	if err != nil {
		return err
	}
	var value lockValue
	if err := json.Unmarshal(entry.Value(), &value); err != nil || value.Owner != owner {
		return locks.ErrLockNotOwned
	}
	return l.rt.locks.Delete(ctx, lockKey(lock.Name), jetstream.LastRevision(entry.Revision()))
}

func (l *LockManager) Locked(ctx context.Context, name string) (bool, error) {
	entry, err := l.rt.locks.Get(ctx, lockKey(name))
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var value lockValue
	if json.Unmarshal(entry.Value(), &value) != nil || time.Now().After(value.Expires) {
		return false, nil
	}
	return true, nil
}

func lockKey(name string) string {
	return strings.NewReplacer(":", "_", "/", "_", " ", "_").Replace(name)
}

func randomToken() string {
	raw := make([]byte, 16)
	_, _ = rand.Read(raw)
	return hex.EncodeToString(raw)
}

func parseLockToken(token string) (string, error) {
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return "", locks.ErrLockNotOwned
	}
	if _, err := fmt.Sscanf(parts[0], "%d", new(uint64)); err != nil {
		return "", locks.ErrLockNotOwned
	}
	return parts[1], nil
}

var _ locks.LockManager = (*LockManager)(nil)
