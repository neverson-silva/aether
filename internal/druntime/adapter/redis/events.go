package redis

import (
	"context"
	"encoding/json"

	"aether/internal/druntime/events"
	"aether/internal/druntime/pubsub"
)

type EventBus struct {
	rt *Runtime
}

func (e *EventBus) Publish(ctx context.Context, topic string, ev events.Event) error {
	raw, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return e.rt.client.Publish(ctx, "ev:"+topic, raw).Err()
}

func (e *EventBus) Subscribe(ctx context.Context, topic string, h events.Handler) (events.Subscription, error) {
	return e.rt.pubsub.Subscribe(ctx, "ev:"+topic, func(ctx context.Context, m pubsub.Message) {
		var ev events.Event
		if err := json.Unmarshal(m.Data, &ev); err == nil {
			h(ctx, ev)
		}
	})
}
