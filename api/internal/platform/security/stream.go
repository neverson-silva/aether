package security

import (
	"errors"
	"sync"

	"github.com/google/uuid"
)

var ErrStreamLimit = errors.New("stream limit reached")

type StreamLimiter struct {
	limit  int
	mu     sync.Mutex
	active map[uuid.UUID]int
}

func NewStreamLimiter(limit int) *StreamLimiter {
	if limit < 1 {
		limit = 1
	}
	return &StreamLimiter{limit: limit, active: make(map[uuid.UUID]int)}
}

func (l *StreamLimiter) Acquire(orgID uuid.UUID) (func(), error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.active[orgID] >= l.limit {
		return nil, ErrStreamLimit
	}
	l.active[orgID]++
	released := false
	return func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		if released {
			return
		}
		released = true
		l.active[orgID]--
		if l.active[orgID] == 0 {
			delete(l.active, orgID)
		}
	}, nil
}
