package api

import (
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

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

func RateLimit(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.Allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}
