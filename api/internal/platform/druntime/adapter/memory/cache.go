package memory

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"aether/internal/platform/druntime/cache"
)

type entry struct {
	value []byte
	exp   time.Time
}

type Cache struct {
	mu   sync.Mutex
	data map[string]entry
	hits int64
	miss int64
	sets int64
	errs int64
}

func NewCache() *Cache {
	return &Cache{data: map[string]entry{}}
}

func (c *Cache) Get(_ context.Context, key string) ([]byte, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.data[key]
	if !ok || (!e.exp.IsZero() && time.Now().After(e.exp)) {
		if ok {
			delete(c.data, key)
		}
		c.miss++
		return nil, false, nil
	}
	c.hits++
	return e.value, true, nil
}

func (c *Cache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	exp := time.Time{}
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.data[key] = entry{value: value, exp: exp}
	c.sets++
	return nil
}

func (c *Cache) Add(_ context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.data[key]; ok && (e.exp.IsZero() || time.Now().Before(e.exp)) {
		c.miss++
		return false, nil
	}
	exp := time.Time{}
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.data[key] = entry{value: value, exp: exp}
	c.sets++
	return true, nil
}

func (c *Cache) Del(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
	return nil
}

func (c *Cache) Invalidate(_ context.Context, prefix string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.data {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.data, k)
		}
	}
	return nil
}

func (c *Cache) Metrics(_ context.Context) cache.CacheMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	return cache.CacheMetrics{Hits: c.hits, Misses: c.miss, Sets: c.sets, Errors: c.errs}
}

func (c *Cache) GetJSON(ctx context.Context, key string, v any) (bool, error) {
	b, ok, err := c.Get(ctx, key)
	if err != nil || !ok {
		return ok, err
	}
	return true, json.Unmarshal(b, v)
}

func (c *Cache) SetJSON(ctx context.Context, key string, v any, ttl time.Duration) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Set(ctx, key, b, ttl)
}
