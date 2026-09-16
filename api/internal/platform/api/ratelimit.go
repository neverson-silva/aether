package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RateLimiterBackend interface{ Allow(key string) bool }

type rateLimiterWithError interface {
	AllowWithError(key string) (bool, error)
}

type bucket struct {
	tokens float64
	last   time.Time
}

type RateLimiter struct {
	mu         sync.Mutex
	now        func() time.Time
	rate       float64
	burst      float64
	buckets    map[string]*bucket
	maxBuckets int
}

func NewRateLimiter(rate, burst float64) *RateLimiter {
	return &RateLimiter{
		now:        time.Now,
		rate:       rate,
		burst:      burst,
		buckets:    make(map[string]*bucket),
		maxBuckets: 10000,
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := rl.now()
	if len(rl.buckets) >= rl.maxBuckets {
		for k, b := range rl.buckets {
			if now.Sub(b.last) > time.Hour {
				delete(rl.buckets, k)
			}
		}
	}
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: rl.burst, last: now}
		rl.buckets[key] = b
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = math.Min(rl.burst, b.tokens+elapsed*rl.rate)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

type PostgresRateLimiter struct {
	pool   *pgxpool.Pool
	limit  int
	window time.Duration
}

func NewPostgresRateLimiter(pool *pgxpool.Pool, limit int, window time.Duration) *PostgresRateLimiter {
	return &PostgresRateLimiter{pool: pool, limit: limit, window: window}
}

func (rl *PostgresRateLimiter) Allow(key string) bool {
	allowed, err := rl.AllowWithError(key)
	return err == nil && allowed
}

func (rl *PostgresRateLimiter) AllowWithError(key string) (bool, error) {
	if rl == nil || rl.pool == nil || rl.limit <= 0 || rl.window <= 0 {
		return false, fmt.Errorf("rate limiter is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var allowed bool
	if _, err := rl.pool.Exec(ctx, `DELETE FROM rate_limit_buckets WHERE window_started < now() - interval '1 hour'`); err != nil {
		return false, fmt.Errorf("rate limiter cleanup: %w", err)
	}
	err := rl.pool.QueryRow(ctx, `
		INSERT INTO rate_limit_buckets (bucket_key, window_started, request_count, window_seconds, max_requests)
		VALUES ($1, now(), 1, $2, $3)
		ON CONFLICT (bucket_key) DO UPDATE SET
			request_count = CASE WHEN EXTRACT(EPOCH FROM (now() - rate_limit_buckets.window_started)) >= rate_limit_buckets.window_seconds THEN 1 ELSE rate_limit_buckets.request_count + 1 END,
			window_started = CASE WHEN EXTRACT(EPOCH FROM (now() - rate_limit_buckets.window_started)) >= rate_limit_buckets.window_seconds THEN now() ELSE rate_limit_buckets.window_started END,
			window_seconds = EXCLUDED.window_seconds,
			max_requests = EXCLUDED.max_requests
		RETURNING request_count <= max_requests`, key, int(rl.window.Seconds()), rl.limit).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("rate limiter bucket: %w", err)
	}
	return allowed, nil
}

func RateLimit(rl RateLimiterBackend) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Request.URL.Path + ":" + c.ClientIP()
		if backend, ok := rl.(rateLimiterWithError); ok {
			allowed, err := backend.AllowWithError(key)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "authentication rate limit unavailable"})
				return
			}
			if !allowed {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
				return
			}
			c.Next()
			return
		}
		if !rl.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}
