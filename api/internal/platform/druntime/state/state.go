package state

import (
	"context"
	"time"
)

type State interface {
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Del(ctx context.Context, key string) error
	Changes(ctx context.Context, keyPattern string, h func(ctx context.Context, key string, value []byte)) (func(), error)
}
