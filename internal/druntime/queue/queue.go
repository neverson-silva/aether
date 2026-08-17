package queue

import (
	"context"
	"time"
)

type Job struct {
	ID           string
	Type         string
	Priority     int
	Payload      []byte
	OrgID        string
	ProjectID    string
	AppID        string
	DeploymentID string
	ServerID     string
	CreatedAt    time.Time
	Attempt      int
	Timeout      time.Duration
}

type Consumer interface {
	Next(ctx context.Context) (*Job, error)
	Ack(ctx context.Context, job *Job) error
	Nack(ctx context.Context, job *Job) error
	Close() error
}

type Queue interface {
	Enqueue(ctx context.Context, stream string, job Job) error
	NewConsumer(ctx context.Context, stream, group, consumerID string) (Consumer, error)
	Len(ctx context.Context, stream string) (int64, error)
	Pending(ctx context.Context, stream, group string) (int64, error)
	DeadLetterLen(ctx context.Context, stream string) (int64, error)
	Cancel(ctx context.Context, stream, jobID string) error
}
