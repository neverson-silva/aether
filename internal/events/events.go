package events

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"aether/internal/db"
	"aether/internal/domain"
)

type Event struct {
	ID            string          `json:"id"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	Sequence      int64           `json:"sequence"`
	Type          string          `json:"type"`
	Payload       json.RawMessage `json:"payload"`
	TS            time.Time       `json:"ts"`
}

type Handler func(ctx context.Context, e Event)

type Bus struct {
	db          *db.SQL
	mu          sync.Mutex
	handlers    []Handler
	distributed func(ev Event)
}

func NewBus(db *db.SQL) *Bus {
	return &Bus{db: db}
}

func (b *Bus) SetDistributed(fn func(ev Event)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.distributed = fn
}

func (b *Bus) Subscribe(h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, h)
}

func (b *Bus) Publish(ctx context.Context, aggType, aggID, typ string, payload any, beforeCommit func(tx *db.Tx) error) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var seq int64
	if err := tx.QueryRow(`SELECT COALESCE(MAX(sequence),0)+1 FROM events WHERE aggregate_type=? AND aggregate_id=?`, aggType, aggID).Scan(&seq); err != nil {
		return err
	}
	ev := Event{
		ID:            domain.NewID(),
		AggregateType: aggType,
		AggregateID:   aggID,
		Sequence:      seq,
		Type:          typ,
		Payload:       payloadJSON,
		TS:            time.Now().UTC(),
	}
	if _, err := tx.Exec(`INSERT INTO events(aggregate_type,aggregate_id,sequence,type,payload,ts,published) VALUES(?,?,?,?,?,?,0)`,
		aggType, aggID, seq, typ, string(payloadJSON), ev.TS.UnixMilli()); err != nil {
		return err
	}
	if beforeCommit != nil {
		if err := beforeCommit(tx); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	b.dispatch(ctx, ev)
	b.publishDistributed(ev)
	_, err = b.db.Exec(`UPDATE events SET published=1 WHERE aggregate_type=? AND aggregate_id=? AND sequence=?`, aggType, aggID, seq)
	return err
}

func (b *Bus) publishDistributed(ev Event) {
	b.mu.Lock()
	fn := b.distributed
	b.mu.Unlock()
	if fn != nil {
		fn(ev)
	}
}

func (b *Bus) dispatch(ctx context.Context, ev Event) {
	b.mu.Lock()
	handlers := append([]Handler(nil), b.handlers...)
	b.mu.Unlock()
	for _, h := range handlers {
		h(ctx, ev)
	}
}

func (b *Bus) ReplayUnpublished(ctx context.Context) (int, error) {
	rows, err := b.db.Query(`SELECT aggregate_type,aggregate_id,sequence,type,payload,ts FROM events WHERE published=0 ORDER BY ts`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var pending []Event
	for rows.Next() {
		var e Event
		var payload string
		var ts int64
		if err := rows.Scan(&e.AggregateType, &e.AggregateID, &e.Sequence, &e.Type, &payload, &ts); err != nil {
			return 0, err
		}
		e.Payload = json.RawMessage(payload)
		e.TS = time.UnixMilli(ts)
		pending = append(pending, e)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, e := range pending {
		b.dispatch(ctx, e)
		b.publishDistributed(e)
		if _, err := b.db.Exec(`UPDATE events SET published=1 WHERE aggregate_type=? AND aggregate_id=? AND sequence=?`,
			e.AggregateType, e.AggregateID, e.Sequence); err != nil {
			return len(pending), err
		}
	}
	return len(pending), nil
}

func (b *Bus) Timeline(ctx context.Context, aggType, aggID string) ([]Event, error) {
	rows, err := b.db.Query(`SELECT aggregate_type,aggregate_id,sequence,type,payload,ts FROM events WHERE aggregate_type=? AND aggregate_id=? ORDER BY sequence`, aggType, aggID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var payload string
		var ts int64
		if err := rows.Scan(&e.AggregateType, &e.AggregateID, &e.Sequence, &e.Type, &payload, &ts); err != nil {
			return nil, err
		}
		e.Payload = json.RawMessage(payload)
		e.TS = time.UnixMilli(ts)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (b *Bus) Recent(ctx context.Context, limit int) ([]Event, error) {
	rows, err := b.db.Query(`SELECT aggregate_type,aggregate_id,sequence,type,payload,ts FROM events ORDER BY ts DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var payload string
		var ts int64
		if err := rows.Scan(&e.AggregateType, &e.AggregateID, &e.Sequence, &e.Type, &payload, &ts); err != nil {
			return nil, err
		}
		e.Payload = json.RawMessage(payload)
		e.TS = time.UnixMilli(ts)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (b *Bus) Count() (int64, error) {
	var n int64
	err := b.db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&n)
	return n, err
}

func (b *Bus) GC(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan).UnixMilli()
	res, err := b.db.Exec(`DELETE FROM events WHERE ts<? AND published=1`, cutoff)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

var ErrNoHandler = errors.New("no handler")
