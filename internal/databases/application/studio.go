package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"aether/internal/database/adapter"
	"aether/internal/databases/domain"
	"aether/internal/druntime/cache"
)

type Studio struct {
	Databases  *Databases
	Timeout    time.Duration
	MaxRows    int
	Cache      cache.Cache
	CatalogTTL time.Duration
}

func (s *Studio) adapterFor(ctx context.Context, id, orgID uuid.UUID) (adapter.Adapter, error) {
	db, err := s.Databases.Get(ctx, id, orgID)
	if err != nil {
		return nil, err
	}
	if s.Databases.Runtime == nil || db.ContainerID == "" {
		return nil, domain.ErrNotFound
	}
	pass, err := s.Databases.Passwords.Decrypt(db.PassEnc)
	if err != nil {
		return nil, domain.ErrValidation
	}
	ex := &adapter.Executor{
		Runner:      s.Databases.Runtime,
		ContainerID: db.ContainerID,
		Engine:      string(db.Engine),
		User:        db.User,
		Pass:        pass,
		DBName:      db.DBName,
		Timeout:     s.Timeout,
		MaxRows:     s.MaxRows,
	}
	a, err := adapter.New(string(db.Engine), ex)
	if err != nil {
		return nil, err
	}
	return s.cached(id.String(), a), nil
}

func (s *Studio) Engine(ctx context.Context, id, orgID uuid.UUID) (string, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return "", err
	}
	return a.Engine(), nil
}

func (s *Studio) IntrospectMeta(ctx context.Context, id, orgID uuid.UUID) (adapter.Meta, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return adapter.Meta{}, err
	}
	return a.IntrospectMeta(ctx)
}

func (s *Studio) IntrospectSchemas(ctx context.Context, id, orgID uuid.UUID) ([]string, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return nil, err
	}
	return a.IntrospectSchemas(ctx)
}

func (s *Studio) IntrospectObjects(ctx context.Context, id, orgID uuid.UUID, schema string) (adapter.ObjectSummary, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return adapter.ObjectSummary{}, err
	}
	return a.IntrospectObjects(ctx, schema)
}

func (s *Studio) ListObjects(ctx context.Context, id, orgID uuid.UUID, schema string) ([]adapter.Object, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return nil, err
	}
	return a.ListObjects(ctx, schema)
}

func (s *Studio) IntrospectTable(ctx context.Context, id, orgID uuid.UUID, schema, table string) (adapter.TableDetail, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return adapter.TableDetail{}, err
	}
	return a.IntrospectTable(ctx, schema, table)
}

func (s *Studio) TableData(ctx context.Context, id, orgID uuid.UUID, schema, table string, opts adapter.QueryOptions) (adapter.QueryResult, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return adapter.QueryResult{}, err
	}
	return a.TableData(ctx, schema, table, opts)
}

func (s *Studio) Query(ctx context.Context, id, orgID uuid.UUID, sql string, opts adapter.QueryOptions) (adapter.QueryResult, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return adapter.QueryResult{}, err
	}
	return a.Query(ctx, sql, opts)
}

func (s *Studio) Exec(ctx context.Context, id, orgID uuid.UUID, sql string) (adapter.ExecResult, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return adapter.ExecResult{}, err
	}
	return a.Exec(ctx, sql)
}

func (s *Studio) Refresh(ctx context.Context, id, orgID uuid.UUID) error {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return err
	}
	ca, ok := a.(*cachedAdapter)
	if !ok {
		return nil
	}
	ca.invalidate(ctx)
	_, err = ca.getCatalog(ctx)
	return err
}