package redis

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"aether/internal/druntime/queue"
)

const (
	queueMaxRetries = 3
	queueBands      = 10
	queueReapIdle   = 10 * time.Minute
)

type Queue struct {
	rt      *Runtime
	mu      sync.Mutex
	streams map[string]struct{}
	reaper  context.CancelFunc
	reapWg  sync.WaitGroup
	started bool
}

func (q *Queue) track(stream string) {
	q.mu.Lock()
	if q.streams == nil {
		q.streams = map[string]struct{}{}
	}
	q.streams[stream] = struct{}{}
	q.mu.Unlock()
}

func (q *Queue) knownStreams() []string {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]string, 0, len(q.streams))
	for s := range q.streams {
		out = append(out, s)
	}
	return out
}

func (q *Queue) bandKey(stream string, priority int) string {
	if priority < 0 {
		priority = 0
	}
	if priority >= queueBands {
		priority = queueBands - 1
	}
	return "q:" + stream + ":p" + itoa(int64(queueBands-1-priority))
}

func (q *Queue) Enqueue(ctx context.Context, stream string, job queue.Job) error {
	q.track(stream)
	if job.ID == "" {
		job.ID = q.newID()
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	body, err := json.Marshal(job)
	if err != nil {
		return err
	}
	key := q.bandKey(stream, job.Priority)
	q.ensureGroup(ctx, key, "workers")
	err = q.rt.client.XAdd(ctx, &redis.XAddArgs{
		Stream: key,
		Values: map[string]any{"body": string(body)},
	}).Err()
	q.rt.observe(err)
	return err
}

func (q *Queue) NewConsumer(ctx context.Context, stream, group, consumerID string) (queue.Consumer, error) {
	q.track(stream)
	for b := 0; b < queueBands; b++ {
		if err := q.ensureGroup(ctx, "q:"+stream+":p"+itoa(int64(b)), group); err != nil {
			return nil, err
		}
	}
	q.startReaper(ctx)
	return &redisConsumer{q: q, stream: stream, group: group, id: consumerID}, nil
}

func (q *Queue) ensureGroup(ctx context.Context, key, group string) error {
	err := q.rt.client.XGroupCreateMkStream(ctx, key, group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		q.rt.observe(err)
		return err
	}
	return nil
}

func (q *Queue) startReaper(ctx context.Context) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.started {
		return
	}
	q.started = true
	rctx, cancel := context.WithCancel(context.Background())
	q.reaper = cancel
	q.reapWg.Add(1)
	go func() {
		defer q.reapWg.Done()
		t := time.NewTicker(queueReapIdle)
		defer t.Stop()
		for {
			select {
			case <-rctx.Done():
				return
			case <-t.C:
				q.reap(rctx)
			}
		}
	}()
}

func (q *Queue) reap(ctx context.Context) {
	group := "workers"
	for _, stream := range q.knownStreams() {
		for b := 0; b < queueBands; b++ {
			key := "q:" + stream + ":p" + itoa(int64(b))
			pending, err := q.rt.client.XPending(ctx, key, group).Result()
			if err != nil {
				continue
			}
			if pending == nil || pending.Count == 0 {
				continue
			}
			entries, _, err := q.rt.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
				Stream:   key,
				Group:    group,
				Consumer: "reaper",
				MinIdle:  queueReapIdle,
			}).Result()
			if err != nil {
				continue
			}
			for _, m := range entries {
				q.requeue(ctx, stream, key, group, m.ID, m.Values["body"])
			}
		}
	}
}

func (q *Queue) requeue(ctx context.Context, stream, key, group, msgID string, raw any) {
	body, _ := raw.(string)
	if body == "" {
		_ = q.rt.client.XAck(ctx, key, group, msgID).Err()
		return
	}
	var job queue.Job
	if err := json.Unmarshal([]byte(body), &job); err != nil {
		_ = q.rt.client.XAck(ctx, key, group, msgID).Err()
		return
	}
	job.Attempt++
	_ = q.rt.client.XAck(ctx, key, group, msgID).Err()
	if job.Attempt >= queueMaxRetries {
		q.deadLetter(ctx, stream, &job)
		return
	}
	nb, _ := json.Marshal(job)
	_ = q.rt.client.XAdd(ctx, &redis.XAddArgs{Stream: q.bandKey(stream, job.Priority), Values: map[string]any{"body": string(nb)}}).Err()
}

func (q *Queue) deadLetter(ctx context.Context, stream string, job *queue.Job) {
	body, _ := json.Marshal(job)
	_ = q.rt.client.XAdd(ctx, &redis.XAddArgs{Stream: "q:" + stream + ":dlq", Values: map[string]any{"body": string(body)}}).Err()
}

type redisConsumer struct {
	q      *Queue
	stream string
	group  string
	id     string
}

func (c *redisConsumer) Next(ctx context.Context) (*queue.Job, error) {
	for {
		for b := 0; b < queueBands; b++ {
			key := "q:" + c.stream + ":p" + itoa(int64(b))
			msgs, err := c.q.rt.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    c.group,
				Consumer: c.id,
				Streams:  []string{key, ">"},
				Count:    1,
				Block:    10 * time.Millisecond,
			}).Result()
			if err != nil {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				continue
			}
			if len(msgs) == 1 && len(msgs[0].Messages) == 1 {
				m := msgs[0].Messages[0]
				var job queue.Job
				if err := json.Unmarshal([]byte(m.Values["body"].(string)), &job); err != nil {
					_ = c.q.rt.client.XAck(ctx, key, c.group, m.ID).Err()
					continue
				}
				job.ID = m.ID
				return &job, nil
			}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
}

func (c *redisConsumer) Ack(ctx context.Context, job *queue.Job) error {
	key := "q:" + c.stream + ":p" + itoa(int64(9-job.Priority))
	return c.q.rt.client.XAck(ctx, key, c.group, job.ID).Err()
}

func (c *redisConsumer) Nack(ctx context.Context, job *queue.Job) error {
	key := "q:" + c.stream + ":p" + itoa(int64(9-job.Priority))
	_ = c.q.rt.client.XAck(ctx, key, c.group, job.ID).Err()
	job.Attempt++
	if job.Attempt >= queueMaxRetries {
		c.q.deadLetter(ctx, c.stream, job)
		return nil
	}
	body, _ := json.Marshal(job)
	return c.q.rt.client.XAdd(ctx, &redis.XAddArgs{Stream: c.q.bandKey(c.stream, job.Priority), Values: map[string]any{"body": string(body)}}).Err()
}

func (c *redisConsumer) Close() error {
	return nil
}

func (q *Queue) Len(ctx context.Context, stream string) (int64, error) {
	var total int64
	for b := 0; b < queueBands; b++ {
		p, err := q.rt.client.XPending(ctx, "q:"+stream+":p"+itoa(int64(b)), "workers").Result()
		if err != nil {
			continue
		}
		if p != nil {
			total += p.Count
		}
	}
	return total, nil
}

func (q *Queue) Pending(ctx context.Context, stream, group string) (int64, error) {
	var total int64
	for b := 0; b < queueBands; b++ {
		p, _ := q.rt.client.XPending(ctx, "q:"+stream+":p"+itoa(int64(b)), group).Result()
		if p != nil {
			total += p.Count
		}
	}
	return total, nil
}

func (q *Queue) DeadLetterLen(ctx context.Context, stream string) (int64, error) {
	return q.rt.client.XLen(ctx, "q:"+stream+":dlq").Result()
}

func (q *Queue) Cancel(ctx context.Context, stream, jobID string) error {
	for b := 0; b < queueBands; b++ {
		key := "q:" + stream + ":p" + itoa(int64(b))
		if err := q.rt.client.XDel(ctx, key, jobID).Err(); err == nil {
			_ = q.rt.client.XAck(ctx, key, "workers", jobID).Err()
		}
	}
	return nil
}

func (q *Queue) newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + fmt.Sprintf("%x", b)
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}

func (q *Queue) Close(ctx context.Context) {
	q.mu.Lock()
	if q.reaper != nil {
		q.reaper()
	}
	q.mu.Unlock()
	q.reapWg.Wait()
}
