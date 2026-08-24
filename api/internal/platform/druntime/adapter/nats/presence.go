package nats

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"aether/internal/platform/druntime/presence"
)

type presenceValue struct {
	Member  string    `json:"member"`
	Expires time.Time `json:"expires"`
}

type Presence struct {
	rt *Runtime
}

func presenceKey(scope, member string) string {
	return strings.NewReplacer(":", "_", "/", "_", " ", "_").Replace(scope + "_" + member)
}

func (p *Presence) Join(ctx context.Context, scope, member string, ttl time.Duration) error {
	raw, err := json.Marshal(presenceValue{Member: member, Expires: time.Now().Add(ttl)})
	if err != nil {
		return err
	}
	_, err = p.rt.state.Put(ctx, "presence."+presenceKey(scope, member), raw)
	return err
}

func (p *Presence) Leave(ctx context.Context, scope, member string) error {
	return p.rt.state.Delete(ctx, "presence."+presenceKey(scope, member))
}

func (p *Presence) Heartbeat(ctx context.Context, scope, member string, ttl time.Duration) error {
	return p.Join(ctx, scope, member, ttl)
}

func (p *Presence) Members(ctx context.Context, scope string) ([]string, error) {
	keys, err := p.rt.state.Keys(ctx)
	if err != nil {
		return nil, err
	}
	prefix := "presence." + strings.NewReplacer(":", "_", "/", "_", " ", "_").Replace(scope) + "_"
	now := time.Now()
	members := make([]string, 0)
	for _, key := range keys {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		entry, err := p.rt.state.Get(ctx, key)
		if err != nil {
			continue
		}
		var value presenceValue
		if json.Unmarshal(entry.Value(), &value) != nil || now.After(value.Expires) {
			_ = p.rt.state.Delete(ctx, key)
			continue
		}
		members = append(members, value.Member)
	}
	return members, nil
}

func (p *Presence) Count(ctx context.Context, scope string) (int64, error) {
	members, err := p.Members(ctx, scope)
	return int64(len(members)), err
}

var _ presence.Presence = (*Presence)(nil)
