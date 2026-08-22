package application

import (
	"context"
	"sync"
	"testing"
	"time"

	"aether/internal/platform/database/adapter"
	"aether/internal/platform/druntime/cache"
)

type fakeCache struct {
	mu   sync.Mutex
	data map[string]string
}

func newFakeCache() *fakeCache {
	return &fakeCache{data: map[string]string{}}
}

func (c *fakeCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.data[key]
	if !ok {
		return nil, false, nil
	}
	return []byte(v), true, nil
}

func (c *fakeCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = string(value)
	return nil
}

func (c *fakeCache) Add(_ context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	return false, nil
}

func (c *fakeCache) Del(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
	return nil
}

func (c *fakeCache) Invalidate(_ context.Context, prefix string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.data {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.data, k)
		}
	}
	return nil
}

func (c *fakeCache) Metrics(_ context.Context) cache.CacheMetrics { return cache.CacheMetrics{} }

type fakeAdapter struct {
	metaCalls    int
	schemaCalls  int
	objectCalls  int
	tableCalls   int
	execCalls    int
	version      string
	schemas      []string
	objects      map[string][]adapter.Object
	counts       map[string]adapter.ObjectSummary
	tableDetails map[string]adapter.TableDetail
}

func (f *fakeAdapter) Engine() string { return "postgres" }

func (f *fakeAdapter) IntrospectMeta(_ context.Context) (adapter.Meta, error) {
	f.metaCalls++
	return adapter.Meta{Engine: "postgres", Version: f.version, Status: "healthy", Schemas: len(f.schemas)}, nil
}

func (f *fakeAdapter) IntrospectSchemas(_ context.Context) ([]string, error) {
	f.schemaCalls++
	return f.schemas, nil
}

func (f *fakeAdapter) IntrospectObjects(_ context.Context, schema string) (adapter.ObjectSummary, error) {
	f.objectCalls++
	return f.counts[schema], nil
}

func (f *fakeAdapter) ListObjects(_ context.Context, schema string) ([]adapter.Object, error) {
	return f.objects[schema], nil
}

func (f *fakeAdapter) IntrospectTable(_ context.Context, schema, table string) (adapter.TableDetail, error) {
	f.tableCalls++
	return f.tableDetails[schema+"."+table], nil
}

func (f *fakeAdapter) TableData(_ context.Context, schema, table string, opts adapter.QueryOptions) (adapter.QueryResult, error) {
	return adapter.QueryResult{}, nil
}

func (f *fakeAdapter) Query(_ context.Context, sql string, opts adapter.QueryOptions) (adapter.QueryResult, error) {
	return adapter.QueryResult{}, nil
}

func (f *fakeAdapter) Exec(_ context.Context, sql string) (adapter.ExecResult, error) {
	f.execCalls++
	return adapter.ExecResult{Message: "OK", CommandTag: "INSERT 0 1"}, nil
}

func newFake() (*fakeAdapter, *fakeCache) {
	f := &fakeAdapter{
		version: "16.15",
		schemas: []string{"public", "app"},
		objects: map[string][]adapter.Object{
			"public": {{Name: "users", Type: "table"}, {Name: "v_users", Type: "view"}},
			"app":    {{Name: "orders", Type: "table"}},
		},
		counts: map[string]adapter.ObjectSummary{
			"public": {Tables: 1, Views: 1},
			"app":    {Tables: 1},
		},
		tableDetails: map[string]adapter.TableDetail{
			"public.users": {Schema: "public", Name: "users", Type: "table", Owner: "app"},
		},
	}
	return f, newFakeCache()
}

func TestCachedAdapterCatalogServedFromCache(t *testing.T) {
	f, c := newFake()
	ca := &cachedAdapter{Adapter: f, cache: c, dbID: "db-1", ttl: time.Minute}

	meta, err := ca.IntrospectMeta(context.Background())
	if err != nil {
		t.Fatalf("meta: %v", err)
	}
	if meta.Version != "16.15" || meta.Tables != 2 || meta.Views != 1 {
		t.Fatalf("unexpected meta: %+v", meta)
	}
	schemas, err := ca.IntrospectSchemas(context.Background())
	if err != nil {
		t.Fatalf("schemas: %v", err)
	}
	if len(schemas) != 2 {
		t.Fatalf("schemas: %v", schemas)
	}
	objs, err := ca.ListObjects(context.Background(), "public")
	if err != nil {
		t.Fatalf("objects: %v", err)
	}
	if len(objs) != 2 {
		t.Fatalf("objects: %v", objs)
	}

	ca.IntrospectMeta(context.Background())
	ca.IntrospectSchemas(context.Background())
	ca.ListObjects(context.Background(), "public")

	if f.metaCalls != 1 {
		t.Fatalf("meta introspected %d times, want 1", f.metaCalls)
	}
	if f.schemaCalls != 1 {
		t.Fatalf("schemas introspected %d times, want 1", f.schemaCalls)
	}
}

func TestCachedAdapterTableDetailCached(t *testing.T) {
	f, c := newFake()
	ca := &cachedAdapter{Adapter: f, cache: c, dbID: "db-1", ttl: time.Minute}

	ca.IntrospectTable(context.Background(), "public", "users")
	ca.IntrospectTable(context.Background(), "public", "users")

	if f.tableCalls != 1 {
		t.Fatalf("table introspected %d times, want 1", f.tableCalls)
	}
}

func TestCachedAdapterExecInvalidates(t *testing.T) {
	f, c := newFake()
	ca := &cachedAdapter{Adapter: f, cache: c, dbID: "db-1", ttl: time.Minute}

	ca.IntrospectMeta(context.Background())
	ca.IntrospectTable(context.Background(), "public", "users")
	if _, err := ca.Exec(context.Background(), "DROP TABLE users"); err != nil {
		t.Fatalf("exec: %v", err)
	}
	ca.IntrospectMeta(context.Background())
	ca.IntrospectTable(context.Background(), "public", "users")

	if f.metaCalls != 2 {
		t.Fatalf("meta introspected %d times after exec, want 2", f.metaCalls)
	}
	if f.tableCalls != 2 {
		t.Fatalf("table introspected %d times after exec, want 2", f.tableCalls)
	}
}

func TestStudioCachedWithoutCacheReturnsRaw(t *testing.T) {
	f, _ := newFake()
	s := &Studio{}
	a := s.cached("db-1", f)
	if a != f {
		t.Fatalf("expected raw adapter when cache disabled")
	}
}
