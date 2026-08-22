package druntime_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6380"})
	_ = client.FlushAll(ctx)
	_ = client.Close()
	cancel()
	os.Exit(m.Run())
}
