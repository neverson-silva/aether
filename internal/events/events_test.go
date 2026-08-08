package events

import (
	"context"
	"sync"
	"testing"
	"time"

	"aether/internal/db"
)

func testBus(t *testing.T) *Bus {
	sqldb := db.OpenTest(t)
	t.Cleanup(func() { sqldb.Close() })
	return NewBus(sqldb)
}

func TestPublishOrderingAndTimeline(t *testing.T) {
	b := testBus(t)
	ctx := context.Background()
	var got []string
	var mu sync.Mutex
	b.Subscribe(func(_ context.Context, e Event) {
		mu.Lock()
		got = append(got, e.Type)
		mu.Unlock()
	})
	for i := 0; i < 5; i++ {
		if err := b.Publish(ctx, "deployment", "dep1", "deployment.step", map[string]int{"i": i}, nil); err != nil {
			t.Fatal(err)
		}
	}
	events, err := b.Timeline(ctx, "deployment", "dep1")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 5 {
		t.Fatalf("esperado 5 eventos, obtido %d", len(events))
	}
	for i, e := range events {
		if e.Sequence != int64(i+1) {
			t.Fatalf("sequência divergiu: %d", e.Sequence)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 5 {
		t.Fatalf("handlers deveriam ter recebido 5, obtido %d", len(got))
	}
}

func TestOutboxReplay(t *testing.T) {
	sqldb := db.OpenTest(t)
	t.Cleanup(func() { sqldb.Close() })
	b := NewBus(sqldb)
	ctx := context.Background()
	if err := b.Publish(ctx, "app", "a1", "app.created", map[string]string{}, nil); err != nil {
		t.Fatal(err)
	}
	// simula crash: insere evento não-publicado direto
	if _, err := sqldb.Exec(`INSERT INTO events(aggregate_type,aggregate_id,sequence,type,payload,ts,published) VALUES('app','a1',2,'app.updated','{}',1,0)`); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var seen []string
	b.Subscribe(func(_ context.Context, e Event) {
		mu.Lock()
		seen = append(seen, e.Type)
		mu.Unlock()
	})
	b2 := NewBus(sqldb)
	b2.Subscribe(func(_ context.Context, e Event) {
		mu.Lock()
		seen = append(seen, e.Type)
		mu.Unlock()
	})
	n, err := b2.ReplayUnpublished(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("esperado 1 replay, obtido %d", n)
	}
	mu.Lock()
	defer mu.Unlock()
	found := false
	for _, s := range seen {
		if s == "app.updated" {
			found = true
		}
	}
	if !found {
		t.Fatal("evento não-publicado não foi entregue no replay")
	}
	// segunda execução do replay não deve reentregar
	seen = nil
	b3 := NewBus(sqldb)
	b3.Subscribe(func(_ context.Context, e Event) {
		mu.Lock()
		seen = append(seen, e.Type)
		mu.Unlock()
	})
	n, err = b3.ReplayUnpublished(ctx)
	if err != nil || n != 0 {
		t.Fatalf("replay deveria ser vazio: %d %v", n, err)
	}
}

func TestGCRemovesOldPublished(t *testing.T) {
	b := testBus(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if err := b.Publish(ctx, "app", "x", "app.event", map[string]int{"i": i}, nil); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(5 * time.Millisecond)
	n, err := b.GC(ctx, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("esperado 3 removidos, obtido %d", n)
	}
}
