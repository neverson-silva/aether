package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		key  string
		want string
		err  error
	}{
		{"", "", ErrInvalidObjectKey},
		{"a\x00b", "", ErrInvalidObjectKey},
		{"/backups/db.sql", "backups/db.sql", nil},
		{"backups/My Database.sql", "backups/My Database.sql", nil},
		{"backups/2026/database.sql.gz", "backups/2026/database.sql.gz", nil},
		{"../escape", "", ErrInvalidObjectKey},
		{"a/../b", "", ErrInvalidObjectKey},
		{"a/./b", "", ErrInvalidObjectKey},
		{"a//b", "", ErrInvalidObjectKey},
		{"a/", "", ErrInvalidObjectKey},
		{"/", "", ErrInvalidObjectKey},
		{"leading-space.txt", "leading-space.txt", nil},
		{"ünïcödé/файл.txt", "ünïcödé/файл.txt", nil},
	}
	for _, tt := range tests {
		got, err := NormalizeKey(tt.key)
		if !errors.Is(err, tt.err) {
			t.Errorf("NormalizeKey(%q) err = %v, want %v", tt.key, err, tt.err)
			continue
		}
		if got != tt.want {
			t.Errorf("NormalizeKey(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

type memObject struct {
	content     []byte
	contentType string
	metadata    map[string]string
	modTime     time.Time
}

type memProvider struct {
	mu      sync.Mutex
	objects map[string]*memObject
}

func newMemProvider() *memProvider {
	return &memProvider{objects: map[string]*memObject{}}
}

func (m *memProvider) Capabilities() Capabilities {
	return Capabilities{Streaming: true, CopyObject: true, Metadata: true}
}

func (m *memProvider) PutObject(_ context.Context, input PutObjectInput) (*PutObjectOutput, error) {
	key, err := NormalizeKey(input.Key)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[key] = &memObject{content: data, contentType: input.ContentType, metadata: input.Metadata, modTime: time.Now()}
	return &PutObjectOutput{Key: key, Size: int64(len(data))}, nil
}

func (m *memProvider) GetObject(_ context.Context, input GetObjectInput) (*GetObjectOutput, error) {
	m.mu.Lock()
	obj, ok := m.objects[input.Key]
	if !ok {
		m.mu.Unlock()
		return nil, ErrObjectNotFound
	}
	m.mu.Unlock()
	return &GetObjectOutput{
		Key: input.Key, Body: io.NopCloser(strings.NewReader(string(obj.content))),
		ContentType: obj.contentType, ContentLength: int64(len(obj.content)),
		LastModified: obj.modTime, Metadata: obj.metadata,
	}, nil
}

func (m *memProvider) HeadObject(_ context.Context, input HeadObjectInput) (*HeadObjectOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	obj, ok := m.objects[input.Key]
	if !ok {
		return nil, ErrObjectNotFound
	}
	return &HeadObjectOutput{
		Key: input.Key, ContentLength: int64(len(obj.content)), ContentType: obj.contentType,
		LastModified: obj.modTime, Metadata: obj.metadata,
	}, nil
}

func (m *memProvider) DeleteObject(_ context.Context, input DeleteObjectInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objects, input.Key)
	return nil
}

func (m *memProvider) ListObjects(_ context.Context, input ListObjectsInput) (*ListObjectsOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := &ListObjectsOutput{}
	limit := input.Limit
	if limit <= 0 {
		limit = 1000
	}
	start := 0
	if input.Cursor != "" {
		var c int
		if _, err := fmt.Sscanf(input.Cursor, "%d", &c); err != nil {
			return nil, ErrInvalidObjectKey
		}
		start = c
	}
	keys := make([]string, 0, len(m.objects))
	for k := range m.objects {
		if strings.HasPrefix(k, input.Prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for i := start; i < len(keys) && len(out.Objects) < limit; i++ {
		obj := m.objects[keys[i]]
		out.Objects = append(out.Objects, ObjectInfo{
			Key: keys[i], Size: int64(len(obj.content)), ContentType: obj.contentType, LastModified: obj.modTime,
		})
	}
	if start+len(out.Objects) < len(keys) {
		out.NextCursor = fmt.Sprint(start + len(out.Objects))
	}
	return out, nil
}

func (m *memProvider) CopyObject(ctx context.Context, input CopyObjectInput) (*CopyObjectOutput, error) {
	src, err := m.GetObject(ctx, GetObjectInput{Key: input.SourceKey})
	if err != nil {
		return nil, err
	}
	data, _ := io.ReadAll(src.Body)
	src.Body.Close()
	if _, err := m.PutObject(ctx, PutObjectInput{Key: input.DestinationKey, Body: strings.NewReader(string(data)), ContentType: src.ContentType}); err != nil {
		return nil, err
	}
	return &CopyObjectOutput{SourceKey: input.SourceKey, DestinationKey: input.DestinationKey}, nil
}

func TestContractAgainstMemProvider(t *testing.T) {
	CheckProviderContract(t, newMemProvider())
}

func TestStorageFacade(t *testing.T) {
	mem := newMemProvider()
	s := New(mem)

	ctx := context.Background()
	if _, err := s.PutObject(ctx, PutObjectInput{Key: "k", Body: strings.NewReader("data")}); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	obj, err := s.GetObject(ctx, GetObjectInput{Key: "k"})
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	data, _ := io.ReadAll(obj.Body)
	obj.Body.Close()
	if string(data) != "data" {
		t.Errorf("facade get = %q", data)
	}
	if _, err := s.HeadObject(ctx, HeadObjectInput{Key: "k"}); err != nil {
		t.Errorf("HeadObject: %v", err)
	}
	if err := s.DeleteObject(ctx, DeleteObjectInput{Key: "k"}); err != nil {
		t.Errorf("DeleteObject: %v", err)
	}
	if _, err := s.CopyObject(ctx, CopyObjectInput{SourceKey: "k", DestinationKey: "k2"}); !errors.Is(err, ErrObjectNotFound) {
		t.Errorf("CopyObject missing = %v, want ErrObjectNotFound", err)
	}
	if !s.Capabilities().Streaming {
		t.Error("facade capabilities not forwarded")
	}
}
