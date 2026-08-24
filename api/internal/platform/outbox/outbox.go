package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"aether/internal/platform/druntime/events"
	"aether/internal/platform/druntime/queue"
)

type Store struct {
	pool *pgxpool.Pool
}

type Item struct {
	ID            uuid.UUID
	Topic         string
	EventType     string
	AggregateType string
	AggregateID   string
	Payload       []byte
	Attempts      int
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Enqueue(ctx context.Context, event events.Event, topic string) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(event.ID)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO outbox_events (id, topic, event_type, aggregate_type, aggregate_id, payload) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (id) DO NOTHING`, id, topic, event.Type, event.AggregateType, event.AggregateID, payload)
	return err
}

func (s *Store) Claim(ctx context.Context, limit int) ([]Item, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `UPDATE outbox_events SET attempts = attempts + 1 WHERE id IN (SELECT id FROM outbox_events WHERE published_at IS NULL AND available_at <= now() ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT $1) RETURNING id, topic, event_type, aggregate_type, aggregate_id, payload, attempts`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Item, 0, limit)
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Topic, &item.EventType, &item.AggregateType, &item.AggregateID, &item.Payload, &item.Attempts); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, tx.Commit(ctx)
}

func (s *Store) MarkPublished(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE outbox_events SET published_at = now() WHERE id = $1`, id)
	return err
}

func (s *Store) Retry(ctx context.Context, id uuid.UUID, delay time.Duration) error {
	_, err := s.pool.Exec(ctx, `UPDATE outbox_events SET available_at = now() + $2::interval WHERE id = $1`, id, delay.String())
	return err
}

type Dispatcher struct {
	Store *Store
	Bus   events.EventBus
	Jobs  queue.Queue
}

func (d *Dispatcher) Run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		items, err := d.Store.Claim(ctx, 32)
		if err == nil {
			for _, item := range items {
				var event events.Event
				if json.Unmarshal(item.Payload, &event) == nil && d.publish(ctx, event, item.Topic) == nil {
					_ = d.Store.MarkPublished(ctx, item.ID)
				} else {
					_ = d.Store.Retry(ctx, item.ID, retryDelay(item.Attempts))
				}
			}
		}
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (d *Dispatcher) publish(ctx context.Context, event events.Event, topic string) error {
	if d.Jobs != nil && strings.HasSuffix(event.Type, ".queued") {
		var job queue.Job
		if err := json.Unmarshal(event.Payload, &job); err != nil {
			return err
		}
		if job.ID == "" {
			job.ID = event.AggregateID
		}
		if job.DeploymentID == "" {
			job.DeploymentID = event.AggregateID
		}
		return d.Jobs.Enqueue(ctx, topic, job)
	}
	if d.Bus == nil {
		return fmt.Errorf("outbox event bus is unavailable")
	}
	return d.Bus.Publish(ctx, topic, event)
}

func retryDelay(attempt int) time.Duration {
	delays := []time.Duration{5 * time.Second, 15 * time.Second, 30 * time.Second, time.Minute, 5 * time.Minute}
	if attempt < 1 {
		return delays[0]
	}
	if attempt > len(delays) {
		return delays[len(delays)-1]
	}
	return delays[attempt-1]
}
