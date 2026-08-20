package adapter

import (
	"context"
	"regexp"
	"strconv"
	"strings"
)

type redisAdapter struct {
	ex *Executor
}

func (r *redisAdapter) Engine() string { return "redis" }

func (r *redisAdapter) run(ctx context.Context, args ...string) (string, string, error) {
	full := append([]string{"redis-cli", "--no-auth-warning"}, args...)
	return r.ex.rawExec(ctx, full...)
}

func (r *redisAdapter) IntrospectMeta(ctx context.Context) (Meta, error) {
	out, _, _ := r.run(ctx, "INFO", "server")
	version := ""
	for _, l := range strings.Split(out, "\r\n") {
		if strings.HasPrefix(l, "redis_version:") {
			version = strings.TrimPrefix(l, "redis_version:")
		}
	}
	objects, err := r.IntrospectObjects(ctx, "default")
	if err != nil {
		return Meta{}, err
	}
	return Meta{Engine: "redis", Version: version, Status: "healthy", Schemas: 1, Tables: objects.Tables}, nil
}

func (r *redisAdapter) IntrospectSchemas(ctx context.Context) ([]string, error) {
	return []string{"default"}, nil
}

func (r *redisAdapter) IntrospectObjects(ctx context.Context, schema string) (ObjectSummary, error) {
	out, _, err := r.run(ctx, "DBSIZE")
	if err != nil {
		return ObjectSummary{}, err
	}
	n, _ := strconv.Atoi(strings.TrimSpace(out))
	return ObjectSummary{Tables: n}, nil
}

func (r *redisAdapter) ListObjects(ctx context.Context, schema string) ([]Object, error) {
	out, _, err := r.run(ctx, "SCAN", "0", "COUNT", "1000")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimRight(out, "\r\n"), "\n")
	if len(lines) < 2 {
		return nil, nil
	}
	keys := lines[1:]
	objs := make([]Object, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		typ, _, _ := r.run(ctx, "TYPE", k)
		objs = append(objs, Object{Name: k, Type: strings.TrimSpace(typ)})
	}
	return objs, nil
}

func (r *redisAdapter) IntrospectTable(ctx context.Context, schema, table string) (TableDetail, error) {
	detail := TableDetail{Schema: schema, Name: table, Type: "key"}
	detail.Columns = []Column{
		{Name: "key", Type: "string", Nullable: false},
		{Name: "type", Type: "string", Nullable: true},
		{Name: "ttl", Type: "integer", Nullable: true},
		{Name: "value", Type: "string", Nullable: true},
	}
	detail.ForeignKeys = []ForeignKey{{Name: "row", Columns: []string{"key", "type", "ttl", "value"}}}
	return detail, nil
}

func (r *redisAdapter) TableData(ctx context.Context, schema, table string, opts QueryOptions) (QueryResult, error) {
	typ, _, _ := r.run(ctx, "TYPE", table)
	ttl, _, _ := r.run(ctx, "TTL", table)
	ttl = strings.TrimSpace(ttl)
	if n, err := strconv.Atoi(ttl); err == nil && n < 0 {
		ttl = ""
	}
	val, _, _ := r.run(ctx, "GET", table)
	return QueryResult{
		Columns:  []string{"key", "type", "ttl", "value"},
		Rows:     [][]interface{}{{table, strings.TrimSpace(typ), strings.TrimSpace(ttl), strings.TrimSpace(val)}},
		RowCount: 1, ReadOnly: true,
	}, nil
}

func (r *redisAdapter) Query(ctx context.Context, sql string, opts QueryOptions) (QueryResult, error) {
	parts := splitCommand(sql)
	if len(parts) == 0 {
		return QueryResult{}, &QueryError{Message: "empty command"}
	}
	out, stderr, err := r.run(ctx, parts...)
	if err != nil {
		return QueryResult{}, &QueryError{Message: strings.TrimSpace(stderr)}
	}
	lines := strings.Split(strings.TrimRight(out, "\r\n"), "\n")
	rows := make([][]interface{}, 0, len(lines))
	for _, l := range lines {
		rows = append(rows, []interface{}{strings.TrimRight(l, "\r")})
	}
	if len(rows) == 0 {
		rows = [][]interface{}{{out}}
	}
	return QueryResult{Columns: []string{"result"}, Rows: rows, RowCount: len(rows), ReadOnly: true}, nil
}

func (r *redisAdapter) Exec(ctx context.Context, sql string) (ExecResult, error) {
	parts := splitCommand(sql)
	if len(parts) == 0 {
		return ExecResult{}, &QueryError{Message: "empty command"}
	}
	out, stderr, err := r.run(ctx, parts...)
	if err != nil {
		return ExecResult{}, &QueryError{Message: strings.TrimSpace(stderr)}
	}
	return ExecResult{Message: strings.TrimSpace(out)}, nil
}

var wordSplit = regexp.MustCompile(`"([^"]*)"|'([^']*)'|(\S+)`)

func splitCommand(s string) []string {
	var out []string
	for _, m := range wordSplit.FindAllStringSubmatch(s, -1) {
		switch {
		case m[1] != "":
			out = append(out, m[1])
		case m[2] != "":
			out = append(out, m[2])
		default:
			out = append(out, m[3])
		}
	}
	return out
}
