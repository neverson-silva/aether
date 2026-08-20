package adapter

import (
	"fmt"
	"strings"
)

func init() {
	Register(&mssqlDialect{})
}

type mssqlDialect struct{}

func (m *mssqlDialect) Engine() string { return "mssql" }

func (m *mssqlDialect) Quote(ident string) string {
	return "[" + strings.ReplaceAll(ident, "]", "]]") + "]"
}

func (m *mssqlDialect) SelectTOP(limit int) string {
	if limit > 0 {
		return fmt.Sprintf("TOP %d ", limit)
	}
	return ""
}

func (m *mssqlDialect) LimitSQL(limit, offset int, hasOrderBy bool) string {
	if offset <= 0 {
		return ""
	}
	if limit <= 0 {
		limit = 1000
	}
	if !hasOrderBy {
		return fmt.Sprintf(" ORDER BY (SELECT NULL) OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", offset, limit)
	}
	return fmt.Sprintf(" OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", offset, limit)
}

func (m *mssqlDialect) Env(user, pass string) []string {
	return []string{"SQLCMDPASSWORD=" + pass}
}

func (m *mssqlDialect) Parser() ResultParser { return pipeParser{} }

func (m *mssqlDialect) QueryArgs(user, pass, db, sql string, opts QueryOptions) []string {
	return []string{"sh", "-c", mql(user, db, sql, "-W -s \"|\"")}
}

func (m *mssqlDialect) ExecArgs(user, pass, db, sql string) []string {
	return []string{"sh", "-c", mql(user, db, sql, "-h -1")}
}

func mql(user, db, sql, extra string) string {
	resolve := "SC=$(command -v sqlcmd 2>/dev/null || ls /opt/mssql-tools18/bin/sqlcmd /opt/mssql-tools/bin/sqlcmd 2>/dev/null | head -1)"
	return fmt.Sprintf("%s; \"$SC\" -C -S localhost -U %s -d %s %s -Q \"SET NOCOUNT ON;\n%s\"", resolve, user, db, extra, strings.ReplaceAll(sql, `"`, `\"`))
}

func (m *mssqlDialect) VersionSQL() string {
	return "SELECT CAST(SERVERPROPERTY('ProductVersion') AS varchar)"
}

func (m *mssqlDialect) SchemaSQL() string {
	return `SELECT name FROM sys.schemas
	WHERE name NOT IN ('guest','sys','INFORMATION_SCHEMA','db_accessadmin','db_backupoperator','db_datareader','db_datawriter','db_ddladmin','db_denydatareader','db_denydatawriter','db_owner','db_securityadmin')
	ORDER BY name`
}

func (m *mssqlDialect) TablesSQL(schema string) string {
	s := lit(schema)
	return fmt.Sprintf(`SELECT t.name, 'table' FROM sys.tables t JOIN sys.schemas s ON s.schema_id=t.schema_id WHERE s.name=%s
UNION ALL
SELECT v.name, 'view' FROM sys.views v JOIN sys.schemas s ON s.schema_id=v.schema_id WHERE s.name=%s
ORDER BY 1`, s, s)
}

func (m *mssqlDialect) ObjectsSQL(schema string) string {
	s := lit(schema)
	return fmt.Sprintf(`SELECT
  (SELECT count(*) FROM sys.tables t JOIN sys.schemas s ON s.schema_id=t.schema_id WHERE s.name=%s),
  (SELECT count(*) FROM sys.views v JOIN sys.schemas s ON s.schema_id=v.schema_id WHERE s.name=%s),
  0,
  (SELECT count(*) FROM sys.objects o JOIN sys.schemas s ON s.schema_id=o.schema_id WHERE s.name=%s AND o.type='FN'),
  (SELECT count(*) FROM sys.objects o JOIN sys.schemas s ON s.schema_id=o.schema_id WHERE s.name=%s AND o.type='P'),
  (SELECT count(*) FROM sys.triggers tr JOIN sys.objects o ON o.object_id=tr.parent_id JOIN sys.schemas s ON s.schema_id=o.schema_id WHERE s.name=%s),
  0, 0, 0`, s, s, s, s, s)
}

func (m *mssqlDialect) ColumnsSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT c.name,
  ty.name + CASE WHEN ty.name IN ('varchar','nvarchar','char','nchar','varbinary','binary')
    THEN '(' + CASE WHEN c.max_length=-1 THEN 'max' ELSE CAST(c.max_length AS varchar) END + ')' ELSE '' END,
  CASE WHEN c.is_nullable=1 THEN 1 ELSE 0 END,
  ISNULL(dc.definition,''),
  CASE WHEN c.is_identity=1 THEN 'identity' ELSE '' END,
  CASE WHEN c.is_computed=1 THEN 'computed' ELSE '' END,
  ISNULL(ep.value,''),
  CASE WHEN EXISTS (SELECT 1 FROM sys.index_columns ic JOIN sys.indexes i ON i.object_id=ic.object_id AND i.index_id=ic.index_id WHERE ic.object_id=c.object_id AND ic.column_id=c.column_id AND i.is_primary_key=1) THEN 1 ELSE 0 END,
  CASE WHEN EXISTS (SELECT 1 FROM sys.index_columns ic JOIN sys.indexes i ON i.object_id=ic.object_id AND i.index_id=ic.index_id WHERE ic.object_id=c.object_id AND ic.column_id=c.column_id AND i.is_unique_constraint=1) THEN 1 ELSE 0 END
FROM sys.columns c
JOIN sys.types ty ON ty.user_type_id=c.user_type_id
LEFT JOIN sys.default_constraints dc ON dc.object_id=c.default_object_id
LEFT JOIN sys.extended_properties ep ON ep.major_id=c.object_id AND ep.minor_id=c.column_id AND ep.class=1
JOIN sys.tables t ON t.object_id=c.object_id
JOIN sys.schemas s ON s.schema_id=t.schema_id
WHERE s.name=%s AND t.name=%s
ORDER BY c.column_id`, s, t)
}

func (m *mssqlDialect) IndexesSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT i.name, 'btree',
  CASE WHEN i.is_unique=1 THEN 1 ELSE 0 END,
  CASE WHEN i.is_primary_key=1 THEN 1 ELSE 0 END,
  (SELECT STRING_AGG(c2.name, ', ') WITHIN GROUP (ORDER BY ic.key_ordinal)
     FROM sys.index_columns ic JOIN sys.columns c2 ON c2.object_id=ic.object_id AND c2.column_id=ic.column_id
     WHERE ic.object_id=i.object_id AND ic.index_id=i.index_id),
  ISNULL(i.filter_definition,'')
FROM sys.indexes i
JOIN sys.tables t ON t.object_id=i.object_id
JOIN sys.schemas s ON s.schema_id=t.schema_id
WHERE s.name=%s AND t.name=%s`, s, t)
}

func (m *mssqlDialect) ConstraintsSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT kc.name, CASE kc.type WHEN 'PK' THEN 'PRIMARY KEY' WHEN 'UQ' THEN 'UNIQUE' END,
  (SELECT STRING_AGG(c2.name, ', ') WITHIN GROUP (ORDER BY ic.key_ordinal)
     FROM sys.index_columns ic JOIN sys.columns c2 ON c2.object_id=ic.object_id AND c2.column_id=ic.column_id
     WHERE ic.object_id=kc.parent_object_id AND ic.index_id=kc.unique_index_id),
  ''
FROM sys.key_constraints kc
JOIN sys.tables t ON t.object_id=kc.parent_object_id
JOIN sys.schemas s ON s.schema_id=t.schema_id
WHERE s.name=%s AND t.name=%s AND kc.type IN ('PK','UQ')
UNION ALL
SELECT cc.name, 'CHECK', '', ISNULL(cc.definition,'')
FROM sys.check_constraints cc
JOIN sys.tables t ON t.object_id=cc.parent_object_id
JOIN sys.schemas s ON s.schema_id=t.schema_id
WHERE s.name=%s AND t.name=%s`, s, t, s, t)
}

func (m *mssqlDialect) ForeignKeysSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT fk.name,
  (SELECT STRING_AGG(c.name, ', ') WITHIN GROUP (ORDER BY fkc.constraint_column_id)
     FROM sys.foreign_key_columns fkc JOIN sys.columns c ON c.object_id=fkc.parent_object_id AND c.column_id=fkc.parent_column_id
     WHERE fkc.constraint_object_id=fk.object_id),
  rt.name,
  (SELECT STRING_AGG(c.name, ', ') WITHIN GROUP (ORDER BY fkc.constraint_column_id)
     FROM sys.foreign_key_columns fkc JOIN sys.columns c ON c.object_id=fkc.referenced_object_id AND c.column_id=fkc.referenced_column_id
     WHERE fkc.constraint_object_id=fk.object_id),
  CASE fk.delete_referential_action WHEN 1 THEN 'CASCADE' WHEN 2 THEN 'SET NULL' WHEN 3 THEN 'SET DEFAULT' ELSE 'NO ACTION' END,
  CASE fk.update_referential_action WHEN 1 THEN 'CASCADE' WHEN 2 THEN 'SET NULL' WHEN 3 THEN 'SET DEFAULT' ELSE 'NO ACTION' END
FROM sys.foreign_keys fk
JOIN sys.tables t ON t.object_id=fk.parent_object_id
JOIN sys.schemas s ON s.schema_id=t.schema_id
JOIN sys.tables rt ON rt.object_id=fk.referenced_object_id
WHERE s.name=%s AND t.name=%s`, s, t)
}

func (m *mssqlDialect) TriggersSQL(schema, table string) string {
	s := lit(schema)
	t := lit(table)
	return fmt.Sprintf(`SELECT tr.name,
  CASE tr.type WHEN 2 THEN 'AFTER' ELSE 'INSTEAD OF' END,
  (SELECT STRING_AGG(te.type_desc, ', ') FROM sys.trigger_events te WHERE te.object_id=tr.object_id),
  ISNULL(OBJECT_DEFINITION(tr.object_id),''),
  CASE tr.is_disabled WHEN 1 THEN 'NO' ELSE 'YES' END
FROM sys.triggers tr
JOIN sys.tables t ON t.object_id=tr.parent_id
JOIN sys.schemas s ON s.schema_id=t.schema_id
WHERE s.name=%s AND t.name=%s`, s, t)
}
