package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/databases/domain"
	"aether/internal/platform/database/adapter"
	"aether/internal/platform/druntime/cache"
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

func studioErr(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	for _, token := range []string{"connection refused", "could not connect", "connect to server", "connection reset", "failed to connect", "dial tcp", "no such host", "unreachable", "password authentication failed", "permission denied", "access denied", "authentication failed", "container", "exec in container"} {
		if strings.Contains(msg, token) {
			return fmt.Errorf("%w: %s", domain.ErrDatabaseUnavailable, err)
		}
	}
	return err
}

func (s *Studio) Engine(ctx context.Context, id, orgID uuid.UUID) (string, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return "", err
	}
	return a.Engine(), nil
}

type CatalogEntry struct {
	Schema      string               `json:"schema"`
	Name        string               `json:"name"`
	Type        string               `json:"type"`
	Columns     []adapter.Column     `json:"columns"`
	ForeignKeys []adapter.ForeignKey `json:"foreign_keys"`
}

func (s *Studio) Catalog(ctx context.Context, id, orgID uuid.UUID) ([]CatalogEntry, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return nil, studioErr(err)
	}
	schemas, err := a.IntrospectSchemas(ctx)
	if err != nil {
		return nil, studioErr(err)
	}
	out := []CatalogEntry{}
	for _, schema := range schemas {
		objs, err := a.ListObjects(ctx, schema)
		if err != nil {
			continue
		}
		for _, o := range objs {
			if o.Type != "table" && o.Type != "view" && o.Type != "materialized view" && o.Type != "collection" {
				continue
			}
			detail, err := a.IntrospectTable(ctx, schema, o.Name)
			if err != nil {
				continue
			}
			out = append(out, CatalogEntry{
				Schema: schema, Name: o.Name, Type: o.Type,
				Columns: detail.Columns, ForeignKeys: detail.ForeignKeys,
			})
		}
	}
	return out, nil
}

func (s *Studio) IntrospectMeta(ctx context.Context, id, orgID uuid.UUID) (adapter.Meta, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return adapter.Meta{}, studioErr(err)
	}
	return a.IntrospectMeta(ctx)
}

func (s *Studio) IntrospectSchemas(ctx context.Context, id, orgID uuid.UUID) ([]string, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return nil, studioErr(err)
	}
	return a.IntrospectSchemas(ctx)
}

func (s *Studio) IntrospectObjects(ctx context.Context, id, orgID uuid.UUID, schema string) (adapter.ObjectSummary, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return adapter.ObjectSummary{}, studioErr(err)
	}
	return a.IntrospectObjects(ctx, schema)
}

func (s *Studio) ListObjects(ctx context.Context, id, orgID uuid.UUID, schema string) ([]adapter.Object, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return nil, studioErr(err)
	}
	return a.ListObjects(ctx, schema)
}

func (s *Studio) IntrospectTable(ctx context.Context, id, orgID uuid.UUID, schema, table string) (adapter.TableDetail, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return adapter.TableDetail{}, studioErr(err)
	}
	return a.IntrospectTable(ctx, schema, table)
}

func (s *Studio) TableData(ctx context.Context, id, orgID uuid.UUID, schema, table string, opts adapter.QueryOptions) (adapter.QueryResult, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return adapter.QueryResult{}, studioErr(err)
	}
	return a.TableData(ctx, schema, table, opts)
}

func (s *Studio) Query(ctx context.Context, id, orgID uuid.UUID, sql string, opts adapter.QueryOptions) (adapter.QueryResult, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return adapter.QueryResult{}, studioErr(err)
	}
	return a.Query(ctx, sql, opts)
}

func (s *Studio) Exec(ctx context.Context, id, orgID uuid.UUID, sql string) (adapter.ExecResult, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return adapter.ExecResult{}, studioErr(err)
	}
	return a.Exec(ctx, sql)
}

func quoteIdent(engine string, name string) string {
	switch engine {
	case string(domain.EngineMysql), string(domain.EngineMariaDB):
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	case string(domain.EngineMSSQL):
		return "[" + name + "]"
	default:
		return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
	}
}

// CreateTable builds and executes a CREATE TABLE statement for the destination
// database engine. Columns are validated and quoted per engine dialect.
func (s *Studio) CreateTable(ctx context.Context, id, orgID uuid.UUID, input domain.CreateTableInput) (adapter.ExecResult, error) {
	if strings.TrimSpace(input.Table) == "" || len(input.Columns) == 0 {
		return adapter.ExecResult{}, domain.ErrValidation
	}
	if strings.TrimSpace(input.Schema) == "" {
		input.Schema = "public"
	}
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return adapter.ExecResult{}, studioErr(err)
	}
	engine := a.Engine()
	var lines []string
	var primaryCols []string
	for _, c := range input.Columns {
		name := strings.TrimSpace(c.Name)
		if name == "" {
			continue
		}
		colType := strings.ToUpper(strings.TrimSpace(c.Type))
		if colType == "" {
			colType = "TEXT"
		}
		line := "  " + quoteIdent(engine, name) + " " + colType
		if c.Primary {
			line += " PRIMARY KEY"
			primaryCols = append(primaryCols, quoteIdent(engine, name))
		} else if !c.Nullable {
			line += " NOT NULL"
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return adapter.ExecResult{}, domain.ErrValidation
	}
	tableQ := quoteIdent(engine, input.Table)
	schemaQ := quoteIdent(engine, input.Schema)
	qualified := tableQ
	if engine != string(domain.EngineMysql) && engine != string(domain.EngineMariaDB) {
		qualified = schemaQ + "." + tableQ
	}
	stmt := "CREATE TABLE " + qualified + " (\n" + strings.Join(lines, ",\n")
	if len(primaryCols) > 1 {
		stmt += ",\n  PRIMARY KEY (" + strings.Join(primaryCols, ", ") + ")"
	}
	stmt += "\n);"
	res, err := a.Exec(ctx, stmt)
	if err == nil {
		if cached, ok := a.(*cachedAdapter); ok {
			cached.invalidate(ctx)
		}
	}
	return res, err
}

func qualifiedTable(engine, schema, table string) string {
	tq := quoteIdent(engine, table)
	if engine == string(domain.EngineMysql) || engine == string(domain.EngineMariaDB) {
		return tq
	}
	return quoteIdent(engine, schema) + "." + tq
}

func (s *Studio) runDDL(ctx context.Context, id, orgID uuid.UUID, stmt string) (adapter.ExecResult, error) {
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return adapter.ExecResult{}, studioErr(err)
	}
	res, err := a.Exec(ctx, stmt)
	if err == nil {
		if cached, ok := a.(*cachedAdapter); ok {
			cached.invalidate(ctx)
		}
	}
	return res, err
}

// RenameTable renames a table (ALTER TABLE ... RENAME TO).
func (s *Studio) RenameTable(ctx context.Context, id, orgID uuid.UUID, schema, table, newName string) (adapter.ExecResult, error) {
	schema = strings.TrimSpace(schema)
	table = strings.TrimSpace(table)
	newName = strings.TrimSpace(newName)
	if schema == "" || table == "" || newName == "" {
		return adapter.ExecResult{}, domain.ErrValidation
	}
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return adapter.ExecResult{}, studioErr(err)
	}
	engine := a.Engine()
	stmt := "ALTER TABLE " + qualifiedTable(engine, schema, table) + " RENAME TO " + quoteIdent(engine, newName)
	return s.runDDL(ctx, id, orgID, stmt)
}

// DropTable drops a table (DROP TABLE).
func (s *Studio) DropTable(ctx context.Context, id, orgID uuid.UUID, schema, table string) (adapter.ExecResult, error) {
	schema = strings.TrimSpace(schema)
	table = strings.TrimSpace(table)
	if schema == "" || table == "" {
		return adapter.ExecResult{}, domain.ErrValidation
	}
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return adapter.ExecResult{}, studioErr(err)
	}
	engine := a.Engine()
	stmt := "DROP TABLE " + qualifiedTable(engine, schema, table)
	return s.runDDL(ctx, id, orgID, stmt)
}

// AlterTable applies a column diff (add / drop / alter type/nullable/default)
// generated server-side from the desired column set. Column renames are not
// inferred (ambiguous); they are separate DDL. Fails if nothing changed.
func (s *Studio) AlterTable(ctx context.Context, id, orgID uuid.UUID, schema, table string, columns []domain.TableColumn) (adapter.ExecResult, error) {
	schema = strings.TrimSpace(schema)
	table = strings.TrimSpace(table)
	if schema == "" || table == "" {
		return adapter.ExecResult{}, domain.ErrValidation
	}
	a, err := s.adapterFor(ctx, id, orgID)
	if err != nil {
		return adapter.ExecResult{}, studioErr(err)
	}
	engine := a.Engine()
	detail, err := a.IntrospectTable(ctx, schema, table)
	if err != nil {
		return adapter.ExecResult{}, studioErr(err)
	}
	existing := map[string]adapter.Column{}
	for _, c := range detail.Columns {
		existing[c.Name] = c
	}
	want := map[string]domain.TableColumn{}
	for _, c := range columns {
		if strings.TrimSpace(c.Name) != "" {
			want[c.Name] = c
		}
	}

	qualified := qualifiedTable(engine, schema, table)
	var stmts []string
	dropCols := []string{}
	for name := range existing {
		if _, ok := want[name]; !ok {
			dropCols = append(dropCols, quoteIdent(engine, name))
		}
	}
	if len(dropCols) > 0 {
		stmts = append(stmts, "ALTER TABLE "+qualified+" DROP COLUMN IF EXISTS "+strings.Join(dropCols, ", DROP COLUMN IF EXISTS "))
	}
	for name, c := range want {
		cur, ok := existing[name]
		if !ok {
			line := "ADD COLUMN " + quoteIdent(engine, c.Name) + " " + strings.ToUpper(strings.TrimSpace(c.Type))
			if c.Primary {
				line += " PRIMARY KEY"
			} else if !c.Nullable {
				line += " NOT NULL"
			}
			if c.Default != "" {
				line += " DEFAULT " + c.Default
			}
			stmts = append(stmts, "ALTER TABLE "+qualified+" "+line)
			continue
		}
		line := "ALTER COLUMN " + quoteIdent(engine, c.Name)
		subs := []string{}
		colType := strings.ToUpper(strings.TrimSpace(c.Type))
		if colType != "" && !strings.EqualFold(colType, cur.Type) {
			subs = append(subs, "TYPE "+colType)
		}
		if cur.Nullable && !c.Nullable {
			subs = append(subs, "SET NOT NULL")
		} else if !cur.Nullable && c.Nullable {
			subs = append(subs, "DROP NOT NULL")
		}
		if c.Default != "" && (cur.Default == nil || *cur.Default != c.Default) {
			subs = append(subs, "SET DEFAULT "+c.Default)
		} else if c.Default == "" && cur.Default != nil {
			subs = append(subs, "DROP DEFAULT")
		}
		if len(subs) > 0 {
			stmts = append(stmts, line+" "+strings.Join(subs, " "))
		}
	}
	if len(stmts) == 0 {
		return adapter.ExecResult{}, domain.ErrValidation
	}
	return s.runDDL(ctx, id, orgID, strings.Join(stmts, ";"+"\n"))
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
