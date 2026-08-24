package nats

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"aether/internal/platform/messaging"
)

type State struct {
	rt *Runtime
	mu sync.Mutex
}

type stateValue struct {
	Value     []byte    `json:"value"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

func stateKey(key string) string {
	return strings.NewReplacer(":", "_", "*", "_", ">", "_").Replace(key)
}

func (s *State) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	entry := stateValue{Value: append([]byte(nil), value...)}
	if ttl > 0 {
		entry.ExpiresAt = time.Now().UTC().Add(ttl)
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = s.rt.state.Put(ctx, stateKey(key), raw)
	if err == nil {
		err = s.rt.conn.Publish(messaging.StatePrefix+stateKey(key), value)
	}
	return err
}

func (s *State) Get(ctx context.Context, key string) ([]byte, bool, error) {
	entry, err := s.rt.state.Get(ctx, stateKey(key))
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	var stored stateValue
	if err := json.Unmarshal(entry.Value(), &stored); err != nil {
		return append([]byte(nil), entry.Value()...), true, nil
	}
	if !stored.ExpiresAt.IsZero() && !time.Now().UTC().Before(stored.ExpiresAt) {
		_ = s.rt.state.Delete(ctx, stateKey(key))
		return nil, false, nil
	}
	return append([]byte(nil), stored.Value...), true, nil
}

func (s *State) Del(ctx context.Context, key string) error {
	if err := s.rt.state.Delete(ctx, stateKey(key)); err != nil {
		return err
	}
	return s.rt.conn.Publish(messaging.StatePrefix+stateKey(key), nil)
}

func (s *State) Changes(ctx context.Context, keyPattern string, handler func(context.Context, string, []byte)) (func(), error) {
	pattern := messaging.StatePrefix + strings.ReplaceAll(keyPattern, ":", "_")
	sub, err := s.rt.conn.Subscribe(pattern, func(msg *natsgo.Msg) {
		key := strings.TrimPrefix(msg.Subject, messaging.StatePrefix)
		handler(ctx, key, append([]byte(nil), msg.Data...))
	})
	if err != nil {
		return nil, err
	}
	return func() { _ = sub.Unsubscribe() }, nil
}
