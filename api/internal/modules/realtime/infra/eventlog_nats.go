package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"aether/internal/modules/realtime/domain"
	"aether/internal/platform/messaging"
)

const natsEventsStream = "AETHER_EVENTS"

type NATSEventLog struct {
	conn  *natsgo.Conn
	js    jetstream.JetStream
	owned bool
}

func NewNATSEventLog(url, name string) (*NATSEventLog, error) {
	return NewNATSEventLogWithAuth(url, name, "", "")
}

func NewNATSEventLogWithAuth(url, name, user, password string) (*NATSEventLog, error) {
	if url == "" {
		url = natsgo.DefaultURL
	}
	options := []natsgo.Option{natsgo.Name(name + "-events")}
	if user != "" || password != "" {
		options = append(options, natsgo.UserInfo(user, password))
	}
	conn, err := natsgo.Connect(url, options...)
	if err != nil {
		return nil, err
	}
	js, err := jetstream.New(conn)
	if err != nil {
		_ = conn.Drain()
		return nil, err
	}
	return newNATSEventLog(conn, js, true), nil
}

func NewNATSEventLogWithConn(conn *natsgo.Conn) (*NATSEventLog, error) {
	js, err := jetstream.New(conn)
	if err != nil {
		return nil, err
	}
	return newNATSEventLog(conn, js, false), nil
}

func newNATSEventLog(conn *natsgo.Conn, js jetstream.JetStream, owned bool) *NATSEventLog {
	return &NATSEventLog{conn: conn, js: js, owned: owned}
}

func (l *NATSEventLog) Close() error {
	if !l.owned {
		return nil
	}
	return l.conn.Drain()
}

func (l *NATSEventLog) Append(ctx context.Context, orgID uuid.UUID, event domain.Event) (int64, error) {
	payload, err := json.Marshal(event)
	if err != nil {
		return 0, err
	}
	raw, err := json.Marshal(messaging.Envelope{ID: event.ID, Type: event.Type, SchemaVersion: 1, OrgID: orgID.String(), ResourceID: event.ResourceID, CreatedAt: event.TS, Payload: payload})
	if err != nil {
		return 0, err
	}
	ack, err := l.js.Publish(ctx, eventSubject(orgID), raw, jetstream.WithMsgID(event.ID))
	if err != nil {
		return 0, err
	}
	event.Seq = int64(ack.Sequence)
	return int64(ack.Sequence), nil
}

func (l *NATSEventLog) Recent(ctx context.Context, orgID uuid.UUID, limit int) ([]domain.Event, error) {
	if limit <= 0 {
		limit = 20
	}
	return l.read(ctx, orgID, 0, limit, true)
}

func (l *NATSEventLog) Replay(ctx context.Context, orgID uuid.UUID, afterSeq int64, limit int) ([]domain.Event, error) {
	if limit <= 0 {
		limit = 50
	}
	return l.read(ctx, orgID, afterSeq, limit, false)
}

func (l *NATSEventLog) read(ctx context.Context, orgID uuid.UUID, afterSeq int64, limit int, recent bool) ([]domain.Event, error) {
	stream, err := l.js.Stream(ctx, natsEventsStream)
	if err != nil {
		return nil, err
	}
	info, err := stream.Info(ctx)
	if err != nil {
		return nil, err
	}
	first, last := info.State.FirstSeq, info.State.LastSeq
	if recent {
		out := make([]domain.Event, 0, limit)
		for seq := last; seq >= first && len(out) < limit; seq-- {
			msg, err := stream.GetMsg(ctx, seq)
			if err != nil {
				if errors.Is(err, jetstream.ErrMsgNotFound) || seq == first {
					break
				}
				continue
			}
			if msg.Subject != eventSubject(orgID) {
				continue
			}
			event, ok := decodeEvent(msg.Data)
			if !ok {
				continue
			}
			event.Seq = int64(msg.Sequence)
			out = append([]domain.Event{event}, out...)
		}
		return out, nil
	}
	out := make([]domain.Event, 0, limit)
	for seq := first; seq <= last && len(out) < limit; seq++ {
		if int64(seq) <= afterSeq {
			continue
		}
		msg, err := stream.GetMsg(ctx, seq)
		if err != nil {
			continue
		}
		if msg.Subject != eventSubject(orgID) {
			continue
		}
		event, ok := decodeEvent(msg.Data)
		if !ok {
			continue
		}
		event.Seq = int64(msg.Sequence)
		out = append(out, event)
	}
	return out, nil
}

func decodeEvent(raw []byte) (domain.Event, bool) {
	var envelope messaging.Envelope
	if json.Unmarshal(raw, &envelope) == nil && len(envelope.Payload) > 0 {
		var event domain.Event
		if json.Unmarshal(envelope.Payload, &event) == nil {
			return event, true
		}
	}
	var event domain.Event
	if json.Unmarshal(raw, &event) != nil {
		return domain.Event{}, false
	}
	return event, true
}

func eventSubject(orgID uuid.UUID) string { return fmt.Sprintf("aether.events.org.%s", orgID.String()) }

var _ interface {
	Append(context.Context, uuid.UUID, domain.Event) (int64, error)
	Recent(context.Context, uuid.UUID, int) ([]domain.Event, error)
	Replay(context.Context, uuid.UUID, int64, int) ([]domain.Event, error)
} = (*NATSEventLog)(nil)
