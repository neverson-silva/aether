package redis

import (
	"context"
	"fmt"
	"time"

	"aether/internal/platform/druntime/ratelimit"
)

const tokenBucketScript = `
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local need = tonumber(ARGV[4])
local ttl = tonumber(ARGV[5])
local current = redis.call("HMGET", key, "tokens", "ts")
local tokens = burst
local ts = now
if current[1] then
  tokens = tonumber(current[1])
  ts = tonumber(current[2])
end
local elapsed = (now - ts) / 1000
tokens = tokens + elapsed * rate
if tokens > burst then tokens = burst end
local allowed = 0
if tokens >= need then
  tokens = tokens - need
  allowed = 1
end
redis.call("HSET", key, "tokens", tokens, "ts", now)
redis.call("PEXPIRE", key, ttl)
local remaining = math.floor(tokens)
local reset = 0
if allowed == 0 then
  reset = math.ceil((need - tokens) / rate * 1000)
end
return {allowed, remaining, reset}
`

type RateLimiter struct {
	rt *Runtime
}

func (r *RateLimiter) Allow(ctx context.Context, tip ratelimit.KeyTip, key string, n int, rate float64, burst int) (ratelimit.Decision, error) {
	k := "rl:" + string(tip) + ":" + key
	now := time.Now().UnixMilli()
	res, err := r.rt.client.Eval(ctx, tokenBucketScript, []string{k}, rate, burst, now, n, 60_000).Result()
	if err != nil {
		r.rt.observe(err)
		return ratelimit.Decision{}, err
	}
	arr, ok := res.([]any)
	if !ok || len(arr) != 3 {
		return ratelimit.Decision{}, fmt.Errorf("invalid rate limit response")
	}
	return ratelimit.Decision{
		Allowed:   arr[0].(int64) == 1,
		Remaining: int(arr[1].(int64)),
		ResetIn:   time.Duration(arr[2].(int64)) * time.Millisecond,
	}, nil
}

func (r *RateLimiter) Reset(ctx context.Context, tip ratelimit.KeyTip, key string) error {
	err := r.rt.client.Del(ctx, "rl:"+string(tip)+":"+key).Err()
	r.rt.observe(err)
	return err
}
