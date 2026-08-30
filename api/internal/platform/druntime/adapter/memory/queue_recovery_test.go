package memory

import (
	"context"
	"testing"
	"time"

	"aether/internal/platform/druntime/queue"
)

func TestConsumerCloseRedeliversUnacknowledgedJobs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	q := NewQueue()
	first, err := q.NewConsumer(ctx, "jobs", "workers", "worker-a")
	if err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(ctx, "jobs", queue.Job{ID: "job-1", Type: "deploy.execute"}); err != nil {
		t.Fatal(err)
	}
	job, err := first.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if job.ID != "job-1" {
		t.Fatalf("unexpected job: %+v", job)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second, err := q.NewConsumer(ctx, "jobs", "workers", "worker-b")
	if err != nil {
		t.Fatal(err)
	}
	redelivered, err := second.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if redelivered.ID != job.ID || redelivered.Type != job.Type {
		t.Fatalf("unexpected redelivery: %+v", redelivered)
	}
	if err := second.Ack(ctx, redelivered); err != nil {
		t.Fatal(err)
	}
}

func TestNackMovesJobToDeadLetterQueueAfterRetryLimit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	q := NewQueue()
	consumer, err := q.NewConsumer(ctx, "jobs", "workers", "worker")
	if err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(ctx, "jobs", queue.Job{ID: "job-dlq", Type: "backup.create"}); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < maxRetries; attempt++ {
		job, nextErr := consumer.Next(ctx)
		if nextErr != nil {
			t.Fatal(nextErr)
		}
		if err := consumer.Nack(ctx, job); err != nil {
			t.Fatal(err)
		}
	}
	length, err := q.DeadLetterLen(ctx, "jobs")
	if err != nil {
		t.Fatal(err)
	}
	if length != 1 {
		t.Fatalf("expected one dead-lettered job, got %d", length)
	}
}

func TestConsumerDeliversJobsInEnqueueOrder(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	q := NewQueue()
	consumer, err := q.NewConsumer(ctx, "deployments", "workers", "worker")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"first", "second", "third"} {
		if err := q.Enqueue(ctx, "deployments", queue.Job{ID: id, Type: "deployment.execute"}); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range []string{"first", "second", "third"} {
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
}

func TestQueueDeduplicatesJobID(t *testing.T) {
	ctx := context.Background()
	q := NewQueue()
	consumer, err := q.NewConsumer(ctx, "deployments", "workers", "worker")
	if err != nil {
		t.Fatal(err)
	}
	job := queue.Job{ID: "duplicate", Type: "deployment.execute"}
	if err := q.Enqueue(ctx, "deployments", job); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(ctx, "deployments", job); err != nil {
		t.Fatal(err)
	}
	first, err := consumer.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := consumer.Ack(ctx, first); err != nil {
		t.Fatal(err)
	}
	shortCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	if _, err := consumer.Next(shortCtx); err == nil {
		t.Fatal("duplicate job was delivered")
	}
}

func TestQueueCancellationNotifiesWatchers(t *testing.T) {
	q := NewQueue()
	ctx := context.Background()
	cancelled := make(chan string, 1)
	stop, err := q.WatchCancellations(ctx, "deployments", func(id string) {
		cancelled <- id
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	if err := q.Enqueue(ctx, "deployments", queue.Job{ID: "deployment-1", Type: "deployment.execute"}); err != nil {
		t.Fatal(err)
	}
	if err := q.Cancel(ctx, "deployments", "deployment-1"); err != nil {
		t.Fatal(err)
	}
	select {
	case id := <-cancelled:
		if id != "deployment-1" {
			t.Fatalf("cancelled id = %q", id)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation watcher was not notified")
	}
}
