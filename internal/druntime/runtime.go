package druntime

import (
	"context"
	"time"

	"aether/internal/druntime/cache"
	"aether/internal/druntime/events"
	"aether/internal/druntime/locks"
	"aether/internal/druntime/presence"
	"aether/internal/druntime/pubsub"
	"aether/internal/druntime/queue"
	"aether/internal/druntime/ratelimit"
	"aether/internal/druntime/state"
)

type Config struct {
	Backend       string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	RedisUsername string
}

type Runtime struct {
	Backend   string
	PubSub    pubsub.PubSub
	Cache     cache.Cache
	Queue     queue.Queue
	Locks     locks.LockManager
	RateLimit ratelimit.RateLimiter
	Presence  presence.Presence
	Events    events.EventBus
	State     state.State

	close func(context.Context) error
}

func (r *Runtime) Close(ctx context.Context) error {
	if r.close != nil {
		return r.close(ctx)
	}
	return nil
}

func (r *Runtime) SetCloser(fn func(context.Context) error) {
	r.close = fn
}

func (r *Runtime) SetState(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.State.Set(ctx, key, value, ttl)
}

func (r *Runtime) RunOnce(ctx context.Context, key string, ttl time.Duration, fn func() error) (bool, error) {
	added, err := r.Cache.Add(ctx, key, []byte("1"), ttl)
	if err != nil {
		return false, err
	}
	if !added {
		return false, nil
	}
	if err := fn(); err != nil {
		return true, err
	}
	return true, nil
}

type Metrics struct {
	Backend       string
	CacheHits     int64
	CacheMisses   int64
	CacheSets     int64
	CacheErrors   int64
	Subscribers   map[string]int
	TotalChannels int
}

func (r *Runtime) Metrics(ctx context.Context) Metrics {
	m := Metrics{Backend: r.Backend}
	cm := r.Cache.Metrics(ctx)
	m.CacheHits = cm.Hits
	m.CacheMisses = cm.Misses
	m.CacheSets = cm.Sets
	m.CacheErrors = cm.Errors
	if subs, err := r.PubSub.Subscribers(ctx); err == nil {
		m.Subscribers = subs
		m.TotalChannels = len(subs)
	}
	return m
}
