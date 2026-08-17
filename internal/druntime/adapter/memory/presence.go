package memory

import (
	"context"
	"sync"
	"time"
)

type Presence struct {
	mu     sync.Mutex
	scopes map[string]map[string]time.Time
}

func NewPresence() *Presence {
	return &Presence{scopes: map[string]map[string]time.Time{}}
}

func (p *Presence) gc(now time.Time) {
	for scope, members := range p.scopes {
		for m, exp := range members {
			if now.After(exp) {
				delete(members, m)
			}
		}
		if len(members) == 0 {
			delete(p.scopes, scope)
		}
	}
}

func (p *Presence) Join(_ context.Context, scope, member string, ttl time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gc(time.Now())
	if p.scopes[scope] == nil {
		p.scopes[scope] = map[string]time.Time{}
	}
	p.scopes[scope][member] = time.Now().Add(ttl)
	return nil
}

func (p *Presence) Leave(_ context.Context, scope, member string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.scopes[scope] != nil {
		delete(p.scopes[scope], member)
		if len(p.scopes[scope]) == 0 {
			delete(p.scopes, scope)
		}
	}
	return nil
}

func (p *Presence) Heartbeat(_ context.Context, scope, member string, ttl time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.scopes[scope] == nil {
		p.scopes[scope] = map[string]time.Time{}
	}
	p.scopes[scope][member] = time.Now().Add(ttl)
	return nil
}

func (p *Presence) Count(_ context.Context, scope string) (int64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gc(time.Now())
	return int64(len(p.scopes[scope])), nil
}

func (p *Presence) Members(_ context.Context, scope string) ([]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gc(time.Now())
	out := make([]string, 0, len(p.scopes[scope]))
	for m := range p.scopes[scope] {
		out = append(out, m)
	}
	return out, nil
}
