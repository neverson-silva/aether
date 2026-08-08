package obs

import (
	"errors"
	"os"
	"sync"
)

type LiveLog struct {
	mu      sync.Mutex
	path    string
	f       *os.File
	subs    map[int]chan []byte
	next    int
	pub     func([]byte)
	ringMax int
	ring    []byte
}

func NewLiveLog(path string) (*LiveLog, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o640)
	if err != nil {
		return nil, err
	}
	return &LiveLog{path: path, f: f, subs: map[int]chan []byte{}}, nil
}

func (l *LiveLog) SetTailBuffer(maxBytes int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ringMax = maxBytes
	if len(l.ring) > maxBytes {
		l.ring = l.ring[len(l.ring)-maxBytes:]
	}
}

func (l *LiveLog) SetPublisher(pub func([]byte)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pub = pub
}

func (l *LiveLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return 0, errors.New("log closed")
	}
	n, err := l.f.Write(p)
	if err == nil {
		l.f.Sync()
	}
	if l.ringMax > 0 {
		l.ring = append(l.ring, p...)
		if len(l.ring) > l.ringMax {
			l.ring = l.ring[len(l.ring)-l.ringMax:]
		}
	}
	if l.pub != nil {
		l.pub(p)
	}
	for _, ch := range l.subs {
		select {
		case ch <- append([]byte(nil), p...):
		default:
		}
	}
	return n, err
}

func (l *LiveLog) Subscribe() (int, <-chan []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	id := l.next
	l.next++
	ch := make(chan []byte, 256)
	l.subs[id] = ch
	return id, ch
}

func (l *LiveLog) Unsubscribe(id int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if ch, ok := l.subs[id]; ok {
		delete(l.subs, id)
		close(ch)
	}
}

func (l *LiveLog) Tail(limitBytes int) ([]byte, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ringMax > 0 && len(l.ring) > 0 {
		from := 0
		if limitBytes > 0 && len(l.ring) > limitBytes {
			from = len(l.ring) - limitBytes
		}
		return append([]byte(nil), l.ring[from:]...), nil
	}
	if l.f == nil {
		return nil, errors.New("log closed")
	}
	info, err := l.f.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()
	offset := size - int64(limitBytes)
	if offset < 0 {
		offset = 0
	}
	buf := make([]byte, size-offset)
	_, err = l.f.ReadAt(buf, offset)
	return buf, err
}

func (l *LiveLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return nil
	}
	err := l.f.Close()
	l.f = nil
	for id, ch := range l.subs {
		delete(l.subs, id)
		close(ch)
	}
	return err
}

func (l *LiveLog) Path() string { return l.path }
