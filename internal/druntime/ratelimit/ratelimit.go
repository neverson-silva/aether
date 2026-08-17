package ratelimit

import (
	"context"
	"time"
)

type Decision struct {
	Allowed   bool
	Remaining int
	ResetIn   time.Duration
}

type KeyTip string

const (
	TipUser    KeyTip = "user"
	TipIP      KeyTip = "ip"
	TipAPIKey  KeyTip = "api_key"
	TipOrg     KeyTip = "org"
	TipRoute   KeyTip = "route"
	TipWebhook KeyTip = "webhook"
	TipAgent   KeyTip = "agent"
)

type RateLimiter interface {
	Allow(ctx context.Context, tip KeyTip, key string, n int, rate float64, burst int) (Decision, error)
	Reset(ctx context.Context, tip KeyTip, key string) error
}
