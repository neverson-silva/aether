package infra

import (
	"context"
	"encoding/json"
	"sync"

	natsgo "github.com/nats-io/nats.go"

	"aether/internal/modules/monitoring/domain"
	"aether/internal/platform/messaging"
	"aether/internal/platform/observability"
)

type Publisher struct {
	conn    *natsgo.Conn
	Metrics *observability.Metrics
}

func NewPublisher(url, name string) (*Publisher, error) {
	return NewPublisherWithAuth(url, name, "", "")
}

func NewPublisherWithAuth(url, name, user, password string) (*Publisher, error) {
	options := []natsgo.Option{natsgo.Name(name + "-monitoring")}
	if user != "" || password != "" {
		options = append(options, natsgo.UserInfo(user, password))
	}
	conn, err := natsgo.Connect(url, options...)
	if err != nil {
		return nil, err
	}
	return &Publisher{conn: conn}, nil
}

func (p *Publisher) Publish(snapshot *domain.Snapshot) {
	raw, err := json.Marshal(snapshot)
	if err == nil {
		err = p.conn.Publish(messaging.MonitoringSnapshot, raw)
	}
	if p.Metrics != nil {
		p.Metrics.ObservePublish(err != nil)
	}
}

func (p *Publisher) Close() error {
	return p.conn.Drain()
}

type Reader struct {
	mu      sync.RWMutex
	latest  *domain.Snapshot
	conn    *natsgo.Conn
	sub     *natsgo.Subscription
	owned   bool
	history interface {
		History(string) []domain.HistoryPoint
		ResourceHistory(string, string) []domain.ResourcePoint
		CollectorStats() domain.CollectorStats
	}
}

func NewReader(ctx context.Context, url, name string, history interface {
	History(string) []domain.HistoryPoint
	ResourceHistory(string, string) []domain.ResourcePoint
	CollectorStats() domain.CollectorStats
}) (*Reader, error) {
	return NewReaderWithAuth(ctx, url, name, "", "", history)
}

func NewReaderWithAuth(ctx context.Context, url, name, user, password string, history interface {
	History(string) []domain.HistoryPoint
	ResourceHistory(string, string) []domain.ResourcePoint
	CollectorStats() domain.CollectorStats
}) (*Reader, error) {
	options := []natsgo.Option{natsgo.Name(name + "-api-monitoring")}
	if user != "" || password != "" {
		options = append(options, natsgo.UserInfo(user, password))
	}
	conn, err := natsgo.Connect(url, options...)
	if err != nil {
		return nil, err
	}
	reader, err := newReader(ctx, conn, history, true)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return reader, nil
}

func NewReaderWithConn(ctx context.Context, conn *natsgo.Conn, history interface {
	History(string) []domain.HistoryPoint
	ResourceHistory(string, string) []domain.ResourcePoint
	CollectorStats() domain.CollectorStats
}) (*Reader, error) {
	return newReader(ctx, conn, history, false)
}

func newReader(ctx context.Context, conn *natsgo.Conn, history interface {
	History(string) []domain.HistoryPoint
	ResourceHistory(string, string) []domain.ResourcePoint
	CollectorStats() domain.CollectorStats
}, owned bool) (*Reader, error) {
	reader := &Reader{history: history, conn: conn, owned: owned}
	sub, err := conn.Subscribe(messaging.MonitoringSnapshot, func(msg *natsgo.Msg) {
		var snapshot domain.Snapshot
		if json.Unmarshal(msg.Data, &snapshot) != nil {
			return
		}
		reader.mu.Lock()
		reader.latest = &snapshot
		reader.mu.Unlock()
	})
	if err != nil {
		if owned {
			conn.Close()
		}
		return nil, err
	}
	reader.sub = sub
	go func() {
		<-ctx.Done()
		_ = sub.Unsubscribe()
		if owned {
			_ = conn.Drain()
		}
	}()
	return reader, nil
}

func (r *Reader) Latest() *domain.Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.latest == nil {
		return &domain.Snapshot{System: domain.Aggregate{Available: true}}
	}
	return r.latest
}

func (r *Reader) History(window string) []domain.HistoryPoint {
	return r.history.History(window)
}

func (r *Reader) ResourceHistory(id, window string) []domain.ResourcePoint {
	return r.history.ResourceHistory(id, window)
}

func (r *Reader) CollectorStats() domain.CollectorStats {
	return r.history.CollectorStats()
}
