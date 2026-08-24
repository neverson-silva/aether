package nats

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"aether/internal/platform/druntime/cache"
)

type cacheEntry struct {
	value   []byte
	expires time.Time
}

type localCache struct {
	mu     sync.RWMutex
	items  map[string]cacheEntry
	hits   atomic.Int64
	misses atomic.Int64
	sets   atomic.Int64
	errors atomic.Int64
}

func NewCache() cache.Cache { return &localCache{items: map[string]cacheEntry{}} }

func (c *localCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || (!entry.expires.IsZero() && time.Now().After(entry.expires)) {
		c.misses.Add(1)
		return nil, false, nil
	}
	c.hits.Add(1)
	return append([]byte(nil), entry.value...), true, nil
}

func (c *localCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	var expires time.Time
	if ttl > 0 {
		expires = time.Now().Add(ttl)
	}
	c.mu.Lock()
	c.items[key] = cacheEntry{value: append([]byte(nil), value...), expires: expires}
	c.mu.Unlock()
	c.sets.Add(1)
	return nil
}

func (c *localCache) Add(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	c.mu.Lock()
	if entry, ok := c.items[key]; ok && (entry.expires.IsZero() || time.Now().Before(entry.expires)) {
		c.mu.Unlock()
		return false, nil
	}
	var expires time.Time
	if ttl > 0 {
		expires = time.Now().Add(ttl)
	}
	c.items[key] = cacheEntry{value: append([]byte(nil), value...), expires: expires}
	c.mu.Unlock()
	c.sets.Add(1)
	return true, nil
}

func (c *localCache) Del(_ context.Context, key string) error {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
	return nil
}

func (c *localCache) Invalidate(_ context.Context, prefix string) error {
	c.mu.Lock()
	for key := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.items, key)
		}
	}
	c.mu.Unlock()
	return nil
}

func (c *localCache) Metrics(_ context.Context) cache.CacheMetrics {
	return cache.CacheMetrics{Hits: c.hits.Load(), Misses: c.misses.Load(), Sets: c.sets.Load(), Errors: c.errors.Load()}
}

func (c *localCache) GetJSON(ctx context.Context, key string, value any) (bool, error) {
	raw, ok, err := c.Get(ctx, key)
	if err != nil || !ok {
		return ok, err
	}
	return true, json.Unmarshal(raw, value)
}

func (c *localCache) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.Set(ctx, key, raw, ttl)
}
