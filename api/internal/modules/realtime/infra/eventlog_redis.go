package infra

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"aether/internal/modules/realtime/domain"
	"aether/internal/platform/druntime"
)

const (
	eventStreamPrefix = "ev:org:"
	eventSeqPrefix    = "ev:seq:"
	eventMaxLen       = 5000
)

type RedisEventLog struct {
	client *redis.Client
}

func NewRedisEventLog(cfg druntime.Config) (*RedisEventLog, error) {
	addr := cfg.RedisAddr
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
		Username: cfg.RedisUsername,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return &RedisEventLog{client: client}, nil
}

func (l *RedisEventLog) Close() error {
	return l.client.Close()
}

func (l *RedisEventLog) Append(ctx context.Context, orgID uuid.UUID, ev domain.Event) (int64, error) {
	seq, err := l.client.Incr(ctx, eventSeqPrefix+orgID.String()).Result()
	if err != nil {
		return 0, err
	}
	ev.Seq = seq
	data, err := json.Marshal(ev)
	if err != nil {
		return 0, err
	}
	return seq, l.client.XAdd(ctx, &redis.XAddArgs{
		Stream: eventStreamPrefix + orgID.String(),
		MaxLen: eventMaxLen,
		Approx: true,
		Values: map[string]interface{}{"data": string(data)},
	}).Err()
}

func (l *RedisEventLog) Recent(ctx context.Context, orgID uuid.UUID, limit int) ([]domain.Event, error) {
	if limit <= 0 {
		limit = 20
	}
	msgs, err := l.client.XRevRangeN(ctx, eventStreamPrefix+orgID.String(), "+", "-", int64(limit)).Result()
	if err != nil {
		return nil, err
	}
	out := make([]domain.Event, 0, len(msgs))
	for i := len(msgs) - 1; i >= 0; i-- {
		ev, ok := parseStreamMessage(msgs[i])
		if ok {
			out = append(out, ev)
		}
	}
	return out, nil
}

func (l *RedisEventLog) Replay(ctx context.Context, orgID uuid.UUID, afterSeq int64, limit int) ([]domain.Event, error) {
	if limit <= 0 {
		limit = 50
	}
	count := int64(limit*2 + 200)
	if count > eventMaxLen {
		count = eventMaxLen
	}
	msgs, err := l.client.XRangeN(ctx, eventStreamPrefix+orgID.String(), "-", "+", count).Result()
	if err != nil {
		return nil, err
	}
	out := make([]domain.Event, 0, limit)
	for _, msg := range msgs {
		ev, ok := parseStreamMessage(msg)
		if !ok || ev.Seq <= afterSeq {
			continue
		}
		out = append(out, ev)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func parseStreamMessage(msg redis.XMessage) (domain.Event, bool) {
	var ev domain.Event
	raw, ok := msg.Values["data"].(string)
	if !ok {
		return ev, false
	}
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		return ev, false
	}
	return ev, true
}
