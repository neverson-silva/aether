package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"aether/internal/platform/messaging"
)

func TestJetStreamJobRedeliveryAfterWorkerCrash(t *testing.T) {
	url := os.Getenv("AETHER_NATS_TEST_URL")
	if url == "" {
		t.Skip("AETHER_NATS_TEST_URL is not set")
	}

	jobTypes := []string{
		"deploy.execute",
		"backup.create",
		"restore.execute",
		"snapshot.create",
		"cleanup.execute",
		"deploy.cancel",
	}
	for _, jobType := range jobTypes {
		t.Run(jobType, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			streamSubject := "aether.jobs.crash-recovery"
			consumerName := fmt.Sprintf("crash-%d", time.Now().UnixNano())
			jobID := fmt.Sprintf("crash-job-%s-%d", jobType, time.Now().UnixNano())
			payload := []byte(`{"job":"replayed"}`)
			envelope := messaging.Envelope{ID: jobID, Type: jobType, SchemaVersion: 1, CreatedAt: time.Now().UTC(), Payload: payload}

			connA, err := natsgo.Connect(url, natsgo.Name("aether-crash-worker-a"))
			if err != nil {
				t.Fatal(err)
			}
			jsA, err := jetstream.New(connA)
			if err != nil {
				connA.Close()
				t.Fatal(err)
			}
			consumer, err := jsA.CreateOrUpdateConsumer(ctx, jobsStream, jetstream.ConsumerConfig{
				Name:          consumerName,
				Durable:       consumerName,
				FilterSubject: streamSubject,
				AckPolicy:     jetstream.AckExplicitPolicy,
				AckWait:       150 * time.Millisecond,
				MaxDeliver:    maxJobDeliveries,
				DeliverPolicy: jetstream.DeliverAllPolicy,
			})
			if err != nil {
				connA.Close()
				t.Fatal(err)
			}
			defer func() {
				connA.Close()
				conn, connectErr := natsgo.Connect(url)
				if connectErr == nil {
					js, jsErr := jetstream.New(conn)
					if jsErr == nil {
						_ = js.DeleteConsumer(context.Background(), jobsStream, consumerName)
					}
					conn.Close()
				}
			}()

			raw, err := json.Marshal(envelope)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := jsA.Publish(ctx, streamSubject, raw, jetstream.WithMsgID(jobID)); err != nil {
				t.Fatal(err)
			}
			batch, err := consumer.Fetch(1, jetstream.FetchMaxWait(2*time.Second))
			if err != nil {
				t.Fatal(err)
			}
			var first jetstream.Msg
			for msg := range batch.Messages() {
				first = msg
				break
			}
			if first == nil {
				t.Fatal("worker A did not receive the job")
			}
			connA.Close()

			connB, err := natsgo.Connect(url, natsgo.Name("aether-crash-worker-b"))
			if err != nil {
				t.Fatal(err)
			}
			defer connB.Close()
			jsB, err := jetstream.New(connB)
			if err != nil {
				t.Fatal(err)
			}
			consumerB, err := jsB.Consumer(ctx, jobsStream, consumerName)
			if err != nil {
				t.Fatal(err)
			}
			batch, err = consumerB.Fetch(1, jetstream.FetchMaxWait(3*time.Second))
			if err != nil {
				t.Fatal(err)
			}
			var replayed jetstream.Msg
			for msg := range batch.Messages() {
				replayed = msg
				break
			}
			if replayed == nil {
				t.Fatal("worker B did not receive the unacknowledged job")
			}
			if string(replayed.Data()) != string(raw) {
				t.Fatalf("redelivered payload differs: got %q want %q", replayed.Data(), raw)
			}
			if err := replayed.Ack(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
