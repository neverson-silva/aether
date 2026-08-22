package cache

import (
	"context"
	"time"
)

type CacheMetrics struct {
	Hits   int64
	Misses int64
	Sets   int64
	Errors int64
}

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Add(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)
	Del(ctx context.Context, key string) error
	Invalidate(ctx context.Context, prefix string) error
	Metrics(ctx context.Context) CacheMetrics
}

type JSONCache interface {
	GetJSON(ctx context.Context, key string, v any) (bool, error)
	SetJSON(ctx context.Context, key string, v any, ttl time.Duration) error
}
