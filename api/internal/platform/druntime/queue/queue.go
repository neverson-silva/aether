package queue

import (
	"context"
	"errors"
	"time"
)

type PermanentError struct {
	Err error
}

func (e PermanentError) Error() string { return e.Err.Error() }
func (e PermanentError) Unwrap() error { return e.Err }

func IsPermanent(err error) bool {
	var permanent PermanentError
	return errors.As(err, &permanent)
}

func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return PermanentError{Err: err}
}

type Job struct {
	ID           string
	DeliveryID   string
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

type ProgressingConsumer interface {
	Consumer
	InProgress(ctx context.Context, job *Job) error
}

func StartProgress(ctx context.Context, consumer Consumer, job *Job) func() {
	progressing, ok := consumer.(ProgressingConsumer)
	if !ok {
		return func() {}
	}
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = progressing.InProgress(ctx, job)
			case <-stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	return func() { close(stop) }
}

type Queue interface {
	Enqueue(ctx context.Context, stream string, job Job) error
	NewConsumer(ctx context.Context, stream, group, consumerID string) (Consumer, error)
	Len(ctx context.Context, stream string) (int64, error)
	Pending(ctx context.Context, stream, group string) (int64, error)
	DeadLetterLen(ctx context.Context, stream string) (int64, error)
	Cancel(ctx context.Context, stream, jobID string) error
}

type Metrics struct {
	Stream       string `json:"stream"`
	Pending      int64  `json:"pending"`
	AckPending   int64  `json:"ack_pending"`
	Redeliveries int64  `json:"redeliveries"`
	DeadLetter   int64  `json:"dead_letter"`
}

type MetricsProvider interface {
	QueueMetrics(ctx context.Context, stream, group string) (Metrics, error)
}
