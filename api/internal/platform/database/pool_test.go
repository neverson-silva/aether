package database

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestLegacyMigrationBaseline(t *testing.T) {
	tests := []struct {
		name    string
		version string
		legacy  bool
	}{
		{name: "last baseline migration", version: "0027_previous.sql", legacy: true},
		{name: "service migration", version: "0028_services_catalog.sql", legacy: false},
		{name: "current migration", version: "0043_alert_trigger_split.sql", legacy: false},
		{name: "invalid filename", version: "migration.sql", legacy: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := legacyMigration(test.version); got != test.legacy {
				t.Fatalf("legacyMigration(%q) = %t, want %t", test.version, got, test.legacy)
			}
		})
	}
}

func TestCleanMigrationSequence(t *testing.T) {
	if os.Getenv("AETHER_TEST_DATABASE_PORT") == "" {
		t.Skip("clean migration test requires AETHER_TEST_DATABASE_PORT")
	}
	port, err := strconv.Atoi(os.Getenv("AETHER_TEST_DATABASE_PORT"))
	if err != nil {
		t.Fatal(err)
	}
	user := os.Getenv("AETHER_TEST_DATABASE_USER")
	password := os.Getenv("AETHER_TEST_DATABASE_PASSWORD")
	if user == "" || password == "" {
		t.Fatal("AETHER_TEST_DATABASE_USER and AETHER_TEST_DATABASE_PASSWORD are required")
	}
	name := "aether_clean_migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	cfg := Config{Host: "127.0.0.1", Port: port, Name: name, User: user, Password: password, SSLMode: "disable", PoolMax: 4, ConnectTimeout: 5}
	ctx := context.Background()
	if err := EnsureDatabase(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	pool, err := Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := Migrate(ctx, pool, "../../../db/migrations"); err != nil {
		t.Fatal(err)
	}
}
