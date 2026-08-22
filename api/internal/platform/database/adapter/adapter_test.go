package adapter

import (
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		sql  string
		want sqlClass
	}{
		{"SELECT * FROM users", classRead},
		{"EXPLAIN SELECT 1", classRead},
		{"WITH x AS (SELECT 1) SELECT * FROM x", classRead},
		{"SHOW tables", classRead},
		{"DELETE FROM users", classDanger},
		{"TRUNCATE users", classDanger},
		{"DROP TABLE users", classDanger},
		{"CREATE TABLE t (id int)", classDDL},
		{"ALTER TABLE t ADD c int", classDDL},
		{"INSERT INTO t VALUES (1)", classWrite},
		{"UPDATE t SET a=1", classWrite},
	}
	for _, c := range cases {
		if got := classify(c.sql); got != c.want {
			t.Errorf("classify(%q) = %v, want %v", c.sql, got, c.want)
		}
	}
}

func TestIsReadOnly(t *testing.T) {
	if !isReadOnly("SELECT * FROM users") {
		t.Error("SELECT should be read-only")
	}
	if !isReadOnly("WITH x AS (SELECT 1) SELECT * FROM x") {
		t.Error("WITH...SELECT should be read-only")
	}
	if isReadOnly("DELETE FROM users") {
		t.Error("DELETE should not be read-only")
	}
}

func TestCSVParser(t *testing.T) {
	in := "id,name,active,meta\n1,\"John, Jr\",true,\"{}\"\nNULL,NULL,NULL,NULL\n2,false,\"oops\",hello\n"
	rows, truncated := csvParser{}.Parse(in, 10, nil)
	if truncated {
		t.Fatal("should not truncate")
	}
	if len(rows.columns) != 4 {
		t.Fatalf("columns = %v", rows.columns)
	}
	if len(rows.rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(rows.rows))
	}
	if rows.rows[0][1] != "John, Jr" {
		t.Errorf("quoted cell = %v", rows.rows[0][1])
	}
	if rows.rows[0][0] != int64(1) {
		t.Errorf("int cell = %v", rows.rows[0][0])
	}
	if rows.rows[1][0] != nil {
		t.Errorf("NULL should map to nil, got %v", rows.rows[1][0])
	}
}

func TestCSVParserTruncates(t *testing.T) {
	var b strings.Builder
	b.WriteString("c\n")
	for i := 0; i < 50; i++ {
		b.WriteString("x\n")
	}
	rows, truncated := csvParser{}.Parse(b.String(), 10, nil)
	if !truncated {
		t.Fatal("should truncate over maxRows")
	}
	if len(rows.rows) != 10 {
		t.Errorf("rows = %d, want 10", len(rows.rows))
	}
}

func TestPipeParserUsesKnownSchema(t *testing.T) {
	schema := []Column{
		{Name: "id", Type: "int4"},
		{Name: "full_name", Type: "text"},
		{Name: "email", Type: "text"},
	}
	in := "|id|name|email|x|\n|----|----|----|----|\n|1|John|john@x|extra|\n|2|Jane\n"
	rows, _ := (pipeParser{}).Parse(in, 10, schema)
	if !stringsEqual(rows.columns, []string{"id", "full_name", "email"}) {
		t.Fatalf("columns = %v, want schema names", rows.columns)
	}
	if len(rows.rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows.rows))
	}
	if len(rows.rows[0]) != 3 {
		t.Fatalf("row0 should be trimmed to schema width, got %d", len(rows.rows[0]))
	}
	if len(rows.rows[1]) != 3 {
		t.Fatalf("row1 should be padded to schema width, got %d", len(rows.rows[1]))
	}
	if rows.rows[1][2] != nil {
		t.Fatalf("padded missing cell should be nil, got %v", rows.rows[1][2])
	}
}

func TestBuildTableQuery(t *testing.T) {
	q := buildTableQuery(&postgresDialect{}, "public", "users", QueryOptions{Limit: 100, Sort: "id", Order: "desc"})
	if !strings.Contains(q, `"public"."users"`) || !strings.Contains(q, "ORDER BY \"id\" DESC") || !strings.Contains(q, "LIMIT 100") {
		t.Errorf("unexpected query: %s", q)
	}
}

func TestRedisSplitCommand(t *testing.T) {
	cases := map[string][]string{
		"GET mykey":             {"GET", "mykey"},
		"SET name \"John Doe\"": {"SET", "name", "John Doe"},
		"LRANGE mylist 0 -1":    {"LRANGE", "mylist", "0", "-1"},
		"SET k 'single quoted'": {"SET", "k", "single quoted"},
	}
	for in, want := range cases {
		if got := splitCommand(in); !stringsEqual(got, want) {
			t.Errorf("splitCommand(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestMongoRows(t *testing.T) {
	docs := []map[string]interface{}{
		{"_id": "a", "name": "x", "meta": map[string]interface{}{"role": "admin"}},
		{"_id": "b", "name": "y"},
	}
	res := mongoRows(docs)
	if len(res.Columns) != 3 || len(res.Rows) != 2 {
		t.Fatalf("mongoRows cols=%v rows=%d", res.Columns, len(res.Rows))
	}
	if s, ok := res.Rows[0][2].(string); !ok || !strings.Contains(s, "admin") {
		t.Errorf("nested json cell = %v", res.Rows[0][2])
	}
}

func stringsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
