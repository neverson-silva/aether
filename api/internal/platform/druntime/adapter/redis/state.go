package redis

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type State struct {
	rt *Runtime
}

func (s *State) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	var err error
	if ttl > 0 {
		err = s.rt.client.Set(ctx, "st:"+key, value, ttl).Err()
	} else {
		err = s.rt.client.Set(ctx, "st:"+key, value, 0).Err()
	}
	if err != nil {
		s.rt.observe(err)
		return err
	}
	s.rt.client.Publish(ctx, "stch:"+key, value)
	return nil
}

func (s *State) Get(ctx context.Context, key string) ([]byte, bool, error) {
	v, err := s.rt.client.Get(ctx, "st:"+key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		s.rt.observe(err)
		return nil, false, err
	}
	return v, true, nil
}

func (s *State) Del(ctx context.Context, key string) error {
	err := s.rt.client.Del(ctx, "st:"+key).Err()
	s.rt.observe(err)
	if err == nil {
		s.rt.client.Publish(ctx, "stch:"+key, nil)
	}
	return err
}

func (s *State) Changes(ctx context.Context, keyPattern string, h func(ctx context.Context, key string, value []byte)) (func(), error) {
	sub := s.rt.client.PSubscribe(ctx, "stch:"+keyPattern)
	go func() {
		for {
			select {
			case <-ctx.Done():
				_ = sub.Close()
				return
			default:
			}
			msg, err := sub.Receive(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			if pm, ok := msg.(*redis.Message); ok {
				key := pm.Channel[len("stch:"):]
				h(ctx, key, []byte(pm.Payload))
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			_ = sub.Close()
		})
	}, nil
}
