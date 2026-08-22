package redis

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/redis/go-redis/v9"

	"aether/internal/platform/druntime/pubsub"
)

type subscriber struct {
	ch   chan pubsub.Message
	drop atomic.Int64
}

type subKey struct {
	channel string
	s       *subscriber
}

type PubSub struct {
	rt   *Runtime
	mu   sync.Mutex
	subs map[string]*redis.PubSub
}

func (p *PubSub) Publish(ctx context.Context, channel string, data []byte) error {
	p.rt.met.events.Add(1)
	err := p.rt.client.Publish(ctx, channel, data).Err()
	p.rt.observe(err)
	return err
}

func (p *PubSub) Subscribe(ctx context.Context, channel string, h pubsub.Handler, opts ...pubsub.Option) (pubsub.Subscription, error) {
	var o pubsub.SubscribeOptions
	for _, opt := range opts {
		opt(&o)
	}
	if o.BufferSize <= 0 {
		o.BufferSize = 256
	}
	rd := p.rt.client.Subscribe(ctx, channel)
	if _, err := rd.Receive(ctx); err != nil {
		return nil, err
	}
	s := &subscriber{ch: make(chan pubsub.Message, o.BufferSize)}

	p.mu.Lock()
	if p.subs == nil {
		p.subs = map[string]*redis.PubSub{}
	}
	if p.subs[channel] == nil {
		p.subs[channel] = rd
	}
	p.mu.Unlock()

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			default:
			}
			msg, err := rd.Receive(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			switch m := msg.(type) {
			case *redis.Message:
				select {
				case s.ch <- pubsub.Message{Channel: m.Channel, Data: []byte(m.Payload)}:
				default:
					s.drop.Add(1)
				}
			}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case m := <-s.ch:
				h(ctx, m)
			}
		}
	}()

	var once sync.Once
	return &sub{p: p, channel: channel, rd: rd, done: done, wg: &wg, once: &once}, nil
}

type sub struct {
	p       *PubSub
	channel string
	rd      *redis.PubSub
	done    chan struct{}
	wg      *sync.WaitGroup
	once    *sync.Once
}

func (s *sub) Unsubscribe() error {
	s.once.Do(func() { close(s.done) })
	s.p.mu.Lock()
	if s.p.subs[s.channel] == s.rd {
		delete(s.p.subs, s.channel)
	}
	s.p.mu.Unlock()
	_ = s.rd.Close()
	s.wg.Wait()
	return nil
}

func (p *PubSub) Subscribers(ctx context.Context) (map[string]int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make(map[string]int, len(p.subs))
	if len(p.subs) == 0 {
		return out, nil
	}
	channels := make([]string, 0, len(p.subs))
	for ch := range p.subs {
		channels = append(channels, ch)
	}
	counts, err := p.rt.client.PubSubNumSub(ctx, channels...).Result()
	if err != nil {
		p.rt.observe(err)
		return nil, err
	}
	for ch := range p.subs {
		if n, ok := counts[ch]; ok {
			out[ch] = int(n)
		} else {
			out[ch] = 0
		}
	}
	return out, nil
}
