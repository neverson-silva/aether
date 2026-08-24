package nats

import (
	"context"
	"encoding/json"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"aether/internal/platform/druntime/events"
	"aether/internal/platform/druntime/pubsub"
	"aether/internal/platform/messaging"
)

type EventBus struct {
	rt *Runtime
}

func (e *EventBus) Publish(ctx context.Context, topic string, event events.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(messaging.Envelope{ID: event.ID, Type: event.Type, SchemaVersion: 1, ResourceID: event.AggregateID, CreatedAt: event.TS, Payload: payload})
	if err != nil {
		return err
	}
	if _, err := e.rt.js.Publish(ctx, messaging.Events(topic), raw, jetstream.WithMsgID(event.ID)); err != nil {
		return err
	}
	return e.rt.conn.Publish(messaging.Live(topic), raw)
}

func (e *EventBus) Subscribe(ctx context.Context, topic string, handler events.Handler) (events.Subscription, error) {
	sub, err := e.rt.conn.Subscribe(messaging.Live(topic), func(msg *natsgo.Msg) {
		var envelope messaging.Envelope
		var event events.Event
		if json.Unmarshal(msg.Data, &envelope) == nil && len(envelope.Payload) > 0 {
			if json.Unmarshal(envelope.Payload, &event) == nil {
				handler(ctx, event)
			}
		} else if json.Unmarshal(msg.Data, &event) == nil {
			handler(ctx, event)
		}
	})
	if err != nil {
		return nil, err
	}
	return eventSubscription{sub: sub}, nil
}

type eventSubscription struct {
	sub *natsgo.Subscription
}

func (s eventSubscription) Unsubscribe() error { return s.sub.Unsubscribe() }

var _ pubsub.Subscription = eventSubscription{}
