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
