package redis

import (
	"context"
	"sync/atomic"

	"github.com/redis/go-redis/v9"

	"aether/internal/platform/druntime"
)

type metrics struct {
	ops    atomic.Int64
	errors atomic.Int64
	hits   atomic.Int64
	misses atomic.Int64
	events atomic.Int64
	subs   atomic.Int64
	queue  atomic.Int64
	locked atomic.Int64
}

type Runtime struct {
	client *redis.Client
	met    metrics
	pubsub *PubSub
	cache  *Cache
	queue  *Queue
	locks  *LockManager
	rl     *RateLimiter
	pres   *Presence
	events *EventBus
	state  *State
}

func New(ctx context.Context, cfg druntime.Config) (*druntime.Runtime, error) {
	addr := cfg.RedisAddr
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
		Username: cfg.RedisUsername,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	rt := &Runtime{client: client}
	rt.pubsub = &PubSub{rt: rt}
	rt.cache = &Cache{rt: rt}
	rt.queue = &Queue{rt: rt}
	rt.locks = &LockManager{rt: rt}
	rt.rl = &RateLimiter{rt: rt}
	rt.pres = &Presence{rt: rt}
	rt.events = &EventBus{rt: rt}
	rt.state = &State{rt: rt}
	r := &druntime.Runtime{
		Backend:   "redis",
		PubSub:    rt.pubsub,
		Cache:     rt.cache,
		Queue:     rt.queue,
		Locks:     rt.locks,
		RateLimit: rt.rl,
		Presence:  rt.pres,
		Events:    rt.events,
		State:     rt.state,
	}
	r.SetCloser(func(ctx context.Context) error {
		rt.queue.Close(ctx)
		return client.Close()
	})
	return r, nil
}

func (r *Runtime) observe(err error) {
	r.met.ops.Add(1)
	if err != nil && err != redis.Nil {
		r.met.errors.Add(1)
	}
}

func (r *Runtime) Close(ctx context.Context) error {
	return r.client.Close()
}
