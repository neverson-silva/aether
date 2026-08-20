package adapter

import (
	"fmt"
	"strings"
)

func init() {
	Register(&oracleDialect{})
}

type oracleDialect struct{}

func (o *oracleDialect) Engine() string { return "oracle" }

func (o *oracleDialect) Quote(ident string) string {
	return `"` + strings.ReplaceAll(strings.ToUpper(ident), `"`, `""`) + `"`
}

func (o *oracleDialect) SelectTOP(limit int) string { return "" }

func (o *oracleDialect) LimitSQL(limit, offset int, hasOrderBy bool) string {
	if limit <= 0 {
		return ""
	}
	if offset <= 0 {
		return fmt.Sprintf(" FETCH FIRST %d ROWS ONLY", limit)
	}
	return fmt.Sprintf(" OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", offset, limit)
}

func (o *oracleDialect) Env(user, pass string) []string { return nil }

func (o *oracleDialect) Parser() ResultParser { return pipeParser{} }

func (o *oracleDialect) QueryArgs(user, pass, db, sql string, opts QueryOptions) []string {
	return []string{"sh", "-c", o.script(user, pass, db, sql)}
}

func (o *oracleDialect) ExecArgs(user, pass, db, sql string) []string {
	return []string{"sh", "-c", o.script(user, pass, db, sql)}
}

func (o *oracleDialect) script(user, pass, db, sql string) string {
	header := "SET PAGESIZE 10000\nSET HEADING ON\nSET COLSEP '|'\nSET FEEDBACK OFF\nSET LINESIZE 32767\nSET WRAP OFF\nSET TRIMSPOOL ON\nSET TRIMOUT ON\nSET VERIFY OFF\nSET SQLBLANKLINES ON\n"
	return fmt.Sprintf("sqlplus -S %s/%s@localhost:1521/%s <<'EOF'\n%s%s;\nEXIT\nEOF", user, pass, db, header, sql)
}

func (o *oracleDialect) VersionSQL() string {
	return "SELECT * FROM v$version WHERE ROWNUM=1"
}

func (o *oracleDialect) SchemaSQL() string {
	return `SELECT username FROM all_users WHERE oracle_maintained='N' ORDER BY username`
}

func (o *oracleDialect) TablesSQL(schema string) string {
	s := lit(schema)
	return fmt.Sprintf(`SELECT table_name, 'table' FROM all_tables WHERE UPPER(owner)=UPPER(%s)
UNION ALL
SELECT view_name, 'view' FROM all_views WHERE UPPER(owner)=UPPER(%s)
ORDER BY 1`, s, s)
}

func (o *oracleDialect) ObjectsSQL(schema string) string {
	s := lit(schema)
	return fmt.Sprintf(`SELECT
  (SELECT count(*) FROM all_tables WHERE UPPER(owner)=UPPER(%s)),
  (SELECT count(*) FROM all_views WHERE UPPER(owner)=UPPER(%s)),
  (SELECT count(*) FROM all_mviews WHERE UPPER(owner)=UPPER(%s)),
  (SELECT count(*) FROM all_objects WHERE UPPER(owner)=UPPER(%s) AND object_type='FUNCTION'),
  (SELECT count(*) FROM all_objects WHERE UPPER(owner)=UPPER(%s) AND object_type='PROCEDURE'),
  (SELECT count(*) FROM all_triggers WHERE UPPER(owner)=UPPER(%s)),
  (SELECT count(*) FROM all_sequences WHERE UPPER(sequence_owner)=UPPER(%s)),
  (SELECT count(*) FROM all_types WHERE UPPER(owner)=UPPER(%s)),
  0`, s, s, s, s, s, s, s, s)
}

func (o *oracleDialect) ColumnsSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT c.column_name,
  c.data_type || CASE WHEN c.data_type IN ('VARCHAR2','CHAR','NVARCHAR2','RAW') THEN '('||c.data_length||')' ELSE '' END,
  CASE WHEN c.nullable='Y' THEN 1 ELSE 0 END,
  '',
  CASE WHEN c.identity_column='YES' THEN 'identity' ELSE '' END, '',
  '',
  CASE WHEN EXISTS (SELECT 1 FROM all_constraints con JOIN all_cons_columns cc ON cc.owner=con.owner AND cc.constraint_name=con.constraint_name WHERE UPPER(con.owner)=UPPER(%s) AND UPPER(con.table_name)=UPPER(%s) AND con.constraint_type='P' AND cc.column_name=c.column_name) THEN 1 ELSE 0 END,
  CASE WHEN EXISTS (SELECT 1 FROM all_constraints con JOIN all_cons_columns cc ON cc.owner=con.owner AND cc.constraint_name=con.constraint_name WHERE UPPER(con.owner)=UPPER(%s) AND UPPER(con.table_name)=UPPER(%s) AND con.constraint_type='U' AND cc.column_name=c.column_name) THEN 1 ELSE 0 END
FROM all_tab_columns c
WHERE UPPER(c.owner)=UPPER(%s) AND UPPER(c.table_name)=UPPER(%s)
ORDER BY c.column_id`, s, t, s, t, s, t)
}

func (o *oracleDialect) IndexesSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT i.index_name, i.index_type,
  CASE WHEN i.uniqueness='UNIQUE' THEN 1 ELSE 0 END,
  CASE WHEN EXISTS (SELECT 1 FROM all_constraints con WHERE con.index_name=i.index_name AND con.constraint_type='P') THEN 1 ELSE 0 END,
  (SELECT LISTAGG(c.column_name, ', ') WITHIN GROUP (ORDER BY c.column_position) FROM all_ind_columns c WHERE c.index_owner=i.owner AND c.index_name=i.index_name AND c.table_name=i.table_name),
  ''
FROM all_indexes i
WHERE UPPER(i.owner)=UPPER(%s) AND UPPER(i.table_name)=UPPER(%s)`, s, t)
}

func (o *oracleDialect) ConstraintsSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT con.constraint_name,
  CASE con.constraint_type WHEN 'P' THEN 'PRIMARY KEY' WHEN 'U' THEN 'UNIQUE' WHEN 'C' THEN 'CHECK' END,
  (SELECT LISTAGG(cc.column_name, ', ') WITHIN GROUP (ORDER BY cc.position) FROM all_cons_columns cc WHERE cc.owner=con.owner AND cc.constraint_name=con.constraint_name),
  ''
FROM all_constraints con
WHERE UPPER(con.owner)=UPPER(%s) AND UPPER(con.table_name)=UPPER(%s) AND con.constraint_type IN ('P','U','C')`, s, t)
}

func (o *oracleDialect) ForeignKeysSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT con.constraint_name,
  (SELECT LISTAGG(cc.column_name, ', ') WITHIN GROUP (ORDER BY cc.position) FROM all_cons_columns cc WHERE cc.owner=con.owner AND cc.constraint_name=con.constraint_name),
  rtab.table_name,
  (SELECT LISTAGG(rcc.column_name, ', ') WITHIN GROUP (ORDER BY rcc.position) FROM all_cons_columns rcc WHERE rcc.owner=rcon.owner AND rcc.constraint_name=rcon.constraint_name),
  con.delete_rule,
  con.delete_rule
FROM all_constraints con
JOIN all_constraints rcon ON rcon.owner=con.r_owner AND rcon.constraint_name=con.r_constraint_name
JOIN all_tables rtab ON rtab.owner=rcon.owner AND rtab.table_name=rcon.table_name
WHERE UPPER(con.owner)=UPPER(%s) AND UPPER(con.table_name)=UPPER(%s) AND con.constraint_type='R'`, s, t)
}

func (o *oracleDialect) TriggersSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT trigger_name,
  SUBSTR(trigger_type, 1, INSTR(trigger_type, ' ')-1),
  'UPDATE',
  '',
  CASE WHEN status='ENABLED' THEN 'YES' ELSE 'NO' END
FROM all_triggers
WHERE UPPER(owner)=UPPER(%s) AND UPPER(table_name)=UPPER(%s)`, s, t)
}