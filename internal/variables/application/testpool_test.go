package application

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"aether/internal/database"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	port := 5432
	if v := os.Getenv("AETHER_API_TEST_DATABASE_PORT"); v != "" {
		port = atoi(v, port)
	}
	if v := os.Getenv("AETHER_TEST_DATABASE_PORT"); v != "" {
		port = atoi(v, port)
	}
	user := os.Getenv("AETHER_API_TEST_DATABASE_USER")
	password := os.Getenv("AETHER_API_TEST_DATABASE_PASSWORD")
	if user == "" {
		if port == 5433 {
			user, password = "postgres", "postgres"
		} else {
			user = "aether"
			password = pgpassPassword()
		}
	}
	ctx := context.Background()
	dbCfg := database.Config{
		Host: "127.0.0.1", Port: port, Name: dbName("aether_api_test_variables"),
		User: user, Password: password, SSLMode: "disable",
		PoolMax: 8, ConnectTimeout: 5,
	}
	if err := database.EnsureDatabase(ctx, dbCfg); err != nil {
		t.Fatalf("criar banco de teste: %v", err)
	}
	pool, err := database.Open(ctx, dbCfg)
	if err != nil {
		t.Fatalf("postgres de teste indisponível: %v", err)
	}
	if err := database.Migrate(ctx, pool, "../../../db/migrations"); err != nil {
		pool.Close()
		t.Fatalf("migrate: %v", err)
	}
	for _, q := range []string{
		"TRUNCATE variable_audit, env_variables, project_assignments, app_volumes, registry_mirrors, out_webhooks, oidc_providers, s3_destinations, branding, pipeline_runs, pipelines, registry_settings, servers, clusters, snapshot_schedules, snapshots, notification_channels, notifications, alert_events, alert_rules, gitops, compose_apps, templates, backups, databases, autopilot_events, app_policies, workers, cron_jobs, previews, domains, deployments, app_env, apps, environments, projects RESTART IDENTITY CASCADE",
		"TRUNCATE audit_logs, api_keys, members, orgs, users RESTART IDENTITY CASCADE",
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			pool.Close()
			t.Fatalf("reset banco: %v", err)
		}
	}
	return pool
}

func pgpassPassword() string {
	home, _ := os.UserHomeDir()
	raw, err := os.ReadFile(home + "/.aether/.pgpass")
	if err != nil {
		return ""
	}
	content := strings.TrimSpace(string(raw))
	if strings.Contains(content, ":") {
		parts := strings.Split(content, ":")
		return parts[len(parts)-1]
	}
	return content
}

func atoi(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func dbName(def string) string {
	if v := os.Getenv("AETHER_API_TEST_DATABASE_NAME"); v != "" {
		return v
	}
	return def
}
