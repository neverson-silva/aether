package core

import (
	"context"
	"errors"
	"log"
	"time"
)

var ErrLockBusy = errors.New("lock ocupado")

const (
	lockDeployTTL  = 20 * time.Minute
	lockCronTTL    = 30 * time.Minute
	lockBackupTTL  = 2 * time.Hour
	lockCleanupTTL = 30 * time.Minute
	lockCertTTL    = 2 * time.Hour
)

func (c *Core) withLock(name string, ttl time.Duration, fn func() error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	lock, ok, err := c.RT.Locks.Acquire(ctx, name, ttl)
	cancel()
	if err != nil {
		return err
	}
	if !ok {
		return ErrLockBusy
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.RT.Locks.Release(ctx, lock)
	}()
	return fn()
}

func (c *Core) withLockSkip(name string, ttl time.Duration, fn func()) {
	if err := c.withLock(name, ttl, func() error {
		fn()
		return nil
	}); err != nil {
		if err == ErrLockBusy {
			log.Printf("[lock] %s: ocupado — pulando", name)
			return
		}
		log.Printf("[lock] %s: %v", name, err)
	}
}
