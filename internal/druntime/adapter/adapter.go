package adapter

import (
	"context"
	"fmt"

	"aether/internal/druntime"
	"aether/internal/druntime/adapter/memory"
	"aether/internal/druntime/adapter/redis"
)

func New(ctx context.Context, cfg druntime.Config) (*druntime.Runtime, error) {
	switch cfg.Backend {
	case "", "memory":
		return memory.New(ctx, cfg)
	case "redis":
		return redis.New(ctx, cfg)
	default:
		return nil, fmt.Errorf("runtime backend desconhecido: %q", cfg.Backend)
	}
}
