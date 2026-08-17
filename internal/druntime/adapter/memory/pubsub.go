package memory

import (
	"context"
	"sync"
	"sync/atomic"

	"aether/internal/druntime/pubsub"
)

type subscriber struct {
	ch   chan pubsub.Message
	drop atomic.Int64
}

type PubSub struct {
	mu   sync.RWMutex
	subs map[string]map[*subscriber]struct{}
}

func NewPubSub() *PubSub {
	return &PubSub{subs: map[string]map[*subscriber]struct{}{}}
}

func (p *PubSub) Publish(_ context.Context, channel string, data []byte) error {
	p.mu.RLock()
	list := p.subs[channel]
	for s := range list {
		select {
		case s.ch <- pubsub.Message{Channel: channel, Data: data}:
		default:
			s.drop.Add(1)
		}
	}
	p.mu.RUnlock()
	return nil
}

type subscription struct {
	p       *PubSub
	channel string
	s       *subscriber
}

func (s *subscription) Unsubscribe() error {
	s.p.mu.Lock()
	defer s.p.mu.Unlock()
	list := s.p.subs[s.channel]
	delete(list, s.s)
	if len(list) == 0 {
		delete(s.p.subs, s.channel)
	}
	return nil
}

type funcSub struct {
	unsubd bool
	unsub  func() error
}

func (f *funcSub) Unsubscribe() error {
	if f.unsubd {
		return nil
	}
	f.unsubd = true
	return f.unsub()
}

func (p *PubSub) Subscribe(ctx context.Context, channel string, h pubsub.Handler, opts ...pubsub.Option) (pubsub.Subscription, error) {
	var o pubsub.SubscribeOptions
	for _, opt := range opts {
		opt(&o)
	}
	if o.BufferSize <= 0 {
		o.BufferSize = 256
	}
	s := &subscriber{ch: make(chan pubsub.Message, o.BufferSize)}
	p.mu.Lock()
	if p.subs[channel] == nil {
		p.subs[channel] = map[*subscriber]struct{}{}
	}
	p.subs[channel][s] = struct{}{}
	p.mu.Unlock()

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			case <-ctx.Done():
				return
			case m := <-s.ch:
				h(ctx, m)
			}
		}
	}()
	sub := &subscription{p: p, channel: channel, s: s}
	var once sync.Once
	return &funcSub{unsub: func() error {
		once.Do(func() { close(stop) })
		_ = sub.Unsubscribe()
		wg.Wait()
		return nil
	}}, nil
}

func (p *PubSub) Subscribers(_ context.Context) (map[string]int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make(map[string]int, len(p.subs))
	for ch, list := range p.subs {
		out[ch] = len(list)
	}
	return out, nil
}
