package adapter

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
)

type mongoAdapter struct {
	ex *Executor
}

func (m *mongoAdapter) Engine() string { return "mongodb" }

func (m *mongoAdapter) base() []string {
	return []string{"mongosh", "--quiet", "mongodb://localhost:27017/" + m.ex.DBName,
		"--username", m.ex.User, "--password", m.ex.Pass, "--authenticationDatabase", "admin"}
}

func (m *mongoAdapter) eval(ctx context.Context, js string) (string, error) {
	args := append(m.base(), "--eval", "print(JSON.stringify("+js+"))")
	stdout, stderr, err := m.ex.rawExec(ctx, args...)
	if err != nil {
		return "", &QueryError{Message: strings.TrimSpace(stderr)}
	}
	return stdout, nil
}

func (m *mongoAdapter) rawRun(ctx context.Context, js string) (string, string, error) {
	args := append(m.base(), "--eval", js)
	return m.ex.rawExec(ctx, args...)
}

func (m *mongoAdapter) IntrospectMeta(ctx context.Context) (Meta, error) {
	version, err := m.eval(ctx, "db.version()")
	if err != nil {
		return Meta{}, err
	}
	schemas, err := m.IntrospectSchemas(ctx)
	if err != nil {
		return Meta{}, err
	}
	meta := Meta{Engine: "mongodb", Version: strings.TrimSpace(version), Status: "healthy", Schemas: len(schemas)}
	for _, s := range schemas {
		objs, err := m.IntrospectObjects(ctx, s)
		if err == nil {
			meta.Tables += objs.Tables
		}
	}
	return meta, nil
}

func (m *mongoAdapter) IntrospectSchemas(ctx context.Context) ([]string, error) {
	return []string{m.ex.DBName}, nil
}

func (m *mongoAdapter) IntrospectObjects(ctx context.Context, schema string) (ObjectSummary, error) {
	out, err := m.eval(ctx, "db.getCollectionNames()")
	if err != nil {
		return ObjectSummary{}, err
	}
	var names []string
	if json.Unmarshal([]byte(out), &names) == nil {
		return ObjectSummary{Tables: len(names)}, nil
	}
	return ObjectSummary{}, nil
}

func (m *mongoAdapter) ListObjects(ctx context.Context, schema string) ([]Object, error) {
	out, err := m.eval(ctx, "db.getCollectionNames()")
	if err != nil {
		return nil, err
	}
	var names []string
	if err := json.Unmarshal([]byte(out), &names); err != nil {
		return nil, &QueryError{Message: err.Error()}
	}
	objs := make([]Object, 0, len(names))
	for _, n := range names {
		objs = append(objs, Object{Name: n, Type: "collection"})
	}
	return objs, nil
}

func (m *mongoAdapter) IntrospectTable(ctx context.Context, schema, table string) (TableDetail, error) {
	detail := TableDetail{Schema: schema, Name: table, Type: "collection"}
	sample, err := m.eval(ctx, "db"+ident(table)+".findOne()")
	if err == nil {
		var doc map[string]interface{}
		if json.Unmarshal([]byte(sample), &doc) == nil {
			i := 0
			for k := range doc {
				detail.Columns = append(detail.Columns, Column{Name: k, Type: "field", Nullable: true})
				i++
				if i > 50 {
					break
				}
			}
		}
	}
	idx, err := m.eval(ctx, "db"+ident(table)+".getIndexes()")
	if err == nil {
		var arr []map[string]interface{}
		if json.Unmarshal([]byte(idx), &arr) == nil {
			for _, ix := range arr {
				detail.Indexes = append(detail.Indexes, Index{
					Name: strOf(ix["name"]), Method: "btree",
					Unique: boolOf(ix["unique"]),
				})
			}
		}
	}
	return detail, nil
}

func (m *mongoAdapter) TableData(ctx context.Context, schema, table string, opts QueryOptions) (QueryResult, error) {
	js := "db" + ident(table) + ".find()"
	if opts.Sort != "" {
		js += ".sort({" + opts.Sort + ":" + (map[bool]string{true: "1", false: "-1"})[opts.Order == "desc"] + "})"
	}
	if opts.Limit > 0 {
		js += ".limit(" + itoa(opts.Limit) + ")"
	}
	if opts.Offset > 0 {
		js += ".skip(" + itoa(opts.Offset) + ")"
	}
	js += ".toArray()"
	out, err := m.eval(ctx, js)
	if err != nil {
		return QueryResult{}, err
	}
	var docs []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &docs); err != nil {
		return QueryResult{}, &QueryError{Message: err.Error()}
	}
	return mongoRows(docs), nil
}

func (m *mongoAdapter) Query(ctx context.Context, sql string, opts QueryOptions) (QueryResult, error) {
	out, err := m.eval(ctx, sql)
	if err != nil {
		return QueryResult{}, err
	}
	var docs []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &docs); err == nil {
		return mongoRows(docs), nil
	}
	var raw interface{}
	if json.Unmarshal([]byte(out), &raw) == nil {
		return QueryResult{Columns: []string{"result"}, Rows: [][]interface{}{{raw}}, RowCount: 1, ReadOnly: true}, nil
	}
	return QueryResult{}, &QueryError{Message: "unparseable result"}
}

func (m *mongoAdapter) Exec(ctx context.Context, sql string) (ExecResult, error) {
	_, stderr, err := m.rawRun(ctx, sql)
	if err != nil {
		return ExecResult{}, &QueryError{Message: strings.TrimSpace(stderr)}
	}
	return ExecResult{Message: "OK"}, nil
}

func mongoRows(docs []map[string]interface{}) QueryResult {
	cols := []string{}
	seen := map[string]bool{}
	for _, d := range docs {
		for k := range d {
			if !seen[k] {
				seen[k] = true
				cols = append(cols, k)
			}
		}
	}
	rows := make([][]interface{}, 0, len(docs))
	for _, d := range docs {
		row := make([]interface{}, len(cols))
		for i, c := range cols {
			row[i] = cellFor(c, d[c])
		}
		rows = append(rows, row)
	}
	return QueryResult{Columns: cols, Rows: rows, RowCount: len(rows), ReadOnly: true}
}

func cellFor(key string, v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}, []interface{}:
		b, _ := json.Marshal(t)
		return string(b)
	default:
		return v
	}
}

func ident(name string) string {
	return "[" + strconv.Quote(name) + "]"
}

func strOf(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func boolOf(v interface{}) bool {
	b, _ := v.(bool)
	return b
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
