package api

import (
	"context"
	"time"

	"aether/internal/druntime/cache"
)

const (
	cacheAppsTTL      = 3 * time.Second
	cacheTemplatesTTL = 60 * time.Second
)

func (s *Server) cacheGetJSON(key string, v any) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if jc, ok := s.core.RT.Cache.(cache.JSONCache); ok {
		hit, err := jc.GetJSON(ctx, key, v)
		return err == nil && hit
	}
	return false
}

func (s *Server) cacheSetJSON(key string, v any, ttl time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if jc, ok := s.core.RT.Cache.(cache.JSONCache); ok {
		_ = jc.SetJSON(ctx, key, v, ttl)
	}
}

func (s *Server) cacheInvalidate(prefix string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.core.RT.Cache.Invalidate(ctx, prefix)
}

func appsCacheKey(orgID string) string {
	return "cache:apps:" + orgID
}
