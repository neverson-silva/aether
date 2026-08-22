package memory

import (
	"context"
	"encoding/json"

	"aether/internal/platform/druntime/events"
	"aether/internal/platform/druntime/pubsub"
)

type EventBus struct {
	pubsub *PubSub
}

func NewEventBus(ps *PubSub) *EventBus {
	return &EventBus{pubsub: ps}
}

func (e *EventBus) Publish(ctx context.Context, topic string, ev events.Event) error {
	raw, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return e.pubsub.Publish(ctx, "ev:"+topic, raw)
}

func (e *EventBus) Subscribe(ctx context.Context, topic string, h events.Handler) (events.Subscription, error) {
	return e.pubsub.Subscribe(ctx, "ev:"+topic, func(ctx context.Context, msg pubsub.Message) {
		var ev events.Event
		if err := json.Unmarshal(msg.Data, &ev); err == nil {
			h(ctx, ev)
		}
	})
}
