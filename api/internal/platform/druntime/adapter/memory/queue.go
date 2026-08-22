package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"aether/internal/platform/druntime/queue"
)

const (
	maxRetries = 3
)

var ErrQueueClosed = errors.New("queue closed")

type qJob struct {
	job      queue.Job
	inFlight string
}

type groupState struct {
	pending  map[string]*qJob
	inflight map[string]*qJob
}

type streamState struct {
	jobs   map[string]*qJob
	groups map[string]*groupState
	dlq    map[string]*qJob
}

type Queue struct {
	mu      sync.Mutex
	streams map[string]*streamState
	seq     int64
}

func NewQueue() *Queue {
	return &Queue{streams: map[string]*streamState{}}
}

func (q *Queue) stream(name string) *streamState {
	s, ok := q.streams[name]
	if !ok {
		s = &streamState{jobs: map[string]*qJob{}, groups: map[string]*groupState{}, dlq: map[string]*qJob{}}
		q.streams[name] = s
	}
	return s
}

func (q *Queue) Enqueue(_ context.Context, stream string, job queue.Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if job.ID == "" {
		q.seq++
		job.ID = time.Now().Format("20060102150405.000000000") + "-" + itoa(q.seq)
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	s := q.stream(stream)
	j := &qJob{job: job}
	s.jobs[job.ID] = j
	for _, g := range s.groups {
		g.pending[job.ID] = j
	}
	return nil
}

type consumer struct {
	q          *Queue
	stream     string
	group      string
	consumerID string
	closed     bool
}

func (q *Queue) NewConsumer(_ context.Context, stream, group, consumerID string) (queue.Consumer, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	s := q.stream(stream)
	if s.groups[group] == nil {
		s.groups[group] = &groupState{pending: map[string]*qJob{}, inflight: map[string]*qJob{}}
	}
	return &consumer{q: q, stream: stream, group: group, consumerID: consumerID}, nil
}

func (c *consumer) Next(ctx context.Context) (*queue.Job, error) {
	for {
		c.q.mu.Lock()
		if c.closed {
			c.q.mu.Unlock()
			return nil, ErrQueueClosed
		}
		s := c.q.stream(c.stream)
		g := s.groups[c.group]
		if g == nil {
			c.q.mu.Unlock()
			return nil, errors.New("group does not exist")
		}
		for id, j := range s.jobs {
			if _, ok := g.pending[id]; ok {
				continue
			}
			if _, ok := g.inflight[id]; ok {
				continue
			}
			g.pending[id] = j
		}
		var best *qJob
		for _, j := range g.pending {
			if j.inFlight != "" {
				continue
			}
			if best == nil || qLess(j.job, best.job) {
				best = j
			}
		}
		if best != nil {
			best.inFlight = c.consumerID
			g.inflight[best.job.ID] = best
			delete(g.pending, best.job.ID)
			job := best.job
			c.q.mu.Unlock()
			return &job, nil
		}
		c.q.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (c *consumer) Ack(_ context.Context, job *queue.Job) error {
	c.q.mu.Lock()
	defer c.q.mu.Unlock()
	if c.closed {
		return ErrQueueClosed
	}
	s := c.q.stream(c.stream)
	g := s.groups[c.group]
	if g == nil {
		return nil
	}
	delete(g.inflight, job.ID)
	delete(s.jobs, job.ID)
	return nil
}

func (c *consumer) Nack(_ context.Context, job *queue.Job) error {
	c.q.mu.Lock()
	defer c.q.mu.Unlock()
	if c.closed {
		return ErrQueueClosed
	}
	s := c.q.stream(c.stream)
	g := s.groups[c.group]
	if g == nil {
		return nil
	}
	j := g.inflight[job.ID]
	delete(g.inflight, job.ID)
	if j == nil {
		return nil
	}
	j.job.Attempt++
	if j.job.Attempt >= maxRetries {
		s.dlq[job.ID] = j
		delete(s.jobs, job.ID)
		return nil
	}
	j.inFlight = ""
	g.pending[job.ID] = j
	return nil
}

func (c *consumer) Close() error {
	c.q.mu.Lock()
	defer c.q.mu.Unlock()
	c.closed = true
	return nil
}

func (q *Queue) Len(_ context.Context, stream string) (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	s := q.stream(stream)
	return int64(len(s.jobs)), nil
}

func (q *Queue) Pending(_ context.Context, stream, group string) (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	s := q.stream(stream)
	if s.groups[group] == nil {
		return 0, nil
	}
	return int64(len(s.groups[group].pending) + len(s.groups[group].inflight)), nil
}

func (q *Queue) DeadLetterLen(_ context.Context, stream string) (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	s := q.stream(stream)
	return int64(len(s.dlq)), nil
}

func (q *Queue) Cancel(_ context.Context, stream, jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	s := q.stream(stream)
	if _, ok := s.jobs[jobID]; !ok {
		return nil
	}
	for _, g := range s.groups {
		delete(g.pending, jobID)
		delete(g.inflight, jobID)
	}
	delete(s.jobs, jobID)
	return nil
}

func qLess(a, b queue.Job) bool {
	if a.Priority != b.Priority {
		return a.Priority > b.Priority
	}
	return a.CreatedAt.Before(b.CreatedAt)
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
