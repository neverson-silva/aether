package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type Presence struct {
	rt *Runtime
}

func (p *Presence) scopeKey(scope string) string {
	return "pres:" + scope
}

func (p *Presence) Join(ctx context.Context, scope, member string, ttl time.Duration) error {
	key := p.scopeKey(scope)
	now := time.Now().UnixMilli()
	pipe := p.rt.client.TxPipeline()
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now + ttl.Milliseconds()), Member: member})
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(now, 10))
	_, err := pipe.Exec(ctx)
	p.rt.observe(err)
	return err
}

func (p *Presence) Leave(ctx context.Context, scope, member string) error {
	err := p.rt.client.ZRem(ctx, p.scopeKey(scope), member).Err()
	p.rt.observe(err)
	return err
}

func (p *Presence) Heartbeat(ctx context.Context, scope, member string, ttl time.Duration) error {
	return p.Join(ctx, scope, member, ttl)
}

func (p *Presence) Count(ctx context.Context, scope string) (int64, error) {
	now := time.Now().UnixMilli()
	key := p.scopeKey(scope)
	pipe := p.rt.client.TxPipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(now, 10))
	pipe.ZCard(ctx, key)
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		p.rt.observe(err)
		return 0, err
	}
	return cmds[1].(*redis.IntCmd).Val(), nil
}

func (p *Presence) Members(ctx context.Context, scope string) ([]string, error) {
	now := time.Now().UnixMilli()
	key := p.scopeKey(scope)
	pipe := p.rt.client.TxPipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(now, 10))
	pipe.ZRangeByScore(ctx, key, &redis.ZRangeBy{Min: strconv.FormatInt(now, 10), Max: "+inf"})
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		p.rt.observe(err)
		return nil, err
	}
	return cmds[1].(*redis.StringSliceCmd).Val(), nil
}
