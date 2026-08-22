package adapter

import (
	"fmt"
	"strings"
)

func init() {
	Register(&postgresDialect{})
}

type postgresDialect struct{}

func (p *postgresDialect) Engine() string { return "postgres" }

func (p *postgresDialect) Quote(ident string) string { return quoteIdent(ident) }

func (p *postgresDialect) SelectTOP(limit int) string { return "" }

func (p *postgresDialect) LimitSQL(limit, offset int, hasOrderBy bool) string {
	var b strings.Builder
	if limit > 0 {
		b.WriteString(fmt.Sprintf(" LIMIT %d", limit))
	}
	if offset > 0 {
		b.WriteString(fmt.Sprintf(" OFFSET %d", offset))
	}
	return b.String()
}

func (p *postgresDialect) Env(user, pass string) []string {
	return []string{"PGPASSWORD=" + pass}
}

func (p *postgresDialect) Parser() ResultParser { return csvParser{} }

func (p *postgresDialect) QueryArgs(user, pass, db, sql string, opts QueryOptions) []string {
	s := sql
	if classify(sql) == classRead {
		s = wrapCopy(sql)
	}
	return []string{"psql", "-U", user, "-d", db, "-X", "--no-psqlrc", "--csv", "-c", s}
}

func (p *postgresDialect) ExecArgs(user, pass, db, sql string) []string {
	return []string{"psql", "-U", user, "-d", db, "-X", "--no-psqlrc", "-c", sql}
}

func wrapCopy(sql string) string {
	s := strings.TrimRight(strings.TrimSpace(sql), ";")
	return fmt.Sprintf("COPY (%s) TO STDOUT WITH (FORMAT csv, HEADER true, NULL 'NULL', FORCE_QUOTE *)", s)
}

func lit(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func (p *postgresDialect) VersionSQL() string {
	return "SELECT current_setting('server_version')"
}

func (p *postgresDialect) SchemaSQL() string {
	return `SELECT nspname FROM pg_catalog.pg_namespace
	WHERE nspname NOT IN ('pg_catalog','information_schema','pg_toast')
	  AND nspname NOT LIKE 'pg_%'
	ORDER BY nspname`
}

func (p *postgresDialect) TablesSQL(schema string) string {
	s := lit(schema)
	return fmt.Sprintf(`SELECT c.relname, CASE c.relkind WHEN 'r' THEN 'table' WHEN 'p' THEN 'table' WHEN 'v' THEN 'view' WHEN 'm' THEN 'materialized view' END
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
WHERE n.nspname=%s AND c.relkind IN ('r','p','v','m')
ORDER BY c.relname`, s)
}

func (p *postgresDialect) ObjectsSQL(schema string) string {
	s := lit(schema)
	return fmt.Sprintf(`SELECT
  (SELECT count(*) FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=%s AND c.relkind='r'),
  (SELECT count(*) FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=%s AND c.relkind='v'),
  (SELECT count(*) FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=%s AND c.relkind='m'),
  (SELECT count(*) FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname=%s AND p.prokind='f'),
  (SELECT count(*) FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname=%s AND p.prokind='p'),
  (SELECT count(*) FROM pg_catalog.pg_trigger t JOIN pg_catalog.pg_class c ON c.oid=t.tgrelid JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=%s AND NOT t.tgisinternal),
  (SELECT count(*) FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=%s AND c.relkind='S'),
  (SELECT count(*) FROM pg_catalog.pg_type t JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace WHERE n.nspname=%s AND t.typtype IN ('e','d','c') AND t.typrelid=0),
  (SELECT count(*) FROM pg_catalog.pg_extension e)`, s, s, s, s, s, s, s, s)
}

func (p *postgresDialect) ColumnsSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT
  a.attname,
  format_type(a.atttypid, a.atttypmod),
  NOT a.attnotnull,
  COALESCE(pg_get_expr(d.adbin, d.adrelid), ''),
  a.attidentity,
  a.attgenerated,
  COALESCE(col_description(a.attrelid, a.attnum), ''),
  EXISTS (SELECT 1 FROM pg_catalog.pg_index i WHERE i.indrelid=a.attrelid AND i.indisprimary AND a.attnum=ANY(i.indkey)),
  EXISTS (SELECT 1 FROM pg_catalog.pg_constraint con WHERE con.conrelid=a.attrelid AND con.contype='u' AND a.attnum=ANY(con.conkey))
FROM pg_catalog.pg_attribute a
JOIN pg_catalog.pg_class c ON c.oid=a.attrelid
JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum
WHERE n.nspname=%s AND c.relname=%s AND a.attnum>0 AND NOT a.attisdropped
ORDER BY a.attnum`, s, t)
}

func (p *postgresDialect) IndexesSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT
  i.relname,
  am.amname,
  ix.indisunique,
  ix.indisprimary,
  (SELECT string_agg(pg_get_indexdef(ix.indexrelid, k+1, true), ', ' ORDER BY k)
     FROM generate_series(0, ix.indnkeys-1) k),
  COALESCE(pg_get_expr(ix.indpred, ix.indrelid), '')
FROM pg_catalog.pg_index ix
JOIN pg_catalog.pg_class i ON i.oid=ix.indexrelid
JOIN pg_catalog.pg_class t ON t.oid=ix.indrelid
JOIN pg_catalog.pg_namespace n ON n.oid=t.relnamespace
JOIN pg_catalog.pg_am am ON am.oid=i.relam
WHERE n.nspname=%s AND t.relname=%s
ORDER BY ix.indisprimary DESC, i.relname`, s, t)
}

func (p *postgresDialect) ConstraintsSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT
  con.conname,
  con.contype,
  COALESCE((SELECT string_agg(a.attname, ',') FROM unnest(con.conkey) WITH ORDINALITY k(attnum, ord) JOIN pg_catalog.pg_attribute a ON a.attrelid=con.conrelid AND a.attnum=k.attnum), ''),
  COALESCE(pg_get_constraintdef(con.oid), '')
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class c ON c.oid=con.conrelid
JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
WHERE n.nspname=%s AND c.relname=%s AND con.contype IN ('p','u','c')
ORDER BY con.conname`, s, t)
}

func (p *postgresDialect) ForeignKeysSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT
  con.conname,
  (SELECT string_agg(a.attname, ',') FROM unnest(con.conkey) WITH ORDINALITY k(attnum, ord) JOIN pg_catalog.pg_attribute a ON a.attrelid=con.conrelid AND a.attnum=k.attnum),
  cl.relname,
  (SELECT string_agg(a.attname, ',') FROM unnest(con.confkey) WITH ORDINALITY k(attnum, ord) JOIN pg_catalog.pg_attribute a ON a.attrelid=con.confrelid AND a.attnum=k.attnum),
  con.confdeltype,
  con.confupdtype
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class c ON c.oid=con.conrelid
JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
JOIN pg_catalog.pg_class cl ON cl.oid=con.confrelid
WHERE n.nspname=%s AND c.relname=%s AND con.contype='f'
ORDER BY con.conname`, s, t)
}

func (p *postgresDialect) TriggersSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT
  t.tgname,
  CASE t.tgtype & 66 WHEN 2 THEN 'BEFORE' WHEN 64 THEN 'INSTEAD OF' ELSE 'AFTER' END,
  (CASE WHEN t.tgtype & 4 <> 0 THEN 'INSERT' ELSE '' END ||
   CASE WHEN t.tgtype & 8 <> 0 THEN ' UPDATE' ELSE '' END ||
   CASE WHEN t.tgtype & 16 <> 0 THEN ' DELETE' ELSE '' END),
  p.proname,
  CASE t.tgenabled WHEN 'O' THEN 'YES' ELSE 'NO' END
FROM pg_catalog.pg_trigger t
JOIN pg_catalog.pg_class c ON c.oid=t.tgrelid
JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
JOIN pg_catalog.pg_proc p ON p.oid=t.tgfoid
WHERE n.nspname=%s AND c.relname=%s AND NOT t.tgisinternal
ORDER BY t.tgname`, s, t)
}
