package nats

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"aether/internal/platform/druntime"
	"aether/internal/platform/druntime/queue"
	"aether/internal/platform/druntime/scheduler"
	"aether/internal/platform/messaging"
)

const (
	jobsStream   = "AETHER_JOBS"
	eventsStream = "AETHER_EVENTS"
	dlqStream    = "AETHER_DLQ"
	stateBucket  = "AETHER_STATE"
	locksBucket  = "AETHER_LOCKS"
)

type Runtime struct {
	conn  *natsgo.Conn
	js    jetstream.JetStream
	state jetstream.KeyValue
	locks jetstream.KeyValue
	owned bool
	mu    sync.Mutex
}

func New(ctx context.Context, cfg druntime.Config) (*druntime.Runtime, error) {
	url := cfg.NATSURL
	if url == "" {
		url = natsgo.DefaultURL
	}
	options := []natsgo.Option{natsgo.Name(cfg.NATSName)}
	if cfg.NATSUser != "" || cfg.NATSPassword != "" {
		options = append(options, natsgo.UserInfo(cfg.NATSUser, cfg.NATSPassword))
	}
	conn, err := natsgo.Connect(url, options...)
	if err != nil {
		return nil, err
	}
	if err := validateServerVersion(conn.ConnectedServerVersion()); err != nil {
		conn.Close()
		return nil, err
	}
	runtime, err := NewWithConn(ctx, cfg, conn)
	if err != nil {
		_ = conn.Drain()
		return nil, err
	}
	runtimeCloser(runtime, conn)
	return runtime, nil
}

func validateServerVersion(value string) error {
	parts := strings.Split(strings.TrimPrefix(value, "v"), ".")
	if len(parts) < 2 {
		return fmt.Errorf("unsupported NATS version %q", value)
	}
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	if majorErr != nil || minorErr != nil || major < 2 || major == 2 && minor < 14 {
		return fmt.Errorf("NATS %s is unsupported; minimum version is 2.14.0", value)
	}
	return nil
}

func NewWithConn(ctx context.Context, cfg druntime.Config, conn *natsgo.Conn) (*druntime.Runtime, error) {
	if err := validateServerVersion(conn.ConnectedServerVersion()); err != nil {
		return nil, err
	}
	js, err := jetstream.New(conn)
	if err != nil {
		return nil, err
	}
	if err := ensureStreams(ctx, js); err != nil {
		return nil, err
	}
	state, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: stateBucket, History: 1})
	if err != nil {
		return nil, err
	}
	locks, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: locksBucket, History: 1})
	if err != nil {
		return nil, err
	}
	rt := &Runtime{conn: conn, js: js, state: state, locks: locks}
	runtime := &druntime.Runtime{
		Backend:   "nats",
		PubSub:    &PubSub{rt: rt},
		Cache:     NewCache(),
		Queue:     &Queue{rt: rt},
		Locks:     &LockManager{rt: rt},
		RateLimit: NewRateLimiter(),
		Presence:  &Presence{rt: rt},
		Events:    &EventBus{rt: rt},
		State:     &State{rt: rt},
		Scheduler: &Scheduler{rt: rt},
	}
	runtime.SetCloser(func(ctx context.Context) error { return rt.Close(ctx) })
	return runtime, nil
}

func runtimeCloser(runtime *druntime.Runtime, conn *natsgo.Conn) {
	runtime.SetCloser(func(context.Context) error { return conn.Drain() })
}

type Scheduler struct {
	rt *Runtime
}

func (s *Scheduler) ScheduleAt(ctx context.Context, key string, at time.Time, payload []byte) error {
	return s.ScheduleJobAt(ctx, jobSubject("backups"), key, "backup.schedule", at, payload)
}

func (s *Scheduler) ScheduleJobAt(ctx context.Context, subject, key, jobType string, at time.Time, payload []byte) error {
	jobPayload, err := json.Marshal(queue.Job{ID: "schedule:" + key, Type: jobType, Payload: payload, CreatedAt: time.Now().UTC()})
	if err != nil {
		return err
	}
	envelope, err := json.Marshal(messaging.Envelope{ID: "schedule:" + key, Type: jobType, SchemaVersion: 1, CreatedAt: time.Now().UTC(), Payload: jobPayload})
	if err != nil {
		return err
	}
	scheduleSubject := messaging.Jobs("schedules." + strings.NewReplacer(".", "_", ">", "_", "*", "_").Replace(subject))
	_, err = s.rt.js.Publish(ctx, scheduleSubject, envelope,
		jetstream.WithMsgID("schedule:"+key+":"+at.UTC().Format(time.RFC3339Nano)),
		jetstream.WithScheduleAt(at), jetstream.WithScheduleTarget(subject))
	return err
}

func (s *Scheduler) ScheduleJobCron(ctx context.Context, subject, key, jobType, expression, timezone string, payload []byte) error {
	envelope, err := recurringEnvelope(key, jobType, payload)
	if err != nil {
		return err
	}
	options := []jetstream.PublishOpt{jetstream.WithScheduleCron(expression), jetstream.WithScheduleTarget(subject)}
	if timezone != "" {
		options = append(options, jetstream.WithScheduleTimeZone(timezone))
	}
	return s.publishRecurring(ctx, key, envelope, options...)
}

func (s *Scheduler) ScheduleJobEvery(ctx context.Context, subject, key, jobType string, interval time.Duration, payload []byte) error {
	envelope, err := recurringEnvelope(key, jobType, payload)
	if err != nil {
		return err
	}
	return s.publishRecurring(ctx, key, envelope, jetstream.WithScheduleEvery(interval), jetstream.WithScheduleTarget(subject))
}

func recurringEnvelope(key, jobType string, payload []byte) ([]byte, error) {
	jobPayload, err := json.Marshal(queue.Job{ID: "schedule:" + key, Type: jobType, Payload: payload, CreatedAt: time.Now().UTC()})
	if err != nil {
		return nil, err
	}
	return json.Marshal(messaging.Envelope{ID: "schedule:" + key, Type: jobType, SchemaVersion: 1, CreatedAt: time.Now().UTC(), Payload: jobPayload})
}

func (s *Scheduler) publishRecurring(ctx context.Context, key string, envelope []byte, options ...jetstream.PublishOpt) error {
	source := recurringScheduleSubject(key)
	stream, err := s.rt.js.Stream(ctx, jobsStream)
	if err != nil {
		return err
	}
	if err := stream.Purge(ctx, jetstream.WithPurgeSubject(source)); err != nil {
		return err
	}
	options = append([]jetstream.PublishOpt{jetstream.WithMsgID(recurringMessageID(key))}, options...)
	if _, err := s.rt.js.Publish(ctx, source, envelope, options...); err != nil {
		return err
	}
	_, err = s.rt.state.Put(ctx, recurringStateKey(key), []byte(key))
	return err
}

func (s *Scheduler) ReconcileRecurring(ctx context.Context, namespace string, activeKeys []string) error {
	active := make(map[string]struct{}, len(activeKeys))
	for _, key := range activeKeys {
		active[key] = struct{}{}
	}
	keys, err := s.rt.state.Keys(ctx)
	if err != nil {
		return err
	}
	for _, stateKey := range keys {
		if !strings.HasPrefix(stateKey, recurringStatePrefix) {
			continue
		}
		entry, err := s.rt.state.Get(ctx, stateKey)
		if err != nil {
			if err == jetstream.ErrKeyNotFound {
				continue
			}
			return err
		}
		key := string(entry.Value())
		if !strings.HasPrefix(key, namespace+":") {
			continue
		}
		if _, ok := active[key]; ok {
			continue
		}
		if err := s.unschedule(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (s *Scheduler) unschedule(ctx context.Context, key string) error {
	stream, err := s.rt.js.Stream(ctx, jobsStream)
	if err != nil {
		return err
	}
	if err := stream.Purge(ctx, jetstream.WithPurgeSubject(recurringScheduleSubject(key))); err != nil {
		return err
	}
	return s.rt.state.Delete(ctx, recurringStateKey(key))
}

const recurringStatePrefix = "scheduler.recurring."

func recurringStateKey(key string) string {
	return recurringStatePrefix + fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
}

func recurringScheduleSubject(key string) string {
	return messaging.Jobs("schedules.recurring." + fmt.Sprintf("%x", sha256.Sum256([]byte(key))))
}

func recurringMessageID(key string) string {
	return "recurring:" + key + ":" + time.Now().UTC().Format(time.RFC3339Nano)
}

var _ scheduler.Scheduler = (*Scheduler)(nil)
var _ scheduler.RecurringScheduler = (*Scheduler)(nil)

func ensureStreams(ctx context.Context, js jetstream.JetStream) error {
	for _, cfg := range []jetstream.StreamConfig{
		{Name: jobsStream, Subjects: []string{messaging.JobsPrefix + ">"}, Storage: jetstream.FileStorage, Retention: jetstream.WorkQueuePolicy, MaxAge: 7 * 24 * 60 * 60 * 1e9, AllowMsgSchedules: true},
		{Name: eventsStream, Subjects: []string{messaging.EventsPrefix + ">"}, Storage: jetstream.FileStorage, Retention: jetstream.LimitsPolicy, MaxAge: 24 * 60 * 60 * 1e9, MaxMsgsPerSubject: 5000},
		{Name: dlqStream, Subjects: []string{messaging.DLQPrefix + ">"}, Storage: jetstream.FileStorage, Retention: jetstream.LimitsPolicy, MaxAge: 30 * 24 * 60 * 60 * 1e9, MaxMsgsPerSubject: 10000},
	} {
		if _, err := js.CreateOrUpdateStream(ctx, cfg); err != nil {
			return fmt.Errorf("ensure NATS stream %s: %w", cfg.Name, err)
		}
	}
	return nil
}

func (r *Runtime) Close(ctx context.Context) error {
	if !r.owned {
		return nil
	}
	if err := r.conn.Drain(); err != nil {
		r.conn.Close()
		return err
	}
	return nil
}
