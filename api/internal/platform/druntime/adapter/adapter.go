package adapter

import (
	"context"
	"fmt"

	natsgo "github.com/nats-io/nats.go"

	"aether/internal/platform/druntime"
	"aether/internal/platform/druntime/adapter/memory"
	"aether/internal/platform/druntime/adapter/nats"
)

func New(ctx context.Context, cfg druntime.Config) (*druntime.Runtime, error) {
	switch cfg.Backend {
	case "memory":
		return memory.New(ctx, cfg)
	case "", "nats":
		return nats.New(ctx, cfg)
	default:
		return nil, fmt.Errorf("runtime backend desconhecido: %q", cfg.Backend)
	}
}

func NewWithConn(ctx context.Context, cfg druntime.Config, conn *natsgo.Conn) (*druntime.Runtime, error) {
	switch cfg.Backend {
	case "memory":
		return memory.New(ctx, cfg)
	case "", "nats":
		return nats.NewWithConn(ctx, cfg, conn)
	default:
		return nil, fmt.Errorf("runtime backend desconhecido: %q", cfg.Backend)
	}
}
