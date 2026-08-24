package nats

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestNATSScheduleEveryPrimitive(t *testing.T) {
	url := os.Getenv("AETHER_NATS_TEST_URL")
	if url == "" {
		t.Skip("AETHER_NATS_TEST_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	conn, err := natsgo.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	js, err := jetstream.New(conn)
	if err != nil {
		t.Fatal(err)
	}
	name := "SCHED_" + fmt.Sprint(time.Now().UnixNano())
	stream, err := js.CreateStream(ctx, jetstream.StreamConfig{Name: name, Subjects: []string{"schedule." + name, "target." + name}, Storage: jetstream.FileStorage, Retention: jetstream.WorkQueuePolicy, AllowMsgSchedules: true})
	if err != nil {
		t.Fatal(err)
	}
	consumer, err := stream.CreateConsumer(ctx, jetstream.ConsumerConfig{FilterSubject: "target." + name})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := js.Publish(ctx, "schedule."+name, []byte("value"), jetstream.WithScheduleEvery(time.Second), jetstream.WithScheduleTarget("target."+name)); err != nil {
		t.Fatal(err)
	}
	msg, err := consumer.Next(jetstream.FetchMaxWait(5 * time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if string(msg.Data()) != "value" {
		t.Fatalf("unexpected scheduled payload: %q", msg.Data())
	}
	_ = msg.Ack()
	_ = js.DeleteStream(ctx, name)
}
