package adapter

import (
	"regexp"
	"strings"
)

type sqlClass int

const (
	classRead sqlClass = iota
	classWrite
	classDDL
	classDanger
)

var leadingKw = regexp.MustCompile(`(?i)^\s*(?:--[^\n]*\n|\s|/\*.*?\*/)*\s*([a-z]+)`)

func firstKeyword(sql string) string {
	m := leadingKw.FindStringSubmatch(sql)
	if len(m) < 2 {
		return ""
	}
	return strings.ToUpper(m[1])
}

func classify(sql string) sqlClass {
	kw := firstKeyword(sql)
	switch kw {
	case "SELECT", "EXPLAIN", "TABLE", "VALUES", "SHOW", "DESCRIBE", "WITH":
		return classRead
	case "DROP", "TRUNCATE", "DELETE":
		return classDanger
	case "CREATE", "ALTER":
		return classDDL
	default:
		return classWrite
	}
}

func isReadOnly(sql string) bool {
	if c := classify(sql); c == classRead {
		return true
	}
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(sql)), "WITH") {
		return strings.Contains(strings.ToUpper(sql), "SELECT")
	}
	return false
}

func quoteIdent(id string) string {
	return `"` + strings.ReplaceAll(id, `"`, `""`) + `"`
}

func buildTableQuery(d Dialect, schema, table string, opts QueryOptions) string {
	var b strings.Builder
	b.WriteString("SELECT ")
	b.WriteString(d.SelectTOP(opts.Limit))
	b.WriteString("* FROM ")
	if schema != "" {
		b.WriteString(d.Quote(schema))
		b.WriteString(".")
	}
	b.WriteString(d.Quote(table))
	for _, f := range opts.Filters {
		b.WriteString(" WHERE ")
		b.WriteString(d.Quote(f.Column))
		switch strings.ToUpper(f.Op) {
		case "LIKE", "ILIKE":
			b.WriteString(" " + strings.ToUpper(f.Op) + " ")
			b.WriteString("'" + escapeString(f.Value) + "'")
		case "IN":
			b.WriteString(" IN (" + f.Value + ")")
		default:
			b.WriteString(" " + f.Op + " ")
			b.WriteString("'" + escapeString(f.Value) + "'")
		}
	}
	hasOrderBy := opts.Sort != ""
	if hasOrderBy {
		order := strings.ToLower(opts.Order)
		if order != "desc" {
			order = "asc"
		}
		b.WriteString(" ORDER BY ")
		b.WriteString(d.Quote(opts.Sort))
		b.WriteString(" " + strings.ToUpper(order))
	}
	b.WriteString(d.LimitSQL(opts.Limit, opts.Offset, hasOrderBy))
	return b.String()
}

func escapeString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
