package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"aether/internal/config"
)

var phRe = regexp.MustCompile(`\?`)

func rewritePlaceholders(q string) string {
	if !strings.Contains(q, "?") {
		return q
	}
	n := 0
	return phRe.ReplaceAllStringFunc(q, func(string) string {
		n++
		return "$" + strconv.Itoa(n)
	})
}

type SQL struct {
	*sql.DB
}

func (d *SQL) Exec(query string, args ...any) (sql.Result, error) {
	return d.DB.Exec(rewritePlaceholders(query), args...)
}

func (d *SQL) Query(query string, args ...any) (*sql.Rows, error) {
	return d.DB.Query(rewritePlaceholders(query), args...)
}

func (d *SQL) QueryRow(query string, args ...any) *sql.Row {
	return d.DB.QueryRow(rewritePlaceholders(query), args...)
}

func (d *SQL) Begin() (*Tx, error) {
	tx, err := d.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx}, nil
}

func (d *SQL) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := d.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx}, nil
}

type Tx struct {
	*sql.Tx
}

func (t *Tx) Exec(query string, args ...any) (sql.Result, error) {
	return t.Tx.Exec(rewritePlaceholders(query), args...)
}

func (t *Tx) Query(query string, args ...any) (*sql.Rows, error) {
	return t.Tx.Query(rewritePlaceholders(query), args...)
}

func (t *Tx) QueryRow(query string, args ...any) *sql.Row {
	return t.Tx.QueryRow(rewritePlaceholders(query), args...)
}

func dsn(cfg *config.Config) string {
	if cfg.DatabaseURL != "" {
		return cfg.DatabaseURL
	}
	u := url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%d", cfg.DatabaseHost, cfg.DatabasePort),
		Path:   "/" + cfg.DatabaseName,
	}
	if cfg.DatabaseUser != "" {
		u.User = url.UserPassword(cfg.DatabaseUser, cfg.DatabasePassword)
	}
	q := u.Query()
	q.Set("sslmode", cfg.DatabaseSSLMode)
	q.Set("application_name", cfg.DatabaseApplicationName)
	if cfg.DatabaseConnectTimeout > 0 {
		q.Set("connect_timeout", strconv.Itoa(cfg.DatabaseConnectTimeout))
	}
	if cfg.DatabaseSchema != "" {
		q.Set("options", "-csearch_path="+cfg.DatabaseSchema)
	}
	if cfg.DatabaseStatementTimeout > 0 {
		q.Set("statement_timeout", strconv.Itoa(cfg.DatabaseStatementTimeout*1000))
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func openConn(cfg *config.Config) (*sql.DB, error) {
	sqldb, err := sql.Open("pgx", dsn(cfg))
	if err != nil {
		return nil, err
	}
	sqldb.SetMaxOpenConns(cfg.DatabasePoolMax)
	sqldb.SetMaxIdleConns(cfg.DatabasePoolMin)
	sqldb.SetConnMaxIdleTime(time.Duration(cfg.DatabaseIdleTimeout) * time.Second)
	return sqldb, nil
}

func Open(cfg *config.Config) (*SQL, error) {
	attempts := cfg.DatabaseRetryAttempts
	if attempts < 1 {
		attempts = 1
	}
	delay := cfg.DatabaseRetryDelay
	if delay < 1 {
		delay = 1
	}
	var lastErr error
	for i := 1; i <= attempts; i++ {
		if i > 1 {
			log.Printf("[db] tentativa %d/%d em %ds... (último erro: %v)", i, attempts, delay, lastErr)
			time.Sleep(time.Duration(delay) * time.Second)
			delay = delay * 2
			if delay > 30 {
				delay = 30
			}
		}
		log.Printf("[db] connecting to postgres %s:%d/%s...", cfg.DatabaseHost, cfg.DatabasePort, cfg.DatabaseName)
		sqldb, err := openConn(cfg)
		if err != nil {
			lastErr = err
			continue
		}
		if err := sqldb.Ping(); err != nil {
			if strings.Contains(err.Error(), `database "`+cfg.DatabaseName+`" does not exist`) {
				if cerr := createDatabase(cfg); cerr != nil {
					lastErr = fmt.Errorf("criar banco: %v", cerr)
					sqldb.Close()
					continue
				}
				sqldb.Close()
				sqldb, err = openConn(cfg)
				if err == nil {
					err = sqldb.Ping()
				}
			}
			if err != nil {
				lastErr = err
				sqldb.Close()
				continue
			}
		}
		if err := checkVersion(sqldb); err != nil {
			lastErr = err
			sqldb.Close()
			return nil, err
		}
		if err := ensureSchema(sqldb, cfg.DatabaseSchema); err != nil {
			lastErr = err
			sqldb.Close()
			return nil, err
		}
		sqlDb := &SQL{DB: sqldb}
		log.Printf("[db] database connected.")
		if cfg.DatabaseMigrateOnStart {
			log.Printf("[db] checking migrations...")
			if err := Migrate(sqlDb); err != nil {
				lastErr = err
				sqldb.Close()
				return nil, err
			}
			log.Printf("[db] migrations completed (schema v%d).", schemaVersion(sqlDb))
		}
		return sqlDb, nil
	}
	return nil, fmt.Errorf("conexão com postgres falhou após %d tentativas: %w", attempts, lastErr)
}

func checkVersion(sqldb *sql.DB) error {
	var num int
	if err := sqldb.QueryRow(`SELECT current_setting('server_version_num')::int`).Scan(&num); err != nil {
		return fmt.Errorf("ler versão do postgres: %w", err)
	}
	if num < 150000 {
		return fmt.Errorf("postgres %d detectado; versão mínima suportada é 15", num/10000)
	}
	return nil
}

func ensureSchema(sqldb *sql.DB, schema string) error {
	if schema == "" || schema == "public" {
		return nil
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(schema) {
		return fmt.Errorf("schema inválido: %s", schema)
	}
	_, err := sqldb.Exec(`CREATE SCHEMA IF NOT EXISTS ` + schema)
	return err
}

func createDatabase(cfg *config.Config) error {
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(cfg.DatabaseName) {
		return fmt.Errorf("nome de banco inválido: %s", cfg.DatabaseName)
	}
	main := *cfg
	main.DatabaseName = "postgres"
	sqldb, err := openConn(&main)
	if err != nil {
		return err
	}
	defer sqldb.Close()
	var exists bool
	if err := sqldb.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)`, cfg.DatabaseName).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		if _, err := sqldb.Exec(`CREATE DATABASE ` + cfg.DatabaseName); err != nil {
			return err
		}
		log.Printf("[db] banco %q criado.", cfg.DatabaseName)
	}
	return nil
}

func schemaVersion(d *SQL) int {
	var v int
	_ = d.QueryRow(`SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&v)
	return v
}

func Health(d *SQL) map[string]any {
	out := map[string]any{"db": "down"}
	start := time.Now()
	var one int
	if err := d.QueryRow(`SELECT 1`).Scan(&one); err != nil {
		return out
	}
	latency := time.Since(start).Milliseconds()
	var version string
	_ = d.QueryRow(`SELECT current_setting('server_version')`).Scan(&version)
	stats := d.Stats()
	return map[string]any{
		"db":               "up",
		"latency_ms":       latency,
		"version":          version,
		"open_connections": stats.OpenConnections,
		"in_use":           stats.InUse,
		"idle":             stats.Idle,
	}
}
