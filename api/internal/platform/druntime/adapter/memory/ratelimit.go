package memory

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

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	nowFn   func() time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{buckets: map[string]*bucket{}, nowFn: time.Now}
}

func (r *RateLimiter) Allow(_ context.Context, tip ratelimit.KeyTip, key string, n int, rate float64, burst int) (ratelimit.Decision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fullKey := string(tip) + ":" + key
	b := r.buckets[fullKey]
	now := r.nowFn()
	if b == nil {
		b = &bucket{tokens: float64(burst), last: now}
		r.buckets[fullKey] = b
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * rate
	if b.tokens > float64(burst) {
		b.tokens = float64(burst)
	}
	b.last = now
	need := float64(n)
	if b.tokens >= need {
		b.tokens -= need
		remaining := int(b.tokens)
		return ratelimit.Decision{Allowed: true, Remaining: remaining}, nil
	}
	refill := (need - b.tokens) / rate
	return ratelimit.Decision{Allowed: false, Remaining: int(b.tokens), ResetIn: time.Duration(refill * float64(time.Second))}, nil
}

func (r *RateLimiter) Reset(_ context.Context, tip ratelimit.KeyTip, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.buckets, string(tip)+":"+key)
	return nil
}
