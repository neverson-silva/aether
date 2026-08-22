package presence

import (
	"context"
	"time"
)

type Presence interface {
	Join(ctx context.Context, scope, member string, ttl time.Duration) error
	Leave(ctx context.Context, scope, member string) error
	Heartbeat(ctx context.Context, scope, member string, ttl time.Duration) error
	Count(ctx context.Context, scope string) (int64, error)
	Members(ctx context.Context, scope string) ([]string, error)
}
