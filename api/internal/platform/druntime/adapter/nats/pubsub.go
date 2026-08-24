package nats

import (
	"context"
	"sync"

	natsgo "github.com/nats-io/nats.go"

	"aether/internal/platform/druntime/pubsub"
)

type PubSub struct {
	rt   *Runtime
	mu   sync.Mutex
	subs map[string]int
}

func (p *PubSub) Publish(ctx context.Context, channel string, data []byte) error {
	if err := p.rt.conn.Publish(channel, data); err != nil {
		return err
	}
	return p.rt.conn.FlushWithContext(ctx)
}

func (p *PubSub) Subscribe(_ context.Context, channel string, handler pubsub.Handler, opts ...pubsub.Option) (pubsub.Subscription, error) {
	var options pubsub.SubscribeOptions
	for _, opt := range opts {
		opt(&options)
	}
	sub, err := p.rt.conn.Subscribe(channel, func(msg *natsgo.Msg) {
		handler(context.Background(), pubsub.Message{Channel: msg.Subject, Data: append([]byte(nil), msg.Data...)})
	})
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	if p.subs == nil {
		p.subs = map[string]int{}
	}
	p.subs[channel]++
	p.mu.Unlock()
	return &subscription{parent: p, channel: channel, sub: sub}, nil
}

func (p *PubSub) Subscribers(_ context.Context) (map[string]int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make(map[string]int, len(p.subs))
	for channel, count := range p.subs {
		out[channel] = count
	}
	return out, nil
}

type subscription struct {
	parent  *PubSub
	channel string
	sub     *natsgo.Subscription
	once    sync.Once
}

func (s *subscription) Unsubscribe() error {
	var err error
	s.once.Do(func() {
		err = s.sub.Unsubscribe()
		s.parent.mu.Lock()
		if s.parent.subs[s.channel] > 1 {
			s.parent.subs[s.channel]--
		} else {
			delete(s.parent.subs, s.channel)
		}
		s.parent.mu.Unlock()
	})
	return err
}
