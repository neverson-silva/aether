package adapter

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type liveRunner struct{}

func (liveRunner) Exec(ctx context.Context, containerID string, env []string, args ...string) (string, string, error) {
	pa := []string{"exec"}
	for _, e := range env {
		pa = append(pa, "-e", e)
	}
	pa = append(pa, containerID)
	pa = append(pa, args...)
	cmd := exec.CommandContext(ctx, "docker", pa...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	return out.String(), errb.String(), err
}

func TestLiveEngine(t *testing.T) {
	engine := os.Getenv("AETHER_TEST_ENGINE")
	container := os.Getenv("AETHER_TEST_CONTAINER")
	user := os.Getenv("AETHER_TEST_USER")
	pass := os.Getenv("AETHER_TEST_PASS")
	db := os.Getenv("AETHER_TEST_DB")
	if engine == "" || container == "" {
		t.Skip("set AETHER_TEST_* to run live")
	}
	ex := &Executor{Runner: liveRunner{}, ContainerID: container, Engine: engine, User: user, Pass: pass, DBName: db, Timeout: 20 * time.Second, MaxRows: 100}
	ctx := context.Background()
	a, err := New(engine, ex)
	if err != nil {
		t.Fatalf("New(%s): %v", engine, err)
	}
	meta, err := a.IntrospectMeta(ctx)
	if err != nil {
		t.Fatalf("[%s] meta: %v", engine, err)
	}
	t.Logf("[%s] meta: version=%q schemas=%d tables=%d", engine, meta.Version, meta.Schemas, meta.Tables)
	schemas, err := a.IntrospectSchemas(ctx)
	if err != nil {
		t.Fatalf("[%s] schemas: %v", engine, err)
	}
	t.Logf("[%s] schemas=%v", engine, schemas)
	if len(schemas) == 0 {
		t.Fatalf("[%s] no schemas", engine)
	}
	objs, err := a.IntrospectObjects(ctx, schemas[0])
	if err != nil {
		t.Fatalf("[%s] objects: %v", engine, err)
	}
	t.Logf("[%s] objects=%+v", engine, objs)

	if engine == "redis" {
		_, err = a.Exec(ctx, "SET studio_probe hello")
		if err != nil {
			t.Fatalf("[redis] exec SET: %v", err)
		}
		res, err := a.Query(ctx, "GET studio_probe", QueryOptions{})
		if err != nil {
			t.Fatalf("[redis] query GET: %v", err)
		}
		t.Logf("[redis] GET result rows=%d row0=%v", res.RowCount, res.Rows)
		return
	}

	seed := seedSQL(engine)
	if seed != "" {
		if _, err := a.Exec(ctx, seed); err != nil {
			t.Fatalf("[%s] seed exec: %v", engine, err)
		}
	}
	if engine == "mongodb" {
		seed := `db.studio_probe.drop(); db.studio_probe.insertMany([{name:"alpha",score:1},{name:"beta",score:2},{name:"gamma",meta:{role:"admin"}}]); db.studio_probe.find().count()`
		if _, err := a.Exec(ctx, seed); err != nil {
			t.Fatalf("[mongo] seed exec: %v", err)
		}
		td, err := a.IntrospectTable(ctx, db, "studio_probe")
		if err != nil {
			t.Fatalf("[mongo] table detail: %v", err)
		}
		t.Logf("[mongo] collection studio_probe: cols=%d idx=%d", len(td.Columns), len(td.Indexes))
		dr, err := a.TableData(ctx, db, "studio_probe", QueryOptions{Limit: 10})
		if err != nil {
			t.Fatalf("[mongo] table data: %v", err)
		}
		t.Logf("[mongo] data cols=%v rows=%d", dr.Columns, dr.RowCount)
		for _, r := range dr.Rows {
			t.Logf("[mongo]   row=%v", r)
		}
		return
	}

	objName := "studio_probe"
	introspectSchema := schemas[0]
	if engine == "oracle" {
		introspectSchema = strings.ToUpper(user)
	}
	td, err := a.IntrospectTable(ctx, introspectSchema, objName)
	if err != nil {
		t.Fatalf("[%s] table detail %s: %v", engine, objName, err)
	}
	t.Logf("[%s] table %s: cols=%d idx=%d con=%d fk=%d trig=%d", engine, objName, len(td.Columns), len(td.Indexes), len(td.Constraints), len(td.ForeignKeys), len(td.Triggers))
	for _, c := range td.Columns {
		t.Logf("[%s]   col %s %s nullable=%v pk=%v", engine, c.Name, c.Type, c.Nullable, c.Primary)
	}
	dr, err := a.TableData(ctx, introspectSchema, objName, QueryOptions{Limit: 10})
	if err != nil {
		t.Fatalf("[%s] table data: %v", engine, err)
	}
	t.Logf("[%s] data cols=%v rows=%d", engine, dr.Columns, dr.RowCount)
	for _, r := range dr.Rows {
		t.Logf("[%s]   row=%v", engine, r)
	}
	q, err := a.Query(ctx, fmt.Sprintf("SELECT * FROM %s LIMIT 5", objName), QueryOptions{})
	if err != nil {
		t.Logf("[%s] custom query err (ok if dialect differs): %v", engine, err)
	} else {
		t.Logf("[%s] custom query rows=%d", engine, q.RowCount)
	}
}

func seedSQL(engine string) string {
	switch engine {
	case "postgres":
		return `DROP TABLE IF EXISTS studio_probe; CREATE TABLE studio_probe (id bigserial PRIMARY KEY, name varchar(50) NOT NULL, created_at timestamptz DEFAULT now()); INSERT INTO studio_probe (name) VALUES ('alpha'),('beta'),('gamma');`
	case "mysql", "mariadb":
		return `DROP TABLE IF EXISTS studio_probe; CREATE TABLE studio_probe (id bigint AUTO_INCREMENT PRIMARY KEY, name varchar(50) NOT NULL, created_at timestamp DEFAULT CURRENT_TIMESTAMP); INSERT INTO studio_probe (name) VALUES ('alpha'),('beta'),('gamma');`
	case "mssql":
		return `IF OBJECT_ID('dbo.studio_probe','U') IS NOT NULL DROP TABLE dbo.studio_probe; CREATE TABLE dbo.studio_probe (id bigint IDENTITY PRIMARY KEY, name varchar(50) NOT NULL, created_at datetime2 DEFAULT getdate()); INSERT INTO studio_probe (name) VALUES ('alpha'),('beta'),('gamma');`
	case "oracle":
		return `BEGIN EXECUTE IMMEDIATE 'DROP TABLE studio_probe'; EXCEPTION WHEN OTHERS THEN NULL; END;
/
CREATE TABLE studio_probe (id NUMBER GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY, name VARCHAR2(50) NOT NULL, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
INSERT INTO studio_probe (name) VALUES ('alpha');
INSERT INTO studio_probe (name) VALUES ('beta');
INSERT INTO studio_probe (name) VALUES ('gamma');
COMMIT;`
	case "mongodb":
		return ""
	default:
		return ""
	}
}
