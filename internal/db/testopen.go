package db

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"

	"aether/internal/config"
)

type testTB interface {
	Fatal(...any)
	Name() string
	Cleanup(func())
}

type testT struct {
	name string
	fn   func(...any)
}

func (t *testT) Fatal(args ...any) { t.fn(args...) }
func (t *testT) Name() string      { return t.name }
func (t *testT) Cleanup(func())    {}

func OpenTest(t testTB) *SQL {
	cfg := TestConfig(t)
	cfg.DatabaseSchema = "t_" + schemaName(t.Name())
	sqldb, err := Open(cfg)
	if err != nil {
		t.Fatal("postgres indisponível: " + err.Error() + " — inicie o banco de teste")
	}
	t.Cleanup(func() {
		CleanupTestSchema(cfg)
		sqldb.Close()
	})
	return sqldb
}

func CleanupTestSchema(cfg *config.Config) {
	dropSchema(cfg)
}

func dropSchema(cfg *config.Config) {
	main := *cfg
	main.DatabaseSchema = ""
	conn, err := openConn(&main)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.Exec(`DROP SCHEMA IF EXISTS ` + cfg.DatabaseSchema + ` CASCADE`)
}

func SchemaNameFor(name string) string {
	return schemaName(name)
}

func schemaName(name string) string {
	sum := sha256.Sum256([]byte(name))
	return hex.EncodeToString(sum[:])[:10]
}

func TestConfig(t testTB) *config.Config {
	port := 5432
	if v := os.Getenv("AETHER_TEST_DATABASE_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			port = n
		}
	}
	cfg := &config.Config{
		DatabaseHost:           envOr2("AETHER_TEST_DATABASE_HOST", "127.0.0.1"),
		DatabasePort:           port,
		DatabaseName:           envOr2("AETHER_TEST_DATABASE_NAME", "aether_test"),
		DatabaseUser:           envOr2("AETHER_TEST_DATABASE_USER", "postgres"),
		DatabasePassword:       envOr2("AETHER_TEST_DATABASE_PASSWORD", "postgres"),
		DatabaseSSLMode:        "disable",
		DatabasePoolMin:        2,
		DatabasePoolMax:        10,
		DatabaseConnectTimeout: 5,
		DatabaseMigrateOnStart: true,
		DatabaseRetryAttempts:  3,
		DatabaseRetryDelay:     1,
	}
	return cfg
}

func envOr2(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func ConfigFromEnvPublic(t testTB) *config.Config {
	return TestConfig(t)
}
