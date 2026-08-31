package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"aether/internal/platform/druntime/queue"
	"aether/internal/platform/messaging"
)

type Queue struct {
	rt *Runtime
}

func jobSubject(stream string) string { return messaging.Jobs(stream) }

func (q *Queue) Enqueue(ctx context.Context, stream string, job queue.Job) error {
	if job.ID == "" {
		job.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(messaging.Envelope{ID: job.ID, Type: job.Type, SchemaVersion: 1, OrgID: job.OrgID, ResourceID: job.DeploymentID, CreatedAt: time.Now().UTC(), Payload: payload})
	if err != nil {
		return err
	}
	_, err = q.rt.js.Publish(ctx, jobSubject(stream), raw, jetstream.WithMsgID(job.ID))
	return err
}

func (q *Queue) NewConsumer(ctx context.Context, stream, group, consumerID string) (queue.Consumer, error) {
	jsConsumer, err := q.rt.js.CreateOrUpdateConsumer(ctx, jobsStream, jetstream.ConsumerConfig{
		Name:          group,
		Durable:       group,
		FilterSubject: jobSubject(stream),
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    maxJobDeliveries,
		BackOff:       []time.Duration{5 * time.Second, 15 * time.Second, 30 * time.Second, time.Minute, 5 * time.Minute},
		MaxAckPending: 1,
		MaxWaiting:    1,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
	})
	if err != nil {
		return nil, err
	}
	return &consumer{consumer: jsConsumer, rt: q.rt, stream: stream, pending: map[string]jetstream.Msg{}, mu: sync.Mutex{}}, nil
}

type consumer struct {
	consumer jetstream.Consumer
	rt       *Runtime
	stream   string
	pending  map[string]jetstream.Msg
	mu       sync.Mutex
}

func (c *consumer) Next(ctx context.Context) (*queue.Job, error) {
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		msg, err := c.consumer.Next(jetstream.FetchContext(fetchCtx))
		cancel()
		if err != nil {
			return nil, err
		}
		metadata, metadataErr := msg.Metadata()
		if metadataErr != nil {
			return nil, metadataErr
		}
		if metadata.NumDelivered >= maxJobDeliveries {
			if err := c.publishDLQ(msg, metadata, "maximum delivery attempts exceeded"); err != nil {
				return nil, err
			}
			if err := msg.TermWithReason("maximum delivery attempts exceeded"); err != nil {
				return nil, err
			}
			continue
		}
		job, err := decodeJob(msg.Data())
		if err != nil {
			_ = msg.TermWithReason("invalid job payload")
			continue
		}
		job.DeliveryID = fmt.Sprintf("%d", metadata.Sequence.Consumer)
		job.Attempt = int(metadata.NumDelivered) - 1
		c.mu.Lock()
		c.pending[job.DeliveryID] = msg
		c.mu.Unlock()
		return &job, nil
	}
}

func decodeJob(data []byte) (queue.Job, error) {
	var envelope messaging.Envelope
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.SchemaVersion > 0 && len(envelope.Payload) > 0 {
		var job queue.Job
		if err := json.Unmarshal(envelope.Payload, &job); err != nil {
			return queue.Job{}, err
		}
		if job.ID == "" {
			job.ID = envelope.ID
		}
		if job.Type == "" {
			job.Type = envelope.Type
		}
		if job.OrgID == "" {
			job.OrgID = envelope.OrgID
		}
		if job.DeploymentID == "" {
			job.DeploymentID = envelope.ResourceID
		}
		if job.CreatedAt.IsZero() {
			job.CreatedAt = envelope.CreatedAt
		}
		return job, nil
	}
	var job queue.Job
	if err := json.Unmarshal(data, &job); err != nil {
		return queue.Job{}, err
	}
	return job, nil
}

type dlqMessage struct {
	Job           queue.Job `json:"job"`
	Original      string    `json:"original_subject"`
	Attempts      uint64    `json:"attempts"`
	FailedAt      time.Time `json:"failed_at"`
	Error         string    `json:"error"`
	Payload       []byte    `json:"payload"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	CausationID   string    `json:"causation_id,omitempty"`
	ResourceID    string    `json:"resource_id,omitempty"`
}

func (c *consumer) publishDLQ(msg jetstream.Msg, metadata *jetstream.MsgMetadata, reason string) error {
	job, _ := decodeJob(msg.Data())
	var envelope messaging.Envelope
	_ = json.Unmarshal(msg.Data(), &envelope)
	payload, err := json.Marshal(dlqMessage{Job: job, Original: msg.Subject(), Attempts: metadata.NumDelivered, FailedAt: time.Now().UTC(), Error: reason, Payload: msg.Data(), CorrelationID: envelope.CorrelationID, CausationID: envelope.CausationID, ResourceID: envelope.ResourceID})
	if err != nil {
		return err
	}
	_, err = c.rt.js.Publish(context.Background(), messaging.DLQ(c.stream), payload, jetstream.WithMsgID(fmt.Sprintf("dlq:%d", metadata.Sequence.Stream)))
	return err
}

func (c *consumer) Ack(_ context.Context, job *queue.Job) error {
	msg := c.take(job.DeliveryID)
	if msg == nil {
		return nil
	}
	return msg.Ack()
}

func (c *consumer) Nack(_ context.Context, job *queue.Job) error {
	msg := c.take(job.DeliveryID)
	if msg == nil {
		return nil
	}
	return msg.NakWithDelay(retryDelay(job.Attempt))
}

func (c *consumer) InProgress(_ context.Context, job *queue.Job) error {
	c.mu.Lock()
	msg := c.pending[job.DeliveryID]
	c.mu.Unlock()
	if msg == nil {
		return nil
	}
	return msg.InProgress()
}

func (c *consumer) take(id string) jetstream.Msg {
	c.mu.Lock()
	msg := c.pending[id]
	delete(c.pending, id)
	c.mu.Unlock()
	return msg
}

func (c *consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, msg := range c.pending {
		_ = msg.NakWithDelay(time.Second)
		delete(c.pending, id)
	}
	return nil
}

func retryDelay(attempt int) time.Duration {
	values := []time.Duration{5 * time.Second, 15 * time.Second, 30 * time.Second, time.Minute, 5 * time.Minute}
	if attempt < 0 {
		attempt = 0
	}
	if attempt >= len(values) {
		return values[len(values)-1]
	}
	return values[attempt]
}

const maxJobDeliveries = 5

func (q *Queue) Len(ctx context.Context, stream string) (int64, error) {
	info, err := q.rt.js.Stream(ctx, jobsStream)
	if err != nil {
		return 0, err
	}
	state, err := info.Info(ctx)
	if err != nil {
		return 0, err
	}
	return int64(state.State.Msgs), nil
}

func (q *Queue) Pending(ctx context.Context, stream, group string) (int64, error) {
	consumer, err := q.rt.js.Consumer(ctx, jobsStream, group)
	if err != nil {
		return 0, err
	}
	info, err := consumer.Info(ctx)
	if err != nil {
		return 0, err
	}
	return int64(info.NumAckPending), nil
}

func (q *Queue) DeadLetterLen(ctx context.Context, stream string) (int64, error) {
	info, err := q.rt.js.Stream(ctx, dlqStream)
	if err != nil {
		return 0, err
	}
	state, err := info.Info(ctx)
	if err != nil {
		return 0, err
	}
	return int64(state.State.Msgs), nil
}

func (q *Queue) QueueMetrics(ctx context.Context, stream, group string) (queue.Metrics, error) {
	consumer, err := q.rt.js.Consumer(ctx, jobsStream, group)
	if err != nil {
		return queue.Metrics{}, err
	}
	consumerInfo, err := consumer.Info(ctx)
	if err != nil {
		return queue.Metrics{}, err
	}
	dlq, err := q.rt.js.Stream(ctx, dlqStream)
	if err != nil {
		return queue.Metrics{}, err
	}
	dlqInfo, err := dlq.Info(ctx)
	if err != nil {
		return queue.Metrics{}, err
	}
	return queue.Metrics{
		Stream: stream, Pending: int64(consumerInfo.NumPending), AckPending: int64(consumerInfo.NumAckPending),
		Redeliveries: int64(consumerInfo.NumRedelivered), DeadLetter: int64(dlqInfo.State.Msgs),
	}, nil
}

func (q *Queue) Cancel(ctx context.Context, stream, jobID string) error {
	_, err := q.rt.js.Publish(ctx, messaging.Jobs(stream)+".cancel", []byte(jobID), jetstream.WithMsgID("cancel-"+jobID))
	return err
}

func (q *Queue) WatchCancellations(ctx context.Context, stream string, handler func(string)) (func(), error) {
	subscription, err := q.rt.conn.Subscribe(messaging.Jobs(stream)+".cancel", func(message *natsgo.Msg) {
		handler(string(message.Data))
	})
	if err != nil {
		return nil, err
	}
	if err := q.rt.conn.Flush(); err != nil {
		_ = subscription.Unsubscribe()
		return nil, err
	}
	stop := func() {
		_ = subscription.Unsubscribe()
	}
	go func() {
		<-ctx.Done()
		stop()
	}()
	return stop, nil
}
