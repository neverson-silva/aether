package database

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Host             string
	Port             int
	Name             string
	User             string
	Password         string
	SSLMode          string
	PoolMax          int
	ConnectTimeout   int
	StatementTimeout int
}

func Open(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s pool_max_conns=%d",
		cfg.Host, cfg.Port, cfg.Name, cfg.User, cfg.Password, cfg.SSLMode, cfg.PoolMax,
	)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	connectCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.ConnectTimeout)*time.Second)
	defer cancel()
	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func EnsureDatabase(ctx context.Context, cfg Config) error {
	adminCfg := cfg
	adminCfg.Name = "postgres"
	dsn := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		adminCfg.Host, adminCfg.Port, adminCfg.Name, adminCfg.User, adminCfg.Password, adminCfg.SSLMode,
	)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	var exists bool
	if err := conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)`, cfg.Name).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = conn.Exec(ctx, `CREATE DATABASE `+quoteIdent(cfg.Name))
	return err
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func Migrate(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		version := e.Name()
		applied, err := migrationApplied(ctx, pool, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		raw, err := os.ReadFile(migrationsDir + "/" + e.Name())
		if err != nil {
			return err
		}
		if migrationMaterialized(ctx, pool) {
			if err := recordMigration(ctx, pool, version); err != nil {
				return err
			}
			continue
		}
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(raw)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migration %s: %w", version, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func migrationID(version string, intCol bool) int64 {
	var seed uint64
	for i := 0; i < len(version); i++ {
		seed = seed*31 + uint64(version[i])
	}
	if intCol {
		seed = seed % 2000000000
	}
	return int64(seed)
}

func recordMigration(ctx context.Context, pool *pgxpool.Pool, version string) error {
	var colType string
	if err := pool.QueryRow(ctx, `SELECT data_type FROM information_schema.columns WHERE table_name='schema_migrations' AND column_name='version'`).Scan(&colType); err != nil {
		return err
	}
	if strings.EqualFold(colType, "integer") || strings.EqualFold(colType, "bigint") {
		_, err := pool.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, migrationID(version, strings.EqualFold(colType, "integer")))
		return err
	}
	_, err := pool.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version)
	return err
}

func migrationMaterialized(ctx context.Context, pool *pgxpool.Pool) bool {
	var colType string
	if err := pool.QueryRow(ctx, `SELECT data_type FROM information_schema.columns WHERE table_name='schema_migrations' AND column_name='version'`).Scan(&colType); err != nil {
		return false
	}
	if !strings.EqualFold(colType, "integer") && !strings.EqualFold(colType, "bigint") {
		return false
	}
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('public.users') IS NOT NULL`).Scan(&exists); err != nil {
		return false
	}
	return exists
}

func migrationApplied(ctx context.Context, pool *pgxpool.Pool, version string) (bool, error) {
	var colType string
	if err := pool.QueryRow(ctx, `SELECT data_type FROM information_schema.columns WHERE table_name='schema_migrations' AND column_name='version'`).Scan(&colType); err != nil {
		return false, err
	}
	if strings.EqualFold(colType, "integer") || strings.EqualFold(colType, "bigint") {
		var n int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=$1`, migrationID(version, strings.EqualFold(colType, "integer"))).Scan(&n); err != nil {
			return false, err
		}
		return n > 0, nil
	}
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, version).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
