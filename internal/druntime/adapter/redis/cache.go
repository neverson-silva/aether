package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"aether/internal/druntime/cache"
)

type Cache struct {
	rt *Runtime
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	v, err := c.rt.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		c.rt.met.misses.Add(1)
		return nil, false, nil
	}
	if err != nil {
		c.rt.observe(err)
		return nil, false, err
	}
	c.rt.met.hits.Add(1)
	return v, true, nil
}

func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	err := c.rt.client.Set(ctx, key, value, ttl).Err()
	c.rt.observe(err)
	if err == nil {
		c.rt.met.ops.Add(1)
	}
	return err
}

func (c *Cache) Add(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	ok, err := c.rt.client.SetNX(ctx, key, value, ttl).Result()
	c.rt.observe(err)
	return ok, err
}

func (c *Cache) Del(ctx context.Context, key string) error {
	err := c.rt.client.Del(ctx, key).Err()
	c.rt.observe(err)
	return err
}

func (c *Cache) Invalidate(ctx context.Context, prefix string) error {
	var cursor uint64
	var keys []string
	for {
		var err error
		keys, cursor, err = c.rt.client.Scan(ctx, cursor, prefix+"*", 500).Result()
		if err != nil {
			c.rt.observe(err)
			return err
		}
		if len(keys) > 0 {
			if err := c.rt.client.Del(ctx, keys...).Err(); err != nil {
				c.rt.observe(err)
				return err
			}
		}
		if cursor == 0 {
			break
		}
	}
	return nil
}

func (c *Cache) Metrics(_ context.Context) cache.CacheMetrics {
	return cache.CacheMetrics{
		Hits:   c.rt.met.hits.Load(),
		Misses: c.rt.met.misses.Load(),
		Sets:   c.rt.met.ops.Load(),
		Errors: c.rt.met.errors.Load(),
	}
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
