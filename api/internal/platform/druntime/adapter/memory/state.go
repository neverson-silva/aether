package memory

import (
	"context"
	"strings"
	"sync"
	"time"
)

type stateEntry struct {
	value []byte
	exp   time.Time
}

type stateUpdate struct {
	key   string
	value []byte
}

type watcher struct {
	pattern string
	ch      chan stateUpdate
}

type State struct {
	mu       sync.Mutex
	data     map[string]stateEntry
	watchers []*watcher
}

func NewState() *State {
	return &State{data: map[string]stateEntry{}}
}

func matchPattern(pattern, key string) bool {
	if !strings.Contains(pattern, "*") {
		return pattern == key
	}
	parts := strings.Split(pattern, "*")
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			if !strings.HasPrefix(key, p) {
				return false
			}
		} else if i == len(parts)-1 {
			return strings.HasSuffix(key, p)
		} else if idx := strings.Index(key, p); idx == -1 {
			return false
		} else {
			key = key[idx+len(p):]
		}
	}
	return true
}

func (s *State) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	s.mu.Lock()
	exp := time.Time{}
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	s.data[key] = stateEntry{value: value, exp: exp}
	upd := stateUpdate{key: key, value: value}
	for _, w := range s.watchers {
		if matchPattern(w.pattern, key) {
			select {
			case w.ch <- upd:
			default:
			}
		}
	}
	s.mu.Unlock()
	return nil
}

func (s *State) Get(_ context.Context, key string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if !ok || (!e.exp.IsZero() && time.Now().After(e.exp)) {
		if ok {
			delete(s.data, key)
		}
		return nil, false, nil
	}
	return e.value, true, nil
}

func (s *State) Del(ctx context.Context, key string) error {
	s.mu.Lock()
	delete(s.data, key)
	upd := stateUpdate{key: key}
	for _, w := range s.watchers {
		if matchPattern(w.pattern, key) {
			select {
			case w.ch <- upd:
			default:
			}
		}
	}
	s.mu.Unlock()
	return nil
}

func (s *State) Changes(ctx context.Context, keyPattern string, h func(ctx context.Context, key string, value []byte)) (func(), error) {
	w := &watcher{pattern: keyPattern, ch: make(chan stateUpdate, 128)}
	s.mu.Lock()
	s.watchers = append(s.watchers, w)
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case u := <-w.ch:
				h(ctx, u.key, u.value)
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			s.mu.Lock()
			for i, ww := range s.watchers {
				if ww == w {
					s.watchers = append(s.watchers[:i], s.watchers[i+1:]...)
					break
				}
			}
			s.mu.Unlock()
		})
	}, nil
}
