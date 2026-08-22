package adapter

import (
	"context"
	"errors"
	"strings"
)

var ErrUnsupportedEngine = errors.New("unsupported engine")

type Dialect interface {
	Engine() string
	Quote(ident string) string
	SelectTOP(limit int) string
	LimitSQL(limit, offset int, hasOrderBy bool) string
	Env(user, pass string) []string
	QueryArgs(user, pass, db, sql string, opts QueryOptions) []string
	ExecArgs(user, pass, db, sql string) []string
	Parser() ResultParser
	VersionSQL() string
	SchemaSQL() string
	TablesSQL(schema string) string
	ObjectsSQL(schema string) string
	ColumnsSQL(schema, table string) string
	IndexesSQL(schema, table string) string
	ConstraintsSQL(schema, table string) string
	ForeignKeysSQL(schema, table string) string
	TriggersSQL(schema, table string) string
}

type Adapter interface {
	Engine() string
	IntrospectMeta(ctx context.Context) (Meta, error)
	IntrospectSchemas(ctx context.Context) ([]string, error)
	IntrospectObjects(ctx context.Context, schema string) (ObjectSummary, error)
	ListObjects(ctx context.Context, schema string) ([]Object, error)
	IntrospectTable(ctx context.Context, schema, table string) (TableDetail, error)
	TableData(ctx context.Context, schema, table string, opts QueryOptions) (QueryResult, error)
	Query(ctx context.Context, sql string, opts QueryOptions) (QueryResult, error)
	Exec(ctx context.Context, sql string) (ExecResult, error)
}

var dialects = map[string]Dialect{}

func Register(d Dialect) {
	dialects[d.Engine()] = d
}

func New(engine string, ex *Executor) (Adapter, error) {
	switch strings.ToLower(engine) {
	case "mongodb":
		return &mongoAdapter{ex: ex}, nil
	case "redis":
		return &redisAdapter{ex: ex}, nil
	}
	d, ok := dialects[strings.ToLower(engine)]
	if !ok {
		return nil, ErrUnsupportedEngine
	}
	return &sqlAdapter{dialect: d, ex: ex}, nil
}

type sqlAdapter struct {
	dialect Dialect
	ex      *Executor
}

func (a *sqlAdapter) Engine() string { return a.dialect.Engine() }

func (a *sqlAdapter) IntrospectMeta(ctx context.Context) (Meta, error) {
	rows, err := a.ex.introspect(ctx, a.dialect, a.dialect.VersionSQL())
	if err != nil {
		return Meta{}, err
	}
	version := ""
	if len(rows) > 0 && len(rows[0]) > 0 {
		version = cellString(rows[0][0])
	}
	schemas, err := a.IntrospectSchemas(ctx)
	if err != nil {
		return Meta{}, err
	}
	meta := Meta{Engine: a.Engine(), Version: version, Status: "healthy", Schemas: len(schemas)}
	for _, s := range schemas {
		objs, err := a.IntrospectObjects(ctx, s)
		if err != nil {
			continue
		}
		meta.Tables += objs.Tables
		meta.Views += objs.Views
		meta.Functions += objs.Functions
	}
	return meta, nil
}

func (a *sqlAdapter) IntrospectSchemas(ctx context.Context) ([]string, error) {
	rows, err := a.ex.introspect(ctx, a.dialect, a.dialect.SchemaSQL())
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if len(r) > 0 {
			out = append(out, cellString(r[0]))
		}
	}
	return out, nil
}

func (a *sqlAdapter) IntrospectObjects(ctx context.Context, schema string) (ObjectSummary, error) {
	rows, err := a.ex.introspect(ctx, a.dialect, a.dialect.ObjectsSQL(schema))
	if err != nil || len(rows) == 0 {
		return ObjectSummary{}, err
	}
	r := rows[0]
	o := ObjectSummary{}
	if len(r) > 0 {
		o.Tables = toInt(r[0])
	}
	if len(r) > 1 {
		o.Views = toInt(r[1])
	}
	if len(r) > 2 {
		o.MatViews = toInt(r[2])
	}
	if len(r) > 3 {
		o.Functions = toInt(r[3])
	}
	if len(r) > 4 {
		o.Procedures = toInt(r[4])
	}
	if len(r) > 5 {
		o.Triggers = toInt(r[5])
	}
	if len(r) > 6 {
		o.Sequences = toInt(r[6])
	}
	if len(r) > 7 {
		o.Types = toInt(r[7])
	}
	if len(r) > 8 {
		o.Extensions = toInt(r[8])
	}
	return o, nil
}

func (a *sqlAdapter) ListObjects(ctx context.Context, schema string) ([]Object, error) {
	rows, err := a.ex.introspect(ctx, a.dialect, a.dialect.TablesSQL(schema))
	if err != nil {
		return nil, err
	}
	out := make([]Object, 0, len(rows))
	for _, r := range rows {
		if len(r) == 0 {
			continue
		}
		obj := Object{Name: cellString(r[0])}
		if len(r) > 1 {
			obj.Type = cellString(r[1])
		}
		out = append(out, obj)
	}
	return out, nil
}

func (a *sqlAdapter) IntrospectTable(ctx context.Context, schema, table string) (TableDetail, error) {
	detail := TableDetail{Schema: schema, Name: table, Type: "table"}
	cols, err := a.ex.introspect(ctx, a.dialect, a.dialect.ColumnsSQL(schema, table))
	if err != nil {
		return detail, err
	}
	detail.Columns = mapColumns(cols)
	idx, err := a.ex.introspect(ctx, a.dialect, a.dialect.IndexesSQL(schema, table))
	if err == nil {
		detail.Indexes = mapIndexes(idx)
	}
	con, err := a.ex.introspect(ctx, a.dialect, a.dialect.ConstraintsSQL(schema, table))
	if err == nil {
		detail.Constraints = mapConstraints(con)
	}
	fk, err := a.ex.introspect(ctx, a.dialect, a.dialect.ForeignKeysSQL(schema, table))
	if err == nil {
		detail.ForeignKeys = mapForeignKeys(fk)
	}
	tr, err := a.ex.introspect(ctx, a.dialect, a.dialect.TriggersSQL(schema, table))
	if err == nil {
		detail.Triggers = mapTriggers(tr)
	}
	return detail, nil
}

func (a *sqlAdapter) TableData(ctx context.Context, schema, table string, opts QueryOptions) (QueryResult, error) {
	if opts.MaxRows <= 0 {
		opts.MaxRows = a.ex.MaxRows
	}
	sql := buildTableQuery(a.dialect, schema, table, opts)
	return a.ex.query(ctx, a.dialect, sql, opts)
}

func (a *sqlAdapter) Query(ctx context.Context, sql string, opts QueryOptions) (QueryResult, error) {
	if opts.MaxRows <= 0 {
		opts.MaxRows = a.ex.MaxRows
	}
	if !isReadOnly(sql) {
		res, err := a.ex.exec(ctx, a.dialect, sql)
		if err != nil {
			return QueryResult{}, err
		}
		return QueryResult{RowCount: 0, DurationMs: res.DurationMs, ReadOnly: false, Message: res.Message}, nil
	}
	return a.ex.query(ctx, a.dialect, sql, opts)
}

func (a *sqlAdapter) Exec(ctx context.Context, sql string) (ExecResult, error) {
	return a.ex.exec(ctx, a.dialect, sql)
}
