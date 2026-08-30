package nats

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"aether/internal/platform/druntime"
	"aether/internal/platform/druntime/queue"
)

func TestNATSRuntimeIntegration(t *testing.T) {
	url := os.Getenv("AETHER_NATS_TEST_URL")
	if url == "" {
		t.Skip("AETHER_NATS_TEST_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := natsgo.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		t.Fatal(err)
	}
	stream, err := js.Stream(ctx, jobsStream)
	if err == nil {
		names := stream.ConsumerNames(ctx)
		for name := range names.Name() {
			_ = js.DeleteConsumer(ctx, jobsStream, name)
		}
	}
	conn.Close()
	rt, err := New(ctx, druntime.Config{Backend: "nats", NATSURL: url, NATSName: "aether-integration-test"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	group := fmt.Sprintf("integration-%d", time.Now().UnixNano())
	consumer, err := rt.Queue.NewConsumer(ctx, "backups", group, "integration")
	if err != nil {
		t.Fatal(err)
	}
	defer consumer.Close()

	job := queue.Job{ID: "integration-job-" + group, Type: "backup.create", Payload: []byte(`{"value":"ok"}`), CreatedAt: time.Now().UTC()}
	if err := rt.Queue.Enqueue(ctx, "backups", job); err != nil {
		t.Fatal(err)
	}
	received, err := consumer.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if received.ID != job.ID || received.Type != job.Type || string(received.Payload) != string(job.Payload) {
		t.Fatalf("received job does not match: %#v", received)
	}
	if err := consumer.Ack(ctx, received); err != nil {
		t.Fatal(err)
	}

	lock, acquired, err := rt.Locks.Acquire(ctx, "integration/lock", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !acquired {
		t.Fatal("expected first lock acquisition")
	}
	if locked, err := rt.Locks.Locked(ctx, lock.Name); err != nil || !locked {
		t.Fatalf("expected lock to be held: locked=%v err=%v", locked, err)
	}
	if _, acquired, err := rt.Locks.Acquire(ctx, lock.Name, time.Minute); err != nil || acquired {
		t.Fatalf("expected second lock acquisition to fail: acquired=%v err=%v", acquired, err)
	}
	if err := rt.Locks.Renew(ctx, lock, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := rt.Locks.Release(ctx, lock); err != nil {
		t.Fatal(err)
	}

	stateKey := "integration/state/" + group
	if err := rt.State.Set(ctx, stateKey, []byte("value"), time.Minute); err != nil {
		t.Fatal(err)
	}
	value, found, err := rt.State.Get(ctx, stateKey)
	if err != nil || !found || string(value) != "value" {
		t.Fatalf("unexpected state: found=%v value=%q err=%v", found, value, err)
	}
	if err := rt.State.Del(ctx, stateKey); err != nil {
		t.Fatal(err)
	}

	scheduledJob := queue.Job{ID: "scheduled-" + group, Type: "backup.schedule", Payload: []byte(`{"backup_id":"integration"}`), CreatedAt: time.Now().UTC()}
	when := time.Now().UTC().Add(2 * time.Second)
	if err := rt.Scheduler.ScheduleJobAt(ctx, "aether.jobs.backups", "integration-"+group, scheduledJob.Type, when, scheduledJob.Payload); err != nil {
		t.Fatal(err)
	}
	received, err = consumer.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if received.Type != scheduledJob.Type || string(received.Payload) != string(scheduledJob.Payload) {
		t.Fatalf("received scheduled job does not match: %#v", received)
	}
	if err := consumer.Ack(ctx, received); err != nil {
		t.Fatal(err)
	}
}

func TestNATSDeploymentQueueIsFIFOAndIdempotent(t *testing.T) {
	url := os.Getenv("AETHER_NATS_TEST_URL")
	if url == "" {
		t.Skip("AETHER_NATS_TEST_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rt, err := New(ctx, druntime.Config{Backend: "nats", NATSURL: url, NATSName: "aether-deployment-fifo-test"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	group := fmt.Sprintf("deployment-fifo-%d", time.Now().UnixNano())
	consumer, err := rt.Queue.NewConsumer(ctx, "deployments", group, "worker")
	if err != nil {
		t.Fatal(err)
	}
	defer consumer.Close()
	jobIDs := []string{"deployment-1-" + group, "deployment-2-" + group, "deployment-3-" + group}
	for _, id := range jobIDs {
		if err := rt.Queue.Enqueue(ctx, "deployments", queue.Job{ID: id, Type: "deployment.execute"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := rt.Queue.Enqueue(ctx, "deployments", queue.Job{ID: jobIDs[0], Type: "deployment.execute"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range jobIDs {
		job, err := consumer.Next(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if job.ID != want {
			t.Fatalf("job order = %q, want %q", job.ID, want)
		}
		if err := consumer.Ack(ctx, job); err != nil {
			t.Fatal(err)
		}
	}
	shortCtx, shortCancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer shortCancel()
	if _, err := consumer.Next(shortCtx); err == nil {
		t.Fatal("duplicate deployment job was delivered")
	}
}

func TestNATSAuthenticatedRuntimeIntegration(t *testing.T) {
	url := os.Getenv("AETHER_NATS_AUTH_TEST_URL")
	user := os.Getenv("AETHER_NATS_AUTH_TEST_USER")
	password := os.Getenv("AETHER_NATS_AUTH_TEST_PASSWORD")
	if url == "" || user == "" || password == "" {
		t.Skip("AETHER_NATS_AUTH_TEST_URL, AETHER_NATS_AUTH_TEST_USER and AETHER_NATS_AUTH_TEST_PASSWORD are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rt, err := New(ctx, druntime.Config{Backend: "nats", NATSURL: url, NATSName: "aether-authenticated-test", NATSUser: user, NATSPassword: password})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	bad, err := natsgo.Connect(url, natsgo.UserInfo(user, password+"-invalid"), natsgo.Timeout(time.Second))
	if err == nil {
		bad.Close()
		t.Fatal("invalid NATS credentials were accepted")
	}
}

func TestNATSRuntimeWithSharedConnectionPreservesOwnership(t *testing.T) {
	url := os.Getenv("AETHER_NATS_TEST_URL")
	if url == "" {
		t.Skip("AETHER_NATS_TEST_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := natsgo.Connect(url, natsgo.Name("aether-shared-connection-test"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	rt, err := NewWithConn(ctx, druntime.Config{Backend: "nats", NATSURL: url, NATSName: "aether-shared-connection-test"}, conn)
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if conn.IsClosed() {
		t.Fatal("shared connection was closed by runtime")
	}

	subject := "aether.integration.shared." + fmt.Sprint(time.Now().UnixNano())
	sub, err := conn.SubscribeSync(subject)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()
	if err := conn.Publish(subject, []byte("alive")); err != nil {
		t.Fatal(err)
	}
	if err := conn.Flush(); err != nil {
		t.Fatal(err)
	}
	msg, err := sub.NextMsg(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if string(msg.Data) != "alive" {
		t.Fatalf("unexpected message: %q", msg.Data)
	}
}

func TestNATSRecurringScheduleReconciliation(t *testing.T) {
	url := os.Getenv("AETHER_NATS_TEST_URL")
	if url == "" {
		t.Skip("AETHER_NATS_TEST_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rt, err := New(ctx, druntime.Config{Backend: "nats", NATSURL: url, NATSName: "aether-recurring-test"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	recurring, ok := rt.Scheduler.(interface {
		ScheduleJobCron(context.Context, string, string, string, string, string, []byte) error
		ScheduleJobEvery(context.Context, string, string, string, time.Duration, []byte) error
		ReconcileRecurring(context.Context, string, []string) error
	})
	if !ok {
		t.Fatal("runtime does not expose recurring scheduling")
	}
	group := fmt.Sprintf("recurring-%d", time.Now().UnixNano())
	streamName := "recurring-" + group
	consumer, err := rt.Queue.NewConsumer(ctx, streamName, group, "integration")
	if err != nil {
		t.Fatal(err)
	}
	defer consumer.Close()
	cronKey := "integration:cron:" + group
	if err := recurring.ScheduleJobCron(ctx, "aether.jobs."+streamName, cronKey, "snapshot.create", "0 30 * * * *", "UTC", []byte(`{"value":"cron"}`)); err != nil {
		t.Fatal(err)
	}
	key := "integration:every:" + group
	if err := recurring.ScheduleJobEvery(ctx, "aether.jobs."+streamName, key, "snapshot.create", time.Second, []byte(`{"value":"recurring"}`)); err != nil {
		t.Fatal(err)
	}
	job, err := consumer.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if job.Type != "snapshot.create" || string(job.Payload) != `{"value":"recurring"}` {
		t.Fatalf("unexpected recurring job: %+v", job)
	}
	if err := consumer.Ack(ctx, job); err != nil {
		t.Fatal(err)
	}
	if err := recurring.ReconcileRecurring(ctx, "integration", nil); err != nil {
		t.Fatal(err)
	}
}

func TestNATSRecurringScheduleUpdateAndRemovalDoNotDuplicate(t *testing.T) {
	url := os.Getenv("AETHER_NATS_TEST_URL")
	if url == "" {
		t.Skip("AETHER_NATS_TEST_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	rt, err := New(ctx, druntime.Config{Backend: "nats", NATSURL: url, NATSName: "aether-recurring-update-test"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	recurring := rt.Scheduler.(interface {
		ScheduleJobEvery(context.Context, string, string, string, time.Duration, []byte) error
		ReconcileRecurring(context.Context, string, []string) error
	})
	group := fmt.Sprintf("recurring-update-%d", time.Now().UnixNano())
	streamName := "recurring-update-" + group
	consumer, err := rt.Queue.NewConsumer(ctx, streamName, group, "integration")
	if err != nil {
		t.Fatal(err)
	}
	defer consumer.Close()
	key := "integration:update:" + group
	if err := recurring.ScheduleJobEvery(ctx, "aether.jobs."+streamName, key, "snapshot.create", time.Second, []byte(`{"version":1}`)); err != nil {
		t.Fatal(err)
	}
	job, err := consumer.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if string(job.Payload) != `{"version":1}` {
		t.Fatalf("unexpected initial payload: %q", job.Payload)
	}
	if err := consumer.Ack(ctx, job); err != nil {
		t.Fatal(err)
	}
	if err := recurring.ScheduleJobEvery(ctx, "aether.jobs."+streamName, key, "snapshot.create", 3*time.Second, []byte(`{"version":2}`)); err != nil {
		t.Fatal(err)
	}
	shortCtx, shortCancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer shortCancel()
	if _, err := consumer.Next(shortCtx); err == nil {
		t.Fatal("updated schedule delivered a duplicate before its new interval")
	}
	if err := recurring.ReconcileRecurring(ctx, "integration", nil); err != nil {
		t.Fatal(err)
	}
	removedCtx, removedCancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer removedCancel()
	if _, err := consumer.Next(removedCtx); err == nil {
		t.Fatal("removed schedule delivered a job")
	}
}

func TestNATSRecurringScheduleRecreatedAfterStateLoss(t *testing.T) {
	url := os.Getenv("AETHER_NATS_TEST_URL")
	if url == "" {
		t.Skip("AETHER_NATS_TEST_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	rt, err := New(ctx, druntime.Config{Backend: "nats", NATSURL: url, NATSName: "aether-recurring-recreate-test"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	recurring := rt.Scheduler.(interface {
		ScheduleJobEvery(context.Context, string, string, string, time.Duration, []byte) error
	})
	group := fmt.Sprintf("recurring-recreate-%d", time.Now().UnixNano())
	streamName := "recurring-recreate-" + group
	consumer, err := rt.Queue.NewConsumer(ctx, streamName, group, "integration")
	if err != nil {
		t.Fatal(err)
	}
	defer consumer.Close()
	key := "integration:recreate:" + group
	if err := recurring.ScheduleJobEvery(ctx, "aether.jobs."+streamName, key, "snapshot.create", time.Second, []byte(`{"value":"recreated"}`)); err != nil {
		t.Fatal(err)
	}
	if err := rt.State.Del(ctx, recurringStateKey(key)); err != nil {
		t.Fatal(err)
	}
	if err := recurring.ScheduleJobEvery(ctx, "aether.jobs."+streamName, key, "snapshot.create", time.Second, []byte(`{"value":"recreated"}`)); err != nil {
		t.Fatal(err)
	}
	job, err := consumer.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if string(job.Payload) != `{"value":"recreated"}` {
		t.Fatalf("unexpected recreated payload: %q", job.Payload)
	}
	if err := consumer.Ack(ctx, job); err != nil {
		t.Fatal(err)
	}
}
