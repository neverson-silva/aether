package adapter

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type Runner interface {
	Exec(ctx context.Context, containerID string, env []string, args ...string) (stdout string, stderr string, err error)
}

type Executor struct {
	Runner      Runner
	ContainerID string
	Engine      string
	User        string
	Pass        string
	DBName      string
	Timeout     time.Duration
	MaxRows     int
}

func (e *Executor) run(ctx context.Context, dialect Dialect, env, args []string) (string, string, error) {
	timeout := e.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return e.Runner.Exec(runCtx, e.ContainerID, env, args...)
}

func (e *Executor) rawExec(ctx context.Context, args ...string) (string, string, error) {
	timeout := e.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return e.Runner.Exec(runCtx, e.ContainerID, nil, args...)
}

func (e *Executor) query(ctx context.Context, dialect Dialect, sql string, opts QueryOptions) (QueryResult, error) {
	start := time.Now()
	args := dialect.QueryArgs(e.User, e.Pass, e.DBName, sql, opts)
	stdout, stderr, err := e.run(ctx, dialect, dialect.Env(e.User, e.Pass), args)
	res := QueryResult{DurationMs: time.Since(start).Milliseconds(), ReadOnly: isReadOnly(sql)}
	if err != nil {
		qerr := mapError(stderr)
		res.Error = qerr
		return res, qerr
	}
	rows, truncated := dialect.Parser().Parse(stdout, e.MaxRows, opts.Schema)
	res.Columns = rows.columns
	res.Rows = rows.rows
	res.RowCount = len(rows.rows)
	res.Truncated = truncated
	return res, nil
}

func (e *Executor) exec(ctx context.Context, dialect Dialect, sql string) (ExecResult, error) {
	start := time.Now()
	args := dialect.ExecArgs(e.User, e.Pass, e.DBName, sql)
	stdout, stderr, err := e.run(ctx, dialect, dialect.Env(e.User, e.Pass), args)
	res := ExecResult{DurationMs: time.Since(start).Milliseconds()}
	if err != nil {
		return res, mapError(stderr)
	}
	res.Message = strings.TrimSpace(stdout)
	return res, nil
}

func mapError(stderr string) *QueryError {
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	msg := ""
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" || t == "^" {
			continue
		}
		upper := strings.ToUpper(t)
		if strings.HasPrefix(upper, "ERROR:") || strings.HasPrefix(upper, "FATAL:") ||
			strings.HasPrefix(upper, "SQLSTATE") || strings.HasPrefix(upper, "DETAIL:") ||
			strings.HasPrefix(upper, "HINT:") {
			if msg == "" || strings.HasPrefix(upper, "ERROR:") || strings.HasPrefix(upper, "FATAL:") {
				msg = strings.TrimPrefix(strings.TrimPrefix(t, "psql:"), " ")
			}
			continue
		}
		if msg == "" {
			msg = t
		}
	}
	if msg == "" {
		msg = strings.TrimSpace(stderr)
	}
	return &QueryError{Message: msg}
}

func extractPostgresPosition(stderr string) int {
	for _, l := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(l, "LINE ") {
			return 0
		}
	}
	return 0
}

type parsedRows struct {
	columns []string
	rows    [][]interface{}
}

type ResultParser interface {
	Parse(stdout string, maxRows int, schema []Column) (parsedRows, bool)
}

func namesOf(schema []Column) []string {
	names := make([]string, len(schema))
	for i, c := range schema {
		names[i] = c.Name
	}
	return names
}

func alignCells(cells []string, schema []Column) []string {
	if len(schema) == 0 {
		return cells
	}
	if len(cells) > len(schema) {
		return cells[:len(schema)]
	}
	for len(cells) < len(schema) {
		cells = append(cells, "NULL")
	}
	return cells
}

func parseCell(s string) interface{} {
	if s == "NULL" {
		return nil
	}
	if s == "true" || s == "t" {
		return true
	}
	if s == "false" || s == "f" {
		return false
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}

type csvParser struct{}

func (csvParser) Parse(stdout string, maxRows int, schema []Column) (parsedRows, bool) {
	truncated := false
	out := parsedRows{}
	r := csv.NewReader(strings.NewReader(stdout))
	r.LazyQuotes = true
	header, err := r.Read()
	if err != nil {
		return out, false
	}
	if len(schema) > 0 {
		out.columns = namesOf(schema)
	} else {
		out.columns = header
	}
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		if maxRows > 0 && len(out.rows) >= maxRows {
			truncated = true
			break
		}
		rec = alignCells(rec, schema)
		row := make([]interface{}, len(rec))
		for i, cell := range rec {
			row[i] = parseCell(cell)
		}
		out.rows = append(out.rows, row)
	}
	return out, truncated
}

type tsvParser struct{}

func (tsvParser) Parse(stdout string, maxRows int, schema []Column) (parsedRows, bool) {
	truncated := false
	out := parsedRows{}
	lines := strings.Split(strings.TrimRight(stdout, "\r\n"), "\n")
	if len(lines) == 0 {
		return out, false
	}
	if len(schema) > 0 {
		out.columns = namesOf(schema)
	} else {
		out.columns = strings.Split(lines[0], "\t")
	}
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		if maxRows > 0 && len(out.rows) >= maxRows {
			truncated = true
			break
		}
		cells := alignCells(strings.Split(line, "\t"), schema)
		row := make([]interface{}, len(cells))
		for i, c := range cells {
			row[i] = parseCell(unescapeTSV(c))
		}
		out.rows = append(out.rows, row)
	}
	return out, truncated
}

func unescapeTSV(s string) string {
	r := strings.NewReplacer(`\\`, `\`, `\n`, "\n", `\t`, "\t", `\0`, "\x00", `\'`, "'", `\"`, `"`)
	return r.Replace(s)
}

type pipeParser struct{}

func (pipeParser) Parse(stdout string, maxRows int, schema []Column) (parsedRows, bool) {
	out := parsedRows{}
	lines := strings.Split(strings.TrimRight(stdout, "\r\n"), "\n")
	i := 0
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i >= len(lines) {
		return out, false
	}
	if len(schema) > 0 {
		out.columns = namesOf(schema)
	} else {
		out.columns = splitPipe(lines[i])
	}
	i++
	if i < len(lines) && strings.ContainsAny(strings.TrimSpace(lines[i]), "-_=") {
		i++
	}
	truncated := false
	for ; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if maxRows > 0 && len(out.rows) >= maxRows {
			truncated = true
			break
		}
		cells := alignCells(splitPipe(line), schema)
		row := make([]interface{}, len(cells))
		for j, c := range cells {
			row[j] = parseCell(strings.TrimSpace(c))
		}
		out.rows = append(out.rows, row)
	}
	return out, truncated
}

func splitPipe(s string) []string {
	return strings.Split(strings.TrimPrefix(s, "|"), "|")
}

func (e *Executor) introspect(ctx context.Context, dialect Dialect, sql string) ([][]interface{}, error) {
	res, err := e.query(ctx, dialect, sql, QueryOptions{})
	if err != nil {
		return nil, err
	}
	return res.Rows, nil
}

func cellString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}
