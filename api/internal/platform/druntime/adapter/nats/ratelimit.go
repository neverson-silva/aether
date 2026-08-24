package nats

import (
	"context"
	"sync"
	"time"

	"aether/internal/platform/druntime/ratelimit"
)

type bucket struct {
	tokens float64
	last   time.Time
}

type localRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]bucket
}

func NewRateLimiter() ratelimit.RateLimiter {
	return &localRateLimiter{buckets: map[string]bucket{}}
}

func (r *localRateLimiter) Allow(_ context.Context, tip ratelimit.KeyTip, key string, n int, rate float64, burst int) (ratelimit.Decision, error) {
	if rate <= 0 || burst <= 0 || n <= 0 {
		return ratelimit.Decision{Allowed: false}, nil
	}
	now := time.Now()
	id := string(tip) + ":" + key
	r.mu.Lock()
	b := r.buckets[id]
	if b.last.IsZero() {
		b.tokens = float64(burst)
		b.last = now
	}
	b.tokens += now.Sub(b.last).Seconds() * rate
	if b.tokens > float64(burst) {
		b.tokens = float64(burst)
	}
	b.last = now
	allowed := b.tokens >= float64(n)
	if allowed {
		b.tokens -= float64(n)
	}
	remaining := int(b.tokens)
	reset := time.Duration(0)
	if !allowed {
		reset = time.Duration((float64(n) - b.tokens) / rate * float64(time.Second))
	}
	r.buckets[id] = b
	r.mu.Unlock()
	return ratelimit.Decision{Allowed: allowed, Remaining: remaining, ResetIn: reset}, nil
}

func (r *localRateLimiter) Reset(_ context.Context, tip ratelimit.KeyTip, key string) error {
	r.mu.Lock()
	delete(r.buckets, string(tip)+":"+key)
	r.mu.Unlock()
	return nil
}
