package db

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"
)

var migrations = []string{
	`
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS orgs (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  owner_user_id TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS members (
  org_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  role TEXT NOT NULL,
  PRIMARY KEY (org_id, user_id)
);
CREATE TABLE IF NOT EXISTS api_keys (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  name TEXT NOT NULL,
  key_hash TEXT NOT NULL,
  scopes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  last_used_at TEXT NOT NULL DEFAULT '',
  expires_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS projects (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  name TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS apps (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  project_id TEXT NOT NULL,
  name TEXT NOT NULL,
  source_type TEXT NOT NULL,
  image TEXT NOT NULL DEFAULT '',
  git_url TEXT NOT NULL DEFAULT '',
  git_branch TEXT NOT NULL DEFAULT 'main',
  dockerfile TEXT NOT NULL DEFAULT 'Dockerfile',
  port INTEGER NOT NULL DEFAULT 80,
  cpus TEXT NOT NULL DEFAULT '',
  mem_mb INTEGER NOT NULL DEFAULT 0,
  hc_enabled INTEGER NOT NULL DEFAULT 0,
  hc_path TEXT NOT NULL DEFAULT '/',
  hc_interval_ms INTEGER NOT NULL DEFAULT 5000,
  hc_timeout_ms INTEGER NOT NULL DEFAULT 2000,
  hc_retries INTEGER NOT NULL DEFAULT 3,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(org_id, name)
);
CREATE TABLE IF NOT EXISTS app_volumes (
  id TEXT PRIMARY KEY,
  app_id TEXT NOT NULL,
  name TEXT NOT NULL,
  mount_path TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS app_env (
  app_id TEXT NOT NULL,
  name TEXT NOT NULL,
  value TEXT NOT NULL,
  secret INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (app_id, name)
);
CREATE TABLE IF NOT EXISTS deployments (
  id TEXT PRIMARY KEY,
  app_id TEXT NOT NULL,
  number INTEGER NOT NULL,
  status TEXT NOT NULL,
  commit_sha TEXT NOT NULL DEFAULT '',
  image_ref TEXT NOT NULL DEFAULT '',
  container_id TEXT NOT NULL DEFAULT '',
  error TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT NOT NULL DEFAULT '',
  UNIQUE(app_id, number)
);
CREATE TABLE IF NOT EXISTS domains (
  id TEXT PRIMARY KEY,
  app_id TEXT NOT NULL,
  host TEXT NOT NULL,
  https INTEGER NOT NULL DEFAULT 0,
  cert_status TEXT NOT NULL DEFAULT 'none',
  created_at TEXT NOT NULL,
  UNIQUE(host)
);
CREATE TABLE IF NOT EXISTS backups (
  id TEXT PRIMARY KEY,
  path TEXT NOT NULL,
  size INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS events (
  aggregate_type TEXT NOT NULL,
  aggregate_id TEXT NOT NULL,
  sequence INTEGER NOT NULL,
  type TEXT NOT NULL,
  payload TEXT NOT NULL,
  ts BIGINT NOT NULL,
  published INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (aggregate_type, aggregate_id, sequence)
);
CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts);
CREATE TABLE IF NOT EXISTS consumer_checkpoint (
  consumer TEXT NOT NULL,
  aggregate_type TEXT NOT NULL,
  aggregate_id TEXT NOT NULL,
  sequence INTEGER NOT NULL,
  PRIMARY KEY (consumer, aggregate_type, aggregate_id)
);
CREATE TABLE IF NOT EXISTS certs (
  host TEXT PRIMARY KEY,
  cert_path TEXT NOT NULL,
  key_path TEXT NOT NULL,
  not_after TEXT NOT NULL,
  provider TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
`,
	`
ALTER TABLE apps ADD COLUMN webhook_secret TEXT NOT NULL DEFAULT '';
`,
	`
CREATE TABLE IF NOT EXISTS databases (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  project_id TEXT NOT NULL,
  name TEXT NOT NULL,
  engine TEXT NOT NULL,
  version TEXT NOT NULL DEFAULT '',
  port INTEGER NOT NULL DEFAULT 0,
  db_name TEXT NOT NULL,
  db_user TEXT NOT NULL,
  pass_enc TEXT NOT NULL,
  mem_mb INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'creating',
  container_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  UNIQUE(org_id, name)
);
CREATE TABLE IF NOT EXISTS cron_jobs (
  id TEXT PRIMARY KEY,
  app_id TEXT NOT NULL,
  name TEXT NOT NULL,
  schedule TEXT NOT NULL,
  command TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  last_run TEXT NOT NULL DEFAULT '',
  next_run TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS workers (
  id TEXT PRIMARY KEY,
  app_id TEXT NOT NULL,
  name TEXT NOT NULL,
  command TEXT NOT NULL,
  replicas INTEGER NOT NULL DEFAULT 1,
  enabled INTEGER NOT NULL DEFAULT 1,
  status TEXT NOT NULL DEFAULT 'stopped',
  container_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS s3_destinations (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  name TEXT NOT NULL,
  endpoint TEXT NOT NULL,
  bucket TEXT NOT NULL,
  region TEXT NOT NULL DEFAULT 'us-east-1',
  access_key_enc TEXT NOT NULL,
  secret_key_enc TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS notification_channels (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  config_enc TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS templates (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  category TEXT NOT NULL DEFAULT '',
  icon TEXT NOT NULL DEFAULT '',
  version TEXT NOT NULL DEFAULT '1',
  definition TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS previews (
  id TEXT PRIMARY KEY,
  app_id TEXT NOT NULL,
  branch TEXT NOT NULL,
  deployment_id TEXT NOT NULL DEFAULT '',
  container_id TEXT NOT NULL DEFAULT '',
  domain TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS compose_apps (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  project_id TEXT NOT NULL,
  name TEXT NOT NULL,
  compose TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'stopped',
  created_at TEXT NOT NULL
);
ALTER TABLE users ADD COLUMN totp_secret_enc TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN totp_enabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE backups ADD COLUMN kind TEXT NOT NULL DEFAULT 'state';
ALTER TABLE backups ADD COLUMN dest TEXT NOT NULL DEFAULT 'local';
ALTER TABLE backups ADD COLUMN app_id TEXT NOT NULL DEFAULT '';
`,
	`
ALTER TABLE apps ADD COLUMN build_type TEXT NOT NULL DEFAULT 'dockerfile';
ALTER TABLE apps ADD COLUMN preview_domain TEXT NOT NULL DEFAULT '';
CREATE TABLE IF NOT EXISTS out_webhooks (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  name TEXT NOT NULL,
  url TEXT NOT NULL,
  secret_enc TEXT NOT NULL DEFAULT '',
  events TEXT NOT NULL DEFAULT '',
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS registry_settings (
  id TEXT PRIMARY KEY,
  enabled INTEGER NOT NULL DEFAULT 0,
  host TEXT NOT NULL DEFAULT '127.0.0.1',
  port INTEGER NOT NULL DEFAULT 5000,
  container_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'stopped'
);
`,
	`
ALTER TABLE apps ADD COLUMN server_id TEXT NOT NULL DEFAULT '';
ALTER TABLE deployments ADD COLUMN server_id TEXT NOT NULL DEFAULT '';
CREATE TABLE IF NOT EXISTS servers (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  host TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL DEFAULT 'agent',
  status TEXT NOT NULL DEFAULT 'registered',
  version TEXT NOT NULL DEFAULT '',
  labels TEXT NOT NULL DEFAULT '',
  cpu_cores INTEGER NOT NULL DEFAULT 0,
  mem_total_bytes INTEGER NOT NULL DEFAULT 0,
  load REAL NOT NULL DEFAULT 0,
  last_heartbeat TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS server_tokens (
  token_hash TEXT PRIMARY KEY,
  server_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  expires_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS server_commands (
  id TEXT PRIMARY KEY,
  server_id TEXT NOT NULL,
  kind TEXT NOT NULL DEFAULT '',
  payload TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  delivered INTEGER NOT NULL DEFAULT 0
);
`,
	`
CREATE TABLE IF NOT EXISTS app_policies (
  app_id TEXT PRIMARY KEY,
  enabled INTEGER NOT NULL DEFAULT 0,
  cpu_min REAL NOT NULL DEFAULT 0.25,
  cpu_max REAL NOT NULL DEFAULT 4,
  mem_min_mb INTEGER NOT NULL DEFAULT 128,
  mem_max_mb INTEGER NOT NULL DEFAULT 2048,
  scale_up_pct INTEGER NOT NULL DEFAULT 80,
  scale_down_pct INTEGER NOT NULL DEFAULT 15,
  cooldown_min INTEGER NOT NULL DEFAULT 15,
  updated_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS autopilot_events (
  id TEXT PRIMARY KEY,
  app_id TEXT NOT NULL,
  action TEXT NOT NULL DEFAULT '',
  detail TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS gitops (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  name TEXT NOT NULL,
  repo_url TEXT NOT NULL,
  branch TEXT NOT NULL DEFAULT 'main',
  path TEXT NOT NULL DEFAULT 'aether.yml',
  target_org_id TEXT NOT NULL DEFAULT '',
  apply_mode TEXT NOT NULL DEFAULT 'manual',
  last_sha TEXT NOT NULL DEFAULT '',
  last_status TEXT NOT NULL DEFAULT 'pending',
  drift_added INTEGER NOT NULL DEFAULT 0,
  drift_changed INTEGER NOT NULL DEFAULT 0,
  drift_removed INTEGER NOT NULL DEFAULT 0,
  last_sync TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS registry_mirrors (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  source TEXT NOT NULL,
  dest TEXT NOT NULL,
  dest_tls_verify INTEGER NOT NULL DEFAULT 1,
  tags_filter TEXT NOT NULL DEFAULT '',
  schedule TEXT NOT NULL DEFAULT '',
  last_run TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'idle',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS snapshots (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  app_id TEXT NOT NULL DEFAULT '',
  volume TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL DEFAULT '',
  size INTEGER NOT NULL DEFAULT 0,
  chunks INTEGER NOT NULL DEFAULT 0,
  dedup_saved INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);
`,
	`
ALTER TABLE servers ADD COLUMN cluster_id TEXT NOT NULL DEFAULT '';
ALTER TABLE apps ADD COLUMN cluster_id TEXT NOT NULL DEFAULT '';
CREATE TABLE IF NOT EXISTS clusters (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  name TEXT NOT NULL,
  labels TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS branding (
  org_id TEXT PRIMARY KEY,
  name TEXT NOT NULL DEFAULT '',
  logo_url TEXT NOT NULL DEFAULT '',
  primary_color TEXT NOT NULL DEFAULT '',
  accent_color TEXT NOT NULL DEFAULT '',
  dark_mode INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS pipelines (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  app_id TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL,
  trigger TEXT NOT NULL DEFAULT 'manual',
  stages TEXT NOT NULL DEFAULT '[]',
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS pipeline_runs (
  id TEXT PRIMARY KEY,
  pipeline_id TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'running',
  trigger TEXT NOT NULL DEFAULT '',
  log TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL,
  finished_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS oidc_providers (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  name TEXT NOT NULL,
  issuer TEXT NOT NULL,
  client_id TEXT NOT NULL,
  client_secret_enc TEXT NOT NULL DEFAULT '',
  scopes TEXT NOT NULL DEFAULT 'openid email profile',
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL
);
`,
	`
ALTER TABLE apps ADD COLUMN environment_id TEXT NOT NULL DEFAULT '';
CREATE TABLE IF NOT EXISTS environments (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  name TEXT NOT NULL,
  slug TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  color TEXT NOT NULL DEFAULT '',
  is_default INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_env_project_slug ON environments(project_id, slug);
INSERT INTO environments(id, project_id, name, slug, description, color, is_default, created_at, updated_at)
SELECT 'env-' || gen_random_uuid()::text, id, 'production', 'production', '', '', 1, now(), now()
FROM projects WHERE id NOT IN (SELECT project_id FROM environments);
UPDATE apps SET environment_id = COALESCE((SELECT e.id FROM environments e WHERE e.project_id = apps.project_id AND e.is_default = 1), '')
WHERE environment_id = '';
`,
	`
CREATE TABLE IF NOT EXISTS env_variables (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  environment_id TEXT NOT NULL,
  key TEXT NOT NULL,
  value TEXT NOT NULL,
  is_secret INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (environment_id, key)
);
CREATE TABLE IF NOT EXISTS variable_audit (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  environment_id TEXT NOT NULL,
  action TEXT NOT NULL,
  var_key TEXT NOT NULL,
  previous_value TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);`,
	`
CREATE TABLE IF NOT EXISTS project_variables (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  key TEXT NOT NULL,
  value TEXT NOT NULL,
  is_secret INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (project_id, key)
);`,
	`
ALTER TABLE deployments ADD COLUMN triggered_by TEXT NOT NULL DEFAULT '';
CREATE TABLE IF NOT EXISTS notifications (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  type TEXT NOT NULL,
  message TEXT NOT NULL,
  payload TEXT NOT NULL DEFAULT '{}',
  read INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_notif_org_read ON notifications(org_id, read, created_at DESC);`,
	`
ALTER TABLE templates ADD COLUMN readme TEXT NOT NULL DEFAULT '';
ALTER TABLE templates ADD COLUMN homepage TEXT NOT NULL DEFAULT '';
ALTER TABLE templates ADD COLUMN github TEXT NOT NULL DEFAULT '';
ALTER TABLE templates ADD COLUMN license TEXT NOT NULL DEFAULT 'MIT';
ALTER TABLE templates ADD COLUMN installs INTEGER NOT NULL DEFAULT 0;
ALTER TABLE templates ADD COLUMN featured INTEGER NOT NULL DEFAULT 0;
ALTER TABLE templates ADD COLUMN verified INTEGER NOT NULL DEFAULT 1;
ALTER TABLE templates ADD COLUMN tags TEXT NOT NULL DEFAULT '';
ALTER TABLE templates ADD COLUMN updated_at TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_templates_category ON templates(category);
CREATE INDEX IF NOT EXISTS idx_templates_installs ON templates(installs DESC);`,
	`
ALTER TABLE deployments ADD COLUMN env_snapshot TEXT NOT NULL DEFAULT '';`,
	`
CREATE TABLE IF NOT EXISTS alert_rules (
	id TEXT PRIMARY KEY,
	org_id TEXT NOT NULL,
	name TEXT NOT NULL,
	metric TEXT NOT NULL,
	threshold REAL NOT NULL,
	window_s INTEGER NOT NULL DEFAULT 30,
	severity TEXT NOT NULL DEFAULT 'warning',
	enabled INTEGER NOT NULL DEFAULT 1,
	target_app TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_alert_rules_org ON alert_rules(org_id);
CREATE TABLE IF NOT EXISTS alert_events (
	id TEXT PRIMARY KEY,
	org_id TEXT NOT NULL,
	rule_id TEXT NOT NULL,
	app_id TEXT NOT NULL DEFAULT '',
	app_name TEXT NOT NULL DEFAULT '',
	severity TEXT NOT NULL,
	message TEXT NOT NULL,
	value REAL NOT NULL DEFAULT 0,
	threshold REAL NOT NULL DEFAULT 0,
	metric TEXT NOT NULL,
	created_at TEXT NOT NULL,
	resolved_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_alert_events_org ON alert_events(org_id, created_at);`,
	`
ALTER TABLE templates ADD COLUMN editors_choice INTEGER NOT NULL DEFAULT 0;`,
	`
CREATE TABLE IF NOT EXISTS snapshot_schedules (
	id TEXT PRIMARY KEY,
	org_id TEXT NOT NULL,
	app_id TEXT NOT NULL,
	volume TEXT NOT NULL,
	name_prefix TEXT NOT NULL DEFAULT '',
	cron TEXT NOT NULL,
	retention INTEGER NOT NULL DEFAULT 7,
	enabled INTEGER NOT NULL DEFAULT 1,
	last_run TEXT NOT NULL DEFAULT '',
	next_run TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_snap_sched_org ON snapshot_schedules(org_id);
CREATE INDEX IF NOT EXISTS idx_snap_sched_next ON snapshot_schedules(next_run);`,
	`
ALTER TABLE apps ADD COLUMN image_retention INTEGER NOT NULL DEFAULT 0;`,
	`
ALTER TABLE apps ADD COLUMN storage_mb INTEGER NOT NULL DEFAULT 0;
ALTER TABLE databases ADD COLUMN storage_mb INTEGER NOT NULL DEFAULT 0;`,
	`
ALTER TABLE apps ADD COLUMN upload_id TEXT NOT NULL DEFAULT '';
ALTER TABLE apps ADD COLUMN install_command TEXT NOT NULL DEFAULT '';
ALTER TABLE apps ADD COLUMN build_command TEXT NOT NULL DEFAULT '';
ALTER TABLE apps ADD COLUMN start_command TEXT NOT NULL DEFAULT '';
ALTER TABLE apps ADD COLUMN root_folder TEXT NOT NULL DEFAULT '';
ALTER TABLE apps ADD COLUMN dist_folder TEXT NOT NULL DEFAULT '';
ALTER TABLE apps ADD COLUMN watch_paths TEXT NOT NULL DEFAULT '';`,
	`
CREATE TABLE IF NOT EXISTS deployment_plans (
	id TEXT PRIMARY KEY,
	app_id TEXT NOT NULL,
	framework TEXT NOT NULL DEFAULT '',
	library TEXT NOT NULL DEFAULT '',
	package_manager TEXT NOT NULL DEFAULT '',
	runtime TEXT NOT NULL DEFAULT '',
	build_command TEXT NOT NULL DEFAULT '',
	install_command TEXT NOT NULL DEFAULT '',
	output_dir TEXT NOT NULL DEFAULT '',
	app_type TEXT NOT NULL DEFAULT '',
	web_server TEXT NOT NULL DEFAULT 'nginx',
	container_port INTEGER NOT NULL DEFAULT 80,
	spa_fallback INTEGER NOT NULL DEFAULT 0,
	index_file TEXT NOT NULL DEFAULT 'index.html',
	nginx_conf TEXT NOT NULL DEFAULT '',
	dockerfile TEXT NOT NULL DEFAULT '',
	warnings TEXT NOT NULL DEFAULT '',
	detected_at TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_deployment_plans_app ON deployment_plans(app_id);`,
	// migration 21 — Enterprise Organizations & Multi-Tenancy
	`
ALTER TABLE orgs ADD COLUMN IF NOT EXISTS slug TEXT NOT NULL DEFAULT '';
ALTER TABLE orgs ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
ALTER TABLE orgs ADD COLUMN IF NOT EXISTS avatar TEXT NOT NULL DEFAULT '';
ALTER TABLE orgs ADD COLUMN IF NOT EXISTS color TEXT NOT NULL DEFAULT '';
ALTER TABLE orgs ADD COLUMN IF NOT EXISTS updated_at TEXT NOT NULL DEFAULT '';
ALTER TABLE orgs ADD COLUMN IF NOT EXISTS deleted_at TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS global_role TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS slug TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS color TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS updated_at TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS deleted_at TEXT NOT NULL DEFAULT '';
UPDATE orgs SET slug = lower(regexp_replace(name, '[^a-zA-Z0-9]+', '-', 'g')) WHERE slug = '';
UPDATE projects SET slug = lower(regexp_replace(name, '[^a-zA-Z0-9]+', '-', 'g')) WHERE slug = '';
CREATE TABLE IF NOT EXISTS project_assignments (
  org_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  project_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (org_id, user_id, project_id)
);
CREATE TABLE IF NOT EXISTS audit_logs (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL DEFAULT '',
  details TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_org ON audit_logs(org_id, created_at);
CREATE INDEX IF NOT EXISTS idx_members_user ON members(user_id);`,
	// migration 22 — Deployment Spec First (compose por deployment)
	`
ALTER TABLE deployments ADD COLUMN IF NOT EXISTS compose_yaml TEXT NOT NULL DEFAULT '';
ALTER TABLE deployments ADD COLUMN IF NOT EXISTS deploy_spec TEXT NOT NULL DEFAULT '';
ALTER TABLE deployments ADD COLUMN IF NOT EXISTS compose_hash TEXT NOT NULL DEFAULT '';`,
}

const migrationLockKey = 72811042

func Migrate(d *SQL) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	conn, err := d.DB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(`+strconv.Itoa(migrationLockKey)+`)`); err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	defer conn.ExecContext(ctx, `SELECT pg_advisory_unlock(`+strconv.Itoa(migrationLockKey)+`)`)

	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	var current int
	if err := conn.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&current); err != nil {
		return err
	}
	for i := current; i < len(migrations); i++ {
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations(version) VALUES($1)`, i+1); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		log.Printf("[db] migration %d/%d aplicada.", i+1, len(migrations))
	}
	return nil
}

func Version(d *SQL) (int, error) {
	var v int
	err := d.QueryRow(`SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&v)
	return v, err
}
