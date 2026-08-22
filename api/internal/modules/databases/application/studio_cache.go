package application

import (
	"context"
	"encoding/json"
	"time"

	"aether/internal/platform/database/adapter"
	"aether/internal/platform/druntime/cache"
)

type catalog struct {
	Engine  string                           `json:"engine"`
	Version string                           `json:"version"`
	Schemas []string                         `json:"schemas"`
	Objects map[string][]adapter.Object      `json:"objects"`
	Counts  map[string]adapter.ObjectSummary `json:"counts"`
}

type cachedAdapter struct {
	adapter.Adapter
	cache cache.Cache
	dbID  string
	ttl   time.Duration
}

func (s *Studio) cached(id string, a adapter.Adapter) adapter.Adapter {
	if s.Cache == nil || s.CatalogTTL <= 0 {
		return a
	}
	return &cachedAdapter{Adapter: a, cache: s.Cache, dbID: id, ttl: s.CatalogTTL}
}

func (c *cachedAdapter) prefix() string {
	return "studio:db:" + c.dbID
}

func (c *cachedAdapter) getCatalog(ctx context.Context) (*catalog, error) {
	key := c.prefix() + ":cat"
	if b, ok, err := c.cache.Get(ctx, key); err == nil && ok {
		var cat catalog
		if json.Unmarshal(b, &cat) == nil {
			return &cat, nil
		}
	}
	cat, err := c.buildCatalog(ctx)
	if err != nil {
		return nil, err
	}
	if b, err := json.Marshal(cat); err == nil {
		_ = c.cache.Set(ctx, key, b, c.ttl)
	}
	return cat, nil
}

func (c *cachedAdapter) buildCatalog(ctx context.Context) (*catalog, error) {
	meta, err := c.Adapter.IntrospectMeta(ctx)
	if err != nil {
		return nil, err
	}
	schemas, err := c.Adapter.IntrospectSchemas(ctx)
	if err != nil {
		return nil, err
	}
	cat := &catalog{
		Engine:  c.Adapter.Engine(),
		Version: meta.Version,
		Schemas: schemas,
		Objects: map[string][]adapter.Object{},
		Counts:  map[string]adapter.ObjectSummary{},
	}
	for _, s := range schemas {
		if objs, err := c.Adapter.ListObjects(ctx, s); err == nil {
			cat.Objects[s] = objs
		}
		if counts, err := c.Adapter.IntrospectObjects(ctx, s); err == nil {
			cat.Counts[s] = counts
		}
	}
	return cat, nil
}

func (c *cachedAdapter) invalidate(ctx context.Context) {
	_ = c.cache.Invalidate(ctx, c.prefix()+":")
}

func (c *cachedAdapter) IntrospectMeta(ctx context.Context) (adapter.Meta, error) {
	cat, err := c.getCatalog(ctx)
	if err != nil {
		return adapter.Meta{}, err
	}
	meta := adapter.Meta{Engine: cat.Engine, Version: cat.Version, Status: "healthy", Schemas: len(cat.Schemas)}
	for _, counts := range cat.Counts {
		meta.Tables += counts.Tables
		meta.Views += counts.Views
		meta.Functions += counts.Functions
	}
	return meta, nil
}

func (c *cachedAdapter) IntrospectSchemas(ctx context.Context) ([]string, error) {
	cat, err := c.getCatalog(ctx)
	if err != nil {
		return nil, err
	}
	return cat.Schemas, nil
}

func (c *cachedAdapter) IntrospectObjects(ctx context.Context, schema string) (adapter.ObjectSummary, error) {
	cat, err := c.getCatalog(ctx)
	if err != nil {
		return adapter.ObjectSummary{}, err
	}
	return cat.Counts[schema], nil
}

func (c *cachedAdapter) ListObjects(ctx context.Context, schema string) ([]adapter.Object, error) {
	cat, err := c.getCatalog(ctx)
	if err != nil {
		return nil, err
	}
	return cat.Objects[schema], nil
}

func (c *cachedAdapter) IntrospectTable(ctx context.Context, schema, table string) (adapter.TableDetail, error) {
	key := c.prefix() + ":tbl:" + schema + "." + table
	if b, ok, err := c.cache.Get(ctx, key); err == nil && ok {
		var d adapter.TableDetail
		if json.Unmarshal(b, &d) == nil {
			return d, nil
		}
	}
	d, err := c.Adapter.IntrospectTable(ctx, schema, table)
	if err != nil {
		return d, err
	}
	if b, err := json.Marshal(d); err == nil {
		_ = c.cache.Set(ctx, key, b, c.ttl)
	}
	return d, nil
}

func (c *cachedAdapter) Exec(ctx context.Context, sql string) (adapter.ExecResult, error) {
	res, err := c.Adapter.Exec(ctx, sql)
	if err == nil {
		c.invalidate(ctx)
	}
	return res, err
}

func (c *cachedAdapter) TableData(ctx context.Context, schema, table string, opts adapter.QueryOptions) (adapter.QueryResult, error) {
	if len(opts.Schema) == 0 {
		if d, err := c.IntrospectTable(ctx, schema, table); err == nil {
			opts.Schema = d.Columns
		}
	}
	return c.Adapter.TableData(ctx, schema, table, opts)
}

func (c *cachedAdapter) Query(ctx context.Context, sql string, opts adapter.QueryOptions) (adapter.QueryResult, error) {
	res, err := c.Adapter.Query(ctx, sql, opts)
	if err == nil && !res.ReadOnly {
		c.invalidate(ctx)
	}
	return res, err
}
