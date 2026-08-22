package adapter

import (
	"fmt"
	"strings"
)

func init() {
	Register(&mysqlDialect{engine: "mysql", cli: "mysql"})
	Register(&mysqlDialect{engine: "mariadb", cli: "mariadb"})
}

type mysqlDialect struct {
	engine string
	cli    string
}

func (m *mysqlDialect) Engine() string { return m.engine }

func (m *mysqlDialect) Quote(ident string) string {
	return "`" + strings.ReplaceAll(ident, "`", "``") + "`"
}

func (m *mysqlDialect) SelectTOP(limit int) string { return "" }

func (m *mysqlDialect) LimitSQL(limit, offset int, hasOrderBy bool) string {
	var b strings.Builder
	if limit > 0 {
		b.WriteString(fmt.Sprintf(" LIMIT %d", limit))
	}
	if offset > 0 {
		b.WriteString(fmt.Sprintf(" OFFSET %d", offset))
	}
	return b.String()
}

func (m *mysqlDialect) Env(user, pass string) []string {
	return []string{"MYSQL_PWD=" + pass}
}

func (m *mysqlDialect) Parser() ResultParser { return tsvParser{} }

func (m *mysqlDialect) QueryArgs(user, pass, db, sql string, opts QueryOptions) []string {
	return []string{m.cli, "--user=" + user, "--database=" + db, "--batch", "-e", sql}
}

func (m *mysqlDialect) ExecArgs(user, pass, db, sql string) []string {
	return []string{m.cli, "--user=" + user, "--database=" + db, "-e", sql}
}

func (m *mysqlDialect) VersionSQL() string { return "SELECT VERSION()" }

func (m *mysqlDialect) SchemaSQL() string {
	return `SELECT schema_name FROM information_schema.schemata
	WHERE schema_name NOT IN ('information_schema','mysql','performance_schema','sys')
	ORDER BY schema_name`
}

func (m *mysqlDialect) TablesSQL(schema string) string {
	s := lit(schema)
	return fmt.Sprintf(`SELECT table_name, CASE WHEN table_type='BASE TABLE' THEN 'table' ELSE 'view' END
FROM information_schema.tables
WHERE table_schema=%s
ORDER BY table_name`, s)
}

func (m *mysqlDialect) ObjectsSQL(schema string) string {
	s := lit(schema)
	return fmt.Sprintf(`SELECT
  (SELECT count(*) FROM information_schema.tables WHERE table_schema=%s AND table_type='BASE TABLE'),
  (SELECT count(*) FROM information_schema.tables WHERE table_schema=%s AND table_type='VIEW'),
  0,
  (SELECT count(*) FROM information_schema.routines WHERE routine_schema=%s AND routine_type='FUNCTION'),
  (SELECT count(*) FROM information_schema.routines WHERE routine_schema=%s AND routine_type='PROCEDURE'),
  (SELECT count(*) FROM information_schema.triggers WHERE trigger_schema=%s),
  0, 0, 0`, s, s, s, s, s)
}

func (m *mysqlDialect) ColumnsSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT column_name, column_type,
  IF(is_nullable='YES',1,0), column_default,
  IF(extra LIKE '%%auto_increment%%','auto_increment',''), '',
  column_comment,
  IF(column_key='PRI',1,0), IF(column_key='UNI',1,0)
FROM information_schema.columns
WHERE table_schema=%s AND table_name=%s
ORDER BY ordinal_position`, s, t)
}

func (m *mysqlDialect) IndexesSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT INDEX_NAME, INDEX_TYPE,
  IF(NON_UNIQUE=0,1,0), IF(INDEX_NAME='PRIMARY',1,0),
  GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX SEPARATOR ', '), ''
FROM information_schema.statistics
WHERE table_schema=%s AND table_name=%s
GROUP BY INDEX_NAME, INDEX_TYPE, NON_UNIQUE, INDEX_NAME='PRIMARY'`, s, t)
}

func (m *mysqlDialect) ConstraintsSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT tc.CONSTRAINT_NAME, tc.CONSTRAINT_TYPE,
  COALESCE(GROUP_CONCAT(kcu.COLUMN_NAME ORDER BY kcu.ORDINAL_POSITION SEPARATOR ', '),''), ''
FROM information_schema.table_constraints tc
LEFT JOIN information_schema.key_column_usage kcu
  ON kcu.CONSTRAINT_NAME=tc.CONSTRAINT_NAME AND kcu.TABLE_SCHEMA=tc.TABLE_SCHEMA AND kcu.TABLE_NAME=tc.TABLE_NAME
WHERE tc.TABLE_SCHEMA=%s AND tc.TABLE_NAME=%s AND tc.CONSTRAINT_TYPE IN ('PRIMARY KEY','UNIQUE','CHECK')
GROUP BY tc.CONSTRAINT_NAME, tc.CONSTRAINT_TYPE`, s, t)
}

func (m *mysqlDialect) ForeignKeysSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT kcu.CONSTRAINT_NAME,
  GROUP_CONCAT(kcu.COLUMN_NAME ORDER BY kcu.ORDINAL_POSITION SEPARATOR ', '),
  kcu.REFERENCED_TABLE_NAME,
  GROUP_CONCAT(kcu.REFERENCED_COLUMN_NAME ORDER BY kcu.POSITION_IN_UNIQUE_CONSTRAINT SEPARATOR ', '),
  rc.DELETE_RULE, rc.UPDATE_RULE
FROM information_schema.key_column_usage kcu
JOIN information_schema.referential_constraints rc
  ON rc.CONSTRAINT_SCHEMA=kcu.TABLE_SCHEMA AND rc.CONSTRAINT_NAME=kcu.CONSTRAINT_NAME AND rc.TABLE_NAME=kcu.TABLE_NAME
WHERE kcu.TABLE_SCHEMA=%s AND kcu.TABLE_NAME=%s AND kcu.REFERENCED_TABLE_NAME IS NOT NULL
GROUP BY kcu.CONSTRAINT_NAME, kcu.REFERENCED_TABLE_NAME, rc.DELETE_RULE, rc.UPDATE_RULE`, s, t)
}

func (m *mysqlDialect) TriggersSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT TRIGGER_NAME, ACTION_TIMING, EVENT_MANIPULATION, ACTION_STATEMENT, 'YES'
FROM information_schema.triggers
WHERE TRIGGER_SCHEMA=%s AND EVENT_OBJECT_TABLE=%s`, s, t)
}
