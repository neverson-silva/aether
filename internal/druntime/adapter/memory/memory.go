package memory

import (
	"context"

	"aether/internal/druntime"
)

func New(_ context.Context, _ druntime.Config) (*druntime.Runtime, error) {
	ps := NewPubSub()
	rt := &druntime.Runtime{
		Backend:   "memory",
		PubSub:    ps,
		Cache:     NewCache(),
		Queue:     NewQueue(),
		Locks:     NewLockManager(),
		RateLimit: NewRateLimiter(),
		Presence:  NewPresence(),
		Events:    NewEventBus(ps),
		State:     NewState(),
	}
	rt.SetCloser(func(context.Context) error { return nil })
	return rt, nil
}
