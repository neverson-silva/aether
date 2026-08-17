package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"aether/internal/druntime/locks"
)

const lockScript = `
local nonce = ARGV[1]
local ttl = ARGV[2]
local fence = redis.call("INCR", KEYS[2])
local ok = redis.call("SET", KEYS[1], nonce, "NX", "PX", ttl)
if ok then return fence end
return 0
`

const lockRenewScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`

const lockReleaseScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`

type LockManager struct {
	rt *Runtime
}

func (l *LockManager) Acquire(ctx context.Context, name string, ttl time.Duration) (locks.Lock, bool, error) {
	key := "lock:" + name
	nonce := randHex(16)
	fence, err := l.rt.client.Eval(ctx, lockScript, []string{key, "lock:seq:" + name}, nonce, ttl.Milliseconds()).Result()
	if err != nil {
		l.rt.observe(err)
		return locks.Lock{}, false, err
	}
	n, _ := fence.(int64)
	if n == 0 {
		return locks.Lock{}, false, nil
	}
	return locks.Lock{Name: name, Token: fmt.Sprintf("%d:%s", n, nonce), TTL: ttl}, true, nil
}

func (l *LockManager) Renew(ctx context.Context, lk locks.Lock, ttl time.Duration) error {
	nonce := lockNonce(lk.Token)
	if nonce == "" {
		return locks.ErrLockNotOwned
	}
	res, err := l.rt.client.Eval(ctx, lockRenewScript, []string{"lock:" + lk.Name}, nonce, ttl.Milliseconds()).Result()
	l.rt.observe(err)
	if err != nil {
		return err
	}
	if n, _ := res.(int64); n == 0 {
		return locks.ErrLockNotOwned
	}
	return nil
}

func (l *LockManager) Release(ctx context.Context, lk locks.Lock) error {
	nonce := lockNonce(lk.Token)
	if nonce == "" {
		return locks.ErrLockNotOwned
	}
	res, err := l.rt.client.Eval(ctx, lockReleaseScript, []string{"lock:" + lk.Name}, nonce).Result()
	l.rt.observe(err)
	if err != nil {
		return err
	}
	if n, _ := res.(int64); n == 0 {
		return locks.ErrLockNotOwned
	}
	return nil
}

func (l *LockManager) Locked(ctx context.Context, name string) (bool, error) {
	n, err := l.rt.client.Exists(ctx, "lock:"+name).Result()
	l.rt.observe(err)
	return n > 0, err
}

func lockNonce(token string) string {
	i := strings.Index(token, ":")
	if i < 0 {
		return ""
	}
	return token[i+1:]
}

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
