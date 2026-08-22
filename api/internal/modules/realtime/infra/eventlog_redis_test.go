package infra

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/realtime/domain"
	"aether/internal/platform/druntime"
)

func newRedisEventLog(t *testing.T) *RedisEventLog {
	t.Helper()
	log, err := NewRedisEventLog(druntime.Config{RedisAddr: "127.0.0.1:6380"})
	if err != nil {
		t.Fatalf("new redis eventlog: %v", err)
	}
	t.Cleanup(func() { _ = log.Close() })
	return log
}

func TestRedisEventLogAppendRecentReplay(t *testing.T) {
	log := newRedisEventLog(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	org := uuid.New()
	other := uuid.New()

	base := domain.Event{Type: "deploy.created", Aggregate: "deployment", ResourceType: "deployment"}
	var lastSeq int64
	for i := 0; i < 3; i++ {
		seq, err := log.Append(ctx, org, base)
		if err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
		if seq != int64(i+1) {
			t.Fatalf("seq esperado %d, got %d", i+1, seq)
		}
		lastSeq = seq
	}
	if _, err := log.Append(ctx, other, base); err != nil {
		t.Fatalf("append outra org: %v", err)
	}

	recent, err := log.Recent(ctx, org, 10)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(recent) != 3 {
		t.Fatalf("recent esperado 3, got %d", len(recent))
	}
	if recent[0].Seq != 1 || recent[2].Seq != 3 {
		t.Fatalf("ordem recent: %+v", recent)
	}

	replay, err := log.Replay(ctx, org, 1, 10)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(replay) != 2 || replay[0].Seq != 2 || replay[1].Seq != 3 {
		t.Fatalf("replay: %+v", replay)
	}

	recentOther, err := log.Recent(ctx, other, 10)
	if err != nil {
		t.Fatalf("recent other: %v", err)
	}
	if len(recentOther) != 1 {
		t.Fatalf("isolamento de org quebrado: %d", len(recentOther))
	}
	_ = lastSeq
}
