package pubsub

import "context"

type Message struct {
	Channel string
	Data    []byte
}

type Subscription interface {
	Unsubscribe() error
}

type Handler func(ctx context.Context, msg Message)

type PubSub interface {
	Publish(ctx context.Context, channel string, data []byte) error
	Subscribe(ctx context.Context, channel string, h Handler, opts ...Option) (Subscription, error)
	Subscribers(ctx context.Context) (map[string]int, error)
}

type Option func(*SubscribeOptions)

type SubscribeOptions struct {
	BufferSize int
}

func WithBuffer(n int) Option {
	return func(o *SubscribeOptions) { o.BufferSize = n }
}
