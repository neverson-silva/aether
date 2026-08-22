package druntime_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"aether/internal/platform/druntime"
	"aether/internal/platform/druntime/adapter"
	"aether/internal/platform/druntime/events"
	"aether/internal/platform/druntime/locks"
	"aether/internal/platform/druntime/pubsub"
	"aether/internal/platform/druntime/queue"
	"aether/internal/platform/druntime/ratelimit"
)

func newRT(t *testing.T, backend string) *druntime.Runtime {
	t.Helper()
	cfg := druntime.Config{Backend: backend}
	if backend == "redis" {
		cfg.RedisAddr = "127.0.0.1:6380"
	}
	rt, err := adapter.New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("adapter %s: %v", backend, err)
	}
	t.Cleanup(func() { rt.Close(context.Background()) })
	return rt
}

func TestPubSubFanout(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			rt := newRT(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var mu sync.Mutex
			var got []string
			sub, err := rt.PubSub.Subscribe(ctx, "logs:deployment:1", func(_ context.Context, m pubsub.Message) {
				mu.Lock()
				got = append(got, string(m.Data))
				mu.Unlock()
			})
			if err != nil {
				t.Fatal(err)
			}
			defer sub.Unsubscribe()
			for i := 0; i < 5; i++ {
				if err := rt.PubSub.Publish(ctx, "logs:deployment:1", []byte("linha"+string(rune('0'+i)))); err != nil {
					t.Fatal(err)
				}
			}
			deadline := time.Now().Add(2 * time.Second)
			for len(func() []string { mu.Lock(); defer mu.Unlock(); return got }()) < 5 && time.Now().Before(deadline) {
				time.Sleep(10 * time.Millisecond)
			}
			mu.Lock()
			defer mu.Unlock()
			if len(got) != 5 {
				t.Fatalf("esperado 5 msgs, got %d: %v", len(got), got)
			}

			if err := rt.PubSub.Publish(ctx, "logs:deployment:999", []byte("x")); err != nil {
				t.Fatal(err)
			}
			time.Sleep(100 * time.Millisecond)
			if len(got) != 5 {
				t.Fatalf("canal errado recebeu msg: %v", got)
			}
		})
	}
}

func TestCacheTTLAndInvalidate(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			rt := newRT(t, backend)
			ctx := context.Background()
			if err := rt.Cache.Set(ctx, "k:1", []byte("v1"), 0); err != nil {
				t.Fatal(err)
			}
			v, ok, err := rt.Cache.Get(ctx, "k:1")
			if err != nil || !ok || string(v) != "v1" {
				t.Fatalf("get: %v %v %q", ok, err, v)
			}
			if err := rt.Cache.Invalidate(ctx, "k:"); err != nil {
				t.Fatal(err)
			}
			if _, ok, _ := rt.Cache.Get(ctx, "k:1"); ok {
				t.Fatal("invalidate falhou")
			}

			rt.Cache.Set(ctx, "ttl:1", []byte("v"), 50*time.Millisecond)
			time.Sleep(120 * time.Millisecond)
			if _, ok, _ := rt.Cache.Get(ctx, "ttl:1"); ok {
				t.Fatal("ttl não expirou")
			}
		})
	}
}

func TestLockFencingAndExclusion(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			rt := newRT(t, backend)
			ctx := context.Background()
			l1, ok, err := rt.Locks.Acquire(ctx, "deploy:app:1", time.Second)
			if err != nil || !ok {
				t.Fatalf("acquire 1: %v %v", ok, err)
			}
			_, ok2, _ := rt.Locks.Acquire(ctx, "deploy:app:1", time.Second)
			if ok2 {
				t.Fatal("lock exclusivo violado")
			}
			if locked, _ := rt.Locks.Locked(ctx, "deploy:app:1"); !locked {
				t.Fatal("lock deveria estar retido")
			}
			if err := rt.Locks.Release(ctx, l1); err != nil {
				t.Fatal(err)
			}
			l2, ok3, _ := rt.Locks.Acquire(ctx, "deploy:app:1", time.Second)
			if !ok3 {
				t.Fatal("reacquire falhou")
			}
			if l1.Token == l2.Token {
				t.Fatal("fencing token deveria mudar")
			}

			if err := rt.Locks.Release(ctx, l1); err != locks.ErrLockNotOwned {
				t.Fatalf("release token antigo: %v", err)
			}
			_ = rt.Locks.Release(ctx, l2)
		})
	}
}

func TestLockTTLExpires(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			rt := newRT(t, backend)
			ctx := context.Background()
			l, ok, _ := rt.Locks.Acquire(ctx, "lock:ttl", 80*time.Millisecond)
			if !ok {
				t.Fatal("acquire")
			}
			time.Sleep(150 * time.Millisecond)
			if locked, _ := rt.Locks.Locked(ctx, "lock:ttl"); locked {
				t.Fatal("lock deveria ter expirado")
			}
			if _, ok, _ := rt.Locks.Acquire(ctx, "lock:ttl", time.Second); !ok {
				t.Fatal("reacquire após expiração")
			}
			_ = l
		})
	}
}

func TestRateLimitTokenBucket(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			rt := newRT(t, backend)
			ctx := context.Background()

			var allowed int
			for i := 0; i < 4; i++ {
				d, err := rt.RateLimit.Allow(ctx, ratelimit.TipIP, "1.2.3.4", 1, 1, 3)
				if err != nil {
					t.Fatal(err)
				}
				if d.Allowed {
					allowed++
				}
			}
			if allowed != 3 {
				t.Fatalf("esperado 3 allowed, got %d", allowed)
			}
			if err := rt.RateLimit.Reset(ctx, ratelimit.TipIP, "1.2.3.4"); err != nil {
				t.Fatal(err)
			}
			d, _ := rt.RateLimit.Allow(ctx, ratelimit.TipIP, "1.2.3.4", 1, 1, 3)
			if !d.Allowed {
				t.Fatal("reset não limpou bucket")
			}
		})
	}
}

func TestPresence(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			rt := newRT(t, backend)
			ctx := context.Background()
			rt.Presence.Join(ctx, "deployment:1", "user-a", time.Second)
			rt.Presence.Join(ctx, "deployment:1", "user-b", time.Second)
			n, _ := rt.Presence.Count(ctx, "deployment:1")
			if n != 2 {
				t.Fatalf("count: %d", n)
			}
			rt.Presence.Leave(ctx, "deployment:1", "user-a")
			members, _ := rt.Presence.Members(ctx, "deployment:1")
			if len(members) != 1 || members[0] != "user-b" {
				t.Fatalf("members: %v", members)
			}

			rt.Presence.Join(ctx, "dep:ttl", "u", 80*time.Millisecond)
			time.Sleep(150 * time.Millisecond)
			if n, _ := rt.Presence.Count(ctx, "dep:ttl"); n != 0 {
				t.Fatalf("presença deveria expirar: %d", n)
			}
		})
	}
}

func TestEventBus(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			rt := newRT(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			got := make(chan events.Event, 4)
			sub, err := rt.Events.Subscribe(ctx, "deployment", func(_ context.Context, ev events.Event) {
				got <- ev
			})
			if err != nil {
				t.Fatal(err)
			}
			defer sub.Unsubscribe()
			rt.Events.Publish(ctx, "deployment", events.Event{
				ID: "e1", Type: "deployment.started", AggregateType: "deployment",
				AggregateID: "dep1", Payload: []byte(`{"n":1}`), TS: time.Now().UTC(),
			})
			select {
			case ev := <-got:
				if ev.Type != "deployment.started" || ev.AggregateID != "dep1" {
					t.Fatalf("evento errado: %+v", ev)
				}
				var p map[string]any
				json.Unmarshal(ev.Payload, &p)
				if p["n"].(float64) != 1 {
					t.Fatalf("payload: %v", p)
				}
			case <-ctx.Done():
				t.Fatal("evento não entregue")
			}
		})
	}
}

func TestRuntimeState(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			rt := newRT(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			changes := make(chan string, 4)
			cancelWatch, err := rt.State.Changes(ctx, "deploy:*", func(_ context.Context, key string, value []byte) {
				changes <- key + "=" + string(value)
			})
			if err != nil {
				t.Fatal(err)
			}
			defer cancelWatch()
			rt.State.Set(ctx, "deploy:app:1", []byte("building"), 0)
			select {
			case got := <-changes:
				if got != "deploy:app:1=building" {
					t.Fatalf("change: %s", got)
				}
			case <-ctx.Done():
				t.Fatal("mudança não entregue")
			}
			if v, ok, _ := rt.State.Get(ctx, "deploy:app:1"); !ok || string(v) != "building" {
				t.Fatalf("state: %q %v", v, ok)
			}
			rt.State.Set(ctx, "deploy:app:1", []byte("healthy"), 0)
			select {
			case got := <-changes:
				if got != "deploy:app:1=healthy" {
					t.Fatalf("change2: %s", got)
				}
			case <-ctx.Done():
				t.Fatal("mudança2 não entregue")
			}
			rt.State.Del(ctx, "deploy:app:1")
			if _, ok, _ := rt.State.Get(ctx, "deploy:app:1"); ok {
				t.Fatal("del não removeu")
			}
		})
	}
}

func TestQueueEnqueueAck(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			rt := newRT(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			rt.Queue.Enqueue(ctx, "deploys", queue.Job{
				Type: "deploy", Priority: 0, Payload: []byte("dep1"),
				AppID: "app1", DeploymentID: "dep1",
			})
			rt.Queue.Enqueue(ctx, "deploys", queue.Job{
				Type: "deploy", Priority: 9, Payload: []byte("dep-high"),
				AppID: "app1", DeploymentID: "dep-high",
			})
			rt.Queue.Enqueue(ctx, "deploys", queue.Job{
				Type: "deploy", Priority: 0, Payload: []byte("dep2"),
				AppID: "app1", DeploymentID: "dep2",
			})
			c, err := rt.Queue.NewConsumer(ctx, "deploys", "workers", "w1")
			if err != nil {
				t.Fatal(err)
			}

			j1, err := c.Next(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if string(j1.Payload) != "dep-high" {
				t.Fatalf("prioridade: %s", j1.Payload)
			}
			j2, _ := c.Next(ctx)
			j3, _ := c.Next(ctx)
			if string(j2.Payload) != "dep1" || string(j3.Payload) != "dep2" {
				t.Fatalf("ordem: %s %s", j2.Payload, j3.Payload)
			}
			if err := c.Ack(ctx, j1); err != nil {
				t.Fatal(err)
			}
			if err := c.Ack(ctx, j2); err != nil {
				t.Fatal(err)
			}
			if err := c.Ack(ctx, j3); err != nil {
				t.Fatal(err)
			}
			time.Sleep(100 * time.Millisecond)
			if n, _ := rt.Queue.Len(ctx, "deploys"); n != 0 {
				t.Fatalf("len após ack: %d", n)
			}
		})
	}
}

func TestQueueNackDeadLetter(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			rt := newRT(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			rt.Queue.Enqueue(ctx, "jobs", queue.Job{Type: "x", Payload: []byte("boom")})
			c, _ := rt.Queue.NewConsumer(ctx, "jobs", "workers", "w1")
			for attempt := 0; attempt < 3; attempt++ {
				j, err := c.Next(ctx)
				if err != nil {
					t.Fatal(err)
				}
				if err := c.Nack(ctx, j); err != nil {
					t.Fatal(err)
				}
			}
			time.Sleep(150 * time.Millisecond)
			dlq, _ := rt.Queue.DeadLetterLen(ctx, "jobs")
			if dlq == 0 {
				t.Fatal("job não foi para dead-letter")
			}
			if n, _ := rt.Queue.Len(ctx, "jobs"); n != 0 {
				t.Fatalf("len após dlq: %d", n)
			}
		})
	}
}

func TestQueueCancel(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			rt := newRT(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			rt.Queue.Enqueue(ctx, "jobs", queue.Job{ID: "job-x", Type: "x", Payload: []byte("x")})
			if err := rt.Queue.Cancel(ctx, "jobs", "job-x"); err != nil {
				t.Fatal(err)
			}
			time.Sleep(100 * time.Millisecond)
			if n, _ := rt.Queue.Len(ctx, "jobs"); n != 0 {
				t.Fatalf("cancel não removeu: %d", n)
			}
		})
	}
}

func TestQueueConcurrentConsumersNoDoubleDelivery(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			rt := newRT(t, backend)
			wctx, wcancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer wcancel()
			ctx, cancel := context.WithCancel(wctx)
			defer cancel()
			const total = 20
			for i := 0; i < total; i++ {
				rt.Queue.Enqueue(ctx, "batch", queue.Job{Type: "t", Payload: []byte(itoa(i))})
			}
			var mu sync.Mutex
			seen := map[string]bool{}
			var wg sync.WaitGroup
			consumers := make([]queue.Consumer, 4)
			for w := 0; w < 4; w++ {
				wg.Add(1)
				go func(w int) {
					defer wg.Done()
					c, err := rt.Queue.NewConsumer(ctx, "batch", "workers", itoa(w))
					if err != nil {
						return
					}
					consumers[w] = c
					defer c.Close()
					for {
						j, err := c.Next(ctx)
						if err != nil {
							return
						}
						mu.Lock()
						if seen[string(j.Payload)] {
							mu.Unlock()
							t.Error("job entregue duas vezes: " + string(j.Payload))
							return
						}
						seen[string(j.Payload)] = true
						n := len(seen)
						mu.Unlock()
						if err := c.Ack(ctx, j); err != nil {
							return
						}
						if n == total {
							cancel()
							return
						}
					}
				}(w)
			}
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(30 * time.Second):
				mu.Lock()
				n := len(seen)
				mu.Unlock()
				t.Fatalf("timeout, processados %d/%d", n, total)
			}
			mu.Lock()
			defer mu.Unlock()
			if len(seen) != total {
				t.Fatalf("processados %d/%d", len(seen), total)
			}
		})
	}
}

func TestRunOnceIdempotency(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			rt := newRT(t, backend)
			ctx := context.Background()
			calls := 0
			ran1, err := rt.RunOnce(ctx, "idem:webhook:app1:deliv1", time.Minute, func() error {
				calls++
				return nil
			})
			if err != nil || !ran1 || calls != 1 {
				t.Fatalf("primeira execução: ran=%v calls=%d err=%v", ran1, calls, err)
			}
			ran2, _ := rt.RunOnce(ctx, "idem:webhook:app1:deliv1", time.Minute, func() error {
				calls++
				return nil
			})
			if ran2 || calls != 1 {
				t.Fatalf("segunda execução deveria ser ignorada: ran=%v calls=%d", ran2, calls)
			}
		})
	}
}

func TestCacheAddExclusive(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			rt := newRT(t, backend)
			ctx := context.Background()
			ok1, _ := rt.Cache.Add(ctx, "setnx:1", []byte("x"), time.Minute)
			ok2, _ := rt.Cache.Add(ctx, "setnx:1", []byte("y"), time.Minute)
			if !ok1 || ok2 {
				t.Fatalf("Add exclusivo falhou: %v %v", ok1, ok2)
			}
			if err := rt.Cache.Del(ctx, "setnx:1"); err != nil {
				t.Fatal(err)
			}
			ok3, _ := rt.Cache.Add(ctx, "setnx:1", []byte("z"), time.Minute)
			if !ok3 {
				t.Fatal("Add após Del deveria funcionar")
			}
		})
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
