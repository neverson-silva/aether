package outbox

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"aether/internal/platform/database"
	"aether/internal/platform/druntime"
	"aether/internal/platform/druntime/adapter"
	"aether/internal/platform/druntime/events"
	"aether/internal/platform/druntime/queue"
)

func TestStoreLifecycle(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	store := NewStore(pool)
	ctx := context.Background()
	eventID := uuid.New()
	event := events.Event{ID: eventID.String(), Type: "deployment.queued", AggregateType: "deployment", AggregateID: uuid.NewString(), Payload: []byte(`{"job":"deploy"}`), TS: time.Now().UTC()}
	if err := store.Enqueue(ctx, event, "deployments"); err != nil {
		t.Fatal(err)
	}
	if err := store.Enqueue(ctx, event, "deployments"); err != nil {
		t.Fatal(err)
	}
	items, err := store.Claim(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != eventID || items[0].Topic != "deployments" {
		t.Fatalf("unexpected claimed items: %+v", items)
	}
	if err := store.MarkPublished(ctx, eventID); err != nil {
		t.Fatal(err)
	}
	items, err = store.Claim(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("published event was claimed again: %+v", items)
	}

	retryID := uuid.New()
	retryEvent := events.Event{ID: retryID.String(), Type: "retry", AggregateType: "job", AggregateID: uuid.NewString(), Payload: []byte(`{}`), TS: time.Now().UTC()}
	if err := store.Enqueue(ctx, retryEvent, "jobs"); err != nil {
		t.Fatal(err)
	}
	items, err = store.Claim(ctx, 10)
	if err != nil || len(items) != 1 {
		t.Fatalf("claim retry event: items=%+v err=%v", items, err)
	}
	if err := store.Retry(ctx, retryID, time.Hour); err != nil {
		t.Fatal(err)
	}
	items, err = store.Claim(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("retry event should not be immediately available: %+v", items)
	}
}

func TestRetryDelayBackoff(t *testing.T) {
	want := []time.Duration{5 * time.Second, 15 * time.Second, 30 * time.Second, time.Minute, 5 * time.Minute}
	for attempt, expected := range want {
		if got := retryDelay(attempt + 1); got != expected {
			t.Fatalf("attempt %d: got %s want %s", attempt+1, got, expected)
		}
	}
	if got := retryDelay(100); got != want[len(want)-1] {
		t.Fatalf("maximum retry delay: got %s want %s", got, want[len(want)-1])
	}
}

type jobQueue struct {
	jobs []queue.Job
}

func (q *jobQueue) Enqueue(_ context.Context, _ string, job queue.Job) error {
	q.jobs = append(q.jobs, job)
	return nil
}
func (q *jobQueue) NewConsumer(context.Context, string, string, string) (queue.Consumer, error) {
	return nil, nil
}
func (q *jobQueue) Len(context.Context, string) (int64, error)             { return 0, nil }
func (q *jobQueue) Pending(context.Context, string, string) (int64, error) { return 0, nil }
func (q *jobQueue) DeadLetterLen(context.Context, string) (int64, error)   { return 0, nil }
func (q *jobQueue) Cancel(context.Context, string, string) error           { return nil }

func TestDispatcherPublishesDeploymentDirectlyToJobQueue(t *testing.T) {
	jobs := &jobQueue{}
	dispatcher := &Dispatcher{Jobs: jobs}
	event := events.Event{ID: uuid.NewString(), Type: "deployment.queued", AggregateID: uuid.NewString(), Payload: []byte(`{"ID":"job-1","DeploymentID":"dep-1","AppID":"app-1","OrgID":"org-1"}`)}
	if err := dispatcher.publish(context.Background(), event, "deployments"); err != nil {
		t.Fatal(err)
	}
	if len(jobs.jobs) != 1 || jobs.jobs[0].ID != "job-1" || jobs.jobs[0].DeploymentID != "dep-1" {
		t.Fatalf("unexpected direct job: %+v", jobs.jobs)
	}
}

func TestDispatcherPublishesToNATS(t *testing.T) {
	url := os.Getenv("AETHER_NATS_TEST_URL")
	if url == "" {
		t.Skip("AETHER_NATS_TEST_URL is not set")
	}
	pool := testPool(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	runtime, err := adapter.New(ctx, druntime.Config{Backend: "nats", NATSURL: url, NATSName: "aether-outbox-test"})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close(context.Background())
	received := make(chan events.Event, 1)
	subscription, err := runtime.Events.Subscribe(ctx, "outbox-integration", func(_ context.Context, event events.Event) {
		received <- event
	})
	if err != nil {
		t.Fatal(err)
	}
	defer subscription.Unsubscribe()

	store := NewStore(pool)
	event := events.Event{ID: uuid.NewString(), Type: "outbox.integration", AggregateType: "test", AggregateID: uuid.NewString(), Payload: []byte(`{"ok":true}`), TS: time.Now().UTC()}
	if err := store.Enqueue(ctx, event, "outbox-integration"); err != nil {
		t.Fatal(err)
	}
	dispatchCtx, stop := context.WithCancel(ctx)
	defer stop()
	go (&Dispatcher{Store: store, Bus: runtime.Events}).Run(dispatchCtx)
	select {
	case published := <-received:
		if published.ID != event.ID || published.Type != event.Type {
			t.Fatalf("unexpected published event: %+v", published)
		}
	case <-ctx.Done():
		t.Fatal("outbox event was not published")
	}
	var publishedAt *time.Time
	if err := pool.QueryRow(ctx, "SELECT published_at FROM outbox_events WHERE id=$1", uuid.MustParse(event.ID)).Scan(&publishedAt); err != nil {
		t.Fatal(err)
	}
	if publishedAt == nil {
		t.Fatal("outbox event was not marked published")
	}
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	port := 5432
	if value := os.Getenv("AETHER_TEST_DATABASE_PORT"); value != "" {
		port, _ = strconv.Atoi(value)
	}
	user := os.Getenv("AETHER_TEST_DATABASE_USER")
	password := os.Getenv("AETHER_TEST_DATABASE_PASSWORD")
	if user == "" {
		user, password = "aether", ""
	}
	cfg := database.Config{Host: "127.0.0.1", Port: port, Name: "aether_outbox_test", User: user, Password: password, SSLMode: "disable", PoolMax: 4, ConnectTimeout: 5}
	ctx := context.Background()
	if err := database.EnsureDatabase(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	pool, err := database.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, pool, "../../../db/migrations"); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE outbox_events"); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	return pool
}
