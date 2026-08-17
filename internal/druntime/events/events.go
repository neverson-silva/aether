package events

import (
	"context"
	"time"
)

type Event struct {
	ID            string
	Type          string
	AggregateType string
	AggregateID   string
	Payload       []byte
	TS            time.Time
}

type Subscription interface {
	Unsubscribe() error
}

type Handler func(ctx context.Context, ev Event)

type EventBus interface {
	Publish(ctx context.Context, topic string, ev Event) error
	Subscribe(ctx context.Context, topic string, h Handler) (Subscription, error)
}
