package infra

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"aether/internal/modules/realtime/domain"
)

func TestNATSEventLogIntegration(t *testing.T) {
	url := os.Getenv("AETHER_NATS_TEST_URL")
	if url == "" {
		t.Skip("AETHER_NATS_TEST_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name: "AETHER_EVENTS", Subjects: []string{"aether.events.>"}, Storage: jetstream.FileStorage,
		Retention: jetstream.LimitsPolicy, MaxAge: 24 * time.Hour, MaxMsgsPerSubject: 5000,
	}); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	conn.Close()

	log, err := NewNATSEventLog(url, "aether-eventlog-test")
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	orgID := uuid.New()
	otherOrgID := uuid.New()
	firstID := uuid.NewString()
	secondID := uuid.NewString()
	otherID := uuid.NewString()
	firstSeq, err := log.Append(ctx, orgID, domain.Event{ID: firstID, Type: "integration.first", TS: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	secondSeq, err := log.Append(ctx, orgID, domain.Event{ID: secondID, Type: "integration.second", TS: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := log.Append(ctx, otherOrgID, domain.Event{ID: otherID, Type: "integration.other", TS: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if secondSeq <= firstSeq {
		t.Fatalf("event sequence did not advance: first=%d second=%d", firstSeq, secondSeq)
	}

	recent, err := log.Recent(ctx, orgID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !containsEvent(recent, firstID) || !containsEvent(recent, secondID) || containsEvent(recent, otherID) {
		t.Fatalf("recent events are not scoped correctly: %+v", recent)
	}
	replay, err := log.Replay(ctx, orgID, firstSeq, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !containsEvent(replay, secondID) || containsEvent(replay, firstID) || containsEvent(replay, otherID) {
		t.Fatalf("replay cursor is not scoped correctly: %+v", replay)
	}
}

func containsEvent(events []domain.Event, id string) bool {
	for _, event := range events {
		if event.ID == id {
			return true
		}
	}
	return false
}
