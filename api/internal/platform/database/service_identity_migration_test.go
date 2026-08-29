package database

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestServiceIdentityMigrationBackfillsLegacyRecords(t *testing.T) {
	ctx := context.Background()
	pool := serviceIdentityTestPool(t, ctx)
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	for _, table := range []string{"apps", "compose_apps", "databases"} {
		if _, err := tx.Exec(ctx, `ALTER TABLE `+table+` ALTER COLUMN service_id DROP NOT NULL`); err != nil {
			t.Fatal(err)
		}
	}

	orgID := uuid.New()
	projectID := uuid.New()
	environmentID := uuid.New()
	appID := uuid.New()
	composeID := uuid.New()
	databaseID := uuid.New()
	deploymentID := uuid.New()
	composeDeploymentID := uuid.New()
	databaseDeploymentID := uuid.New()
	domainID := uuid.New()
	composeDomainID := uuid.New()
	databaseDomainID := uuid.New()
	appBackupID := uuid.New()
	databaseBackupID := uuid.New()
	alertRuleID := uuid.New()
	alertEventID := uuid.New()
	snapshotID := uuid.New()
	scheduleID := uuid.New()
	previewID := uuid.New()
	cronID := uuid.New()
	volumeID := uuid.New()
	pipelineID := uuid.New()
	repairDeploymentID := uuid.New()
	repairDomainID := uuid.New()
	appEnvName := "LEGACY_TEST_KEY"

	if _, err := tx.Exec(ctx, `INSERT INTO orgs (id, name, slug) VALUES ($1, $2, $3)`, orgID, "Legacy Identity Test", "legacy-identity-test"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO projects (id, org_id, name, slug) VALUES ($1, $2, $3, $4)`, projectID, orgID, "Legacy Identity Project", "legacy-identity-project"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO environments (id, project_id, name, slug, is_default) VALUES ($1, $2, $3, $4, true)`, environmentID, projectID, "Production", "production"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `ALTER TABLE apps DISABLE TRIGGER USER`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `ALTER TABLE compose_apps DISABLE TRIGGER USER`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `ALTER TABLE databases DISABLE TRIGGER USER`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO apps (id, org_id, project_id, environment_id, name) VALUES ($1, $2, $3, $4, $5)`, appID, orgID, projectID, environmentID, "legacy-app"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO compose_apps (id, org_id, project_id, name, compose, status) VALUES ($1, $2, $3, $4, $5, 'stopped')`, composeID, orgID, projectID, "legacy-compose", "services:\n  app:\n    image: nginx:alpine"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO databases (id, org_id, project_id, name, engine, version, db_name, db_user, pass_enc, status) VALUES ($1, $2, $3, $4, 'postgres', '16', 'legacy', 'legacy', 'encrypted', 'creating')`, databaseID, orgID, projectID, "legacy-database"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `ALTER TABLE apps ENABLE TRIGGER USER`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `ALTER TABLE compose_apps ENABLE TRIGGER USER`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `ALTER TABLE databases ENABLE TRIGGER USER`); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"apps", "compose_apps", "databases"} {
		if _, err := tx.Exec(ctx, `DROP TRIGGER IF EXISTS `+table+`_service_spec_kind ON `+table); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO deployments (id, app_id, number, status) VALUES ($1, $2, 1, 'queued')`, deploymentID, appID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO deployments (id, app_id, number, status) VALUES ($1, $2, 1, 'queued')`, composeDeploymentID, composeID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO deployments (id, app_id, number, status) VALUES ($1, $2, 1, 'queued')`, databaseDeploymentID, databaseID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO domains (id, app_id, service_type, host) VALUES ($1, $2, 'app', 'legacy.example.test')`, domainID, appID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO domains (id, app_id, service_type, host) VALUES ($1, $2, 'compose', 'legacy-compose.example.test')`, composeDomainID, composeID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO domains (id, app_id, service_type, host) VALUES ($1, $2, 'db', 'legacy-database.example.test')`, databaseDomainID, databaseID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO backups (id, org_id, app_id, path) VALUES ($1, $2, $3, 'legacy-app-backup')`, appBackupID, orgID, appID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO backups (id, org_id, database_id, path) VALUES ($1, $2, $3, 'legacy-database-backup')`, databaseBackupID, orgID, databaseID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO app_env (app_id, name, value, secret) VALUES ($1, $2, 'legacy-value', false)`, appID, appEnvName); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO alert_rules (id, org_id, name, metric, threshold, target_app) VALUES ($1, $2, 'legacy-alert', 'cpu', 80, $3)`, alertRuleID, orgID, appID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO alert_events (id, org_id, app_id, severity, message, metric) VALUES ($1, $2, $3, 'warning', 'legacy-event', 'cpu')`, alertEventID, orgID, appID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO snapshots (id, org_id, app_id, volume, name) VALUES ($1, $2, $3, 'legacy-volume', 'legacy-snapshot')`, snapshotID, orgID, appID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO snapshot_schedules (id, org_id, app_id, volume, cron) VALUES ($1, $2, $3, 'legacy-volume', '* * * * *')`, scheduleID, orgID, appID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO pipelines (id, org_id, app_id, name, trigger, stages, enabled) VALUES ($1, $2, $3, 'legacy-pipeline', 'manual', '[]', true)`, pipelineID, orgID, appID); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"apps", "compose_apps", "databases"} {
		if _, err := tx.Exec(ctx, `ALTER TABLE `+table+` DROP CONSTRAINT IF EXISTS `+table+`_service_id_fkey`); err != nil {
			t.Fatal(err)
		}
	}

	migrationNames := []string{
		"0028_services_catalog.sql",
		"0029_service_resource_identity.sql",
		"0030_service_identity_triggers.sql",
		"0031_service_resource_identity_expansion.sql",
		"0032_service_identity_sync.sql",
		"0033_service_source_identity.sql",
		"0034_service_environment_identity.sql",
		"0035_backup_service_identity_triggers.sql",
		"0036_app_resource_identity_triggers.sql",
		"0037_database_environment_identity.sql",
		"0038_deployment_service_identity.sql",
		"0039_domain_service_identity.sql",
		"0040_observability_snapshot_service_identity.sql",
		"0041_service_scope_sync.sql",
		"0042_alert_service_identity_rollout.sql",
		"0043_alert_trigger_split.sql",
		"0044_service_resource_completion.sql",
		"0045_compose_pending_status.sql",
		"0046_compose_deploying_status.sql",
		"0047_generic_service_resource_identity.sql",
		"0048_service_status_consistency.sql",
		"0049_snapshot_service_identity.sql",
		"0050_service_spec_identity_required.sql",
		"0051_service_resource_backfill_repair.sql",
		"0052_generic_resources_service_first.sql",
		"0053_service_spec_kind_integrity.sql",
	}
	for _, name := range migrationNames {
		if name == "0051_service_resource_backfill_repair.sql" {
			if _, err := tx.Exec(ctx, `ALTER TABLE deployments DISABLE TRIGGER USER`); err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(ctx, `ALTER TABLE domains DISABLE TRIGGER USER`); err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO deployments (id, app_id, number, status) VALUES ($1, $2, 99, 'queued')`, repairDeploymentID, appID); err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO domains (id, app_id, service_type, host) VALUES ($1, $2, 'app', 'repair.example.test')`, repairDomainID, appID); err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(ctx, `ALTER TABLE deployments ENABLE TRIGGER USER`); err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(ctx, `ALTER TABLE domains ENABLE TRIGGER USER`); err != nil {
				t.Fatal(err)
			}
		}
		raw, readErr := os.ReadFile("../../../db/migrations/" + name)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, execErr := tx.Exec(ctx, string(raw)); execErr != nil {
			t.Fatalf("execute %s: %v", name, execErr)
		}
	}
	for _, name := range migrationNames {
		raw, readErr := os.ReadFile("../../../db/migrations/" + name)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, execErr := tx.Exec(ctx, string(raw)); execErr != nil {
			t.Fatalf("execute %s twice: %v", name, execErr)
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO previews (id, app_id, branch) VALUES ($1, $2, 'legacy-preview')`, previewID, appID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO cron_jobs (id, app_id, name, schedule, command) VALUES ($1, $2, 'legacy-cron', '* * * * *', 'true')`, cronID, appID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO app_volumes (id, app_id, name, mount_path) VALUES ($1, $2, 'legacy-volume', '/data')`, volumeID, appID); err != nil {
		t.Fatal(err)
	}
	composeRuntimeDeploymentID := uuid.New()
	databaseRuntimeDeploymentID := uuid.New()
	if _, err := tx.Exec(ctx, `INSERT INTO deployments (id, app_id, number, status) VALUES ($1, $2, 2, 'queued')`, composeRuntimeDeploymentID, composeID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO deployments (id, app_id, number, status) VALUES ($1, $2, 2, 'queued')`, databaseRuntimeDeploymentID, databaseID); err != nil {
		t.Fatal(err)
	}
	var composeDeploymentServiceID, databaseDeploymentServiceID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT service_id FROM deployments WHERE id = $1`, composeRuntimeDeploymentID).Scan(&composeDeploymentServiceID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `SELECT service_id FROM deployments WHERE id = $1`, databaseRuntimeDeploymentID).Scan(&databaseDeploymentServiceID); err != nil {
		t.Fatal(err)
	}
	if composeDeploymentServiceID == uuid.Nil || databaseDeploymentServiceID == uuid.Nil {
		t.Fatalf("generic service resource identity was not assigned: compose=%s database=%s", composeDeploymentServiceID, databaseDeploymentServiceID)
	}

	assertServiceIdentity(t, ctx, tx, appID, "app")
	assertServiceIdentity(t, ctx, tx, composeID, "compose")
	assertServiceIdentity(t, ctx, tx, databaseID, "database")
	var serviceID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT service_id FROM apps WHERE id = $1`, appID).Scan(&serviceID); err != nil {
		t.Fatal(err)
	}
	var deploymentServiceID, domainServiceID, envServiceID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT service_id FROM deployments WHERE id = $1`, deploymentID).Scan(&deploymentServiceID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `SELECT service_id FROM domains WHERE id = $1`, domainID).Scan(&domainServiceID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `SELECT service_id FROM app_env WHERE app_id = $1 AND name = $2`, appID, appEnvName).Scan(&envServiceID); err != nil {
		t.Fatal(err)
	}
	if deploymentServiceID != serviceID || domainServiceID != serviceID || envServiceID != serviceID {
		t.Fatalf("legacy resources were not linked to the app service: deployment=%s domain=%s env=%s service=%s", deploymentServiceID, domainServiceID, envServiceID, serviceID)
	}
	for _, resource := range []struct {
		table string
		id    uuid.UUID
	}{
		{table: "deployments", id: repairDeploymentID},
		{table: "domains", id: repairDomainID},
	} {
		var resourceServiceID uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT service_id FROM `+resource.table+` WHERE id = $1`, resource.id).Scan(&resourceServiceID); err != nil {
			t.Fatal(err)
		}
		if resourceServiceID != serviceID {
			t.Fatalf("resource backfill repair did not link %s: %s != %s", resource.table, resourceServiceID, serviceID)
		}
	}
	var snapshotServiceID, scheduleServiceID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT service_id FROM snapshots WHERE id = $1`, snapshotID).Scan(&snapshotServiceID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `SELECT service_id FROM snapshot_schedules WHERE id = $1`, scheduleID).Scan(&scheduleServiceID); err != nil {
		t.Fatal(err)
	}
	if snapshotServiceID != serviceID || scheduleServiceID != serviceID {
		t.Fatalf("legacy snapshot resources were not linked to the app service: snapshot=%s schedule=%s service=%s", snapshotServiceID, scheduleServiceID, serviceID)
	}
	var pipelineServiceID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT service_id FROM pipelines WHERE id = $1`, pipelineID).Scan(&pipelineServiceID); err != nil {
		t.Fatal(err)
	}
	if pipelineServiceID != serviceID {
		t.Fatalf("legacy pipeline was not linked to the app service: pipeline=%s service=%s", pipelineServiceID, serviceID)
	}
	var composeServiceStatus, databaseServiceStatus string
	if err := tx.QueryRow(ctx, `SELECT status FROM services WHERE id = (SELECT service_id FROM compose_apps WHERE id = $1)`, composeID).Scan(&composeServiceStatus); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `SELECT status FROM services WHERE id = (SELECT service_id FROM databases WHERE id = $1)`, databaseID).Scan(&databaseServiceStatus); err != nil {
		t.Fatal(err)
	}
	if composeServiceStatus != "stopped" || databaseServiceStatus != "pending" {
		t.Fatalf("legacy service statuses were not normalized: compose=%s database=%s", composeServiceStatus, databaseServiceStatus)
	}
	for _, item := range []struct {
		name       string
		resourceID uuid.UUID
		specID     uuid.UUID
		table      string
		kind       string
	}{
		{name: "compose deployment", resourceID: composeDeploymentID, specID: composeID, table: "deployments", kind: "compose"},
		{name: "database deployment", resourceID: databaseDeploymentID, specID: databaseID, table: "deployments", kind: "database"},
		{name: "compose domain", resourceID: composeDomainID, specID: composeID, table: "domains", kind: "compose"},
		{name: "database domain", resourceID: databaseDomainID, specID: databaseID, table: "domains", kind: "database"},
	} {
		var resourceServiceID, expectedServiceID uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT service_id FROM `+item.table+` WHERE id = $1`, item.resourceID).Scan(&resourceServiceID); err != nil {
			t.Fatal(err)
		}
		specTable := "compose_apps"
		if item.kind == "database" {
			specTable = "databases"
		}
		if err := tx.QueryRow(ctx, `SELECT service_id FROM `+specTable+` WHERE id = $1`, item.specID).Scan(&expectedServiceID); err != nil {
			t.Fatal(err)
		}
		if resourceServiceID != expectedServiceID {
			t.Fatalf("legacy %s was not linked to the service: %s != %s", item.name, resourceServiceID, expectedServiceID)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE compose_apps SET environment_id = $1 WHERE id = $2`, environmentID, composeID); err != nil {
		t.Fatal(err)
	}
	var composeEnvironmentID *uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT environment_id FROM services WHERE id = (SELECT service_id FROM compose_apps WHERE id = $1)`, composeID).Scan(&composeEnvironmentID); err != nil {
		t.Fatal(err)
	}
	if composeEnvironmentID == nil || *composeEnvironmentID != environmentID {
		t.Fatalf("compose service scope was not synchronized: %v != %s", composeEnvironmentID, environmentID)
	}
	for _, item := range []struct {
		name       string
		resourceID uuid.UUID
		expectedID uuid.UUID
	}{
		{name: "app backup", resourceID: appBackupID, expectedID: serviceID},
		{name: "database backup", resourceID: databaseBackupID},
	} {
		var resourceServiceID uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT service_id FROM backups WHERE id = $1`, item.resourceID).Scan(&resourceServiceID); err != nil {
			t.Fatal(err)
		}
		if item.name == "database backup" {
			if err := tx.QueryRow(ctx, `SELECT service_id FROM databases WHERE id = $1`, databaseID).Scan(&item.expectedID); err != nil {
				t.Fatal(err)
			}
		}
		if resourceServiceID != item.expectedID {
			t.Fatalf("legacy %s was not linked to the service: %s != %s", item.name, resourceServiceID, item.expectedID)
		}
	}
	for _, item := range []struct {
		name string
		id   uuid.UUID
	}{
		{name: "alert rule", id: alertRuleID},
		{name: "alert event", id: alertEventID},
		{name: "snapshot", id: snapshotID},
		{name: "snapshot schedule", id: scheduleID},
	} {
		var resourceServiceID uuid.UUID
		var table string
		switch item.name {
		case "alert rule":
			table = "alert_rules"
		case "alert event":
			table = "alert_events"
		case "snapshot":
			table = "snapshots"
		default:
			table = "snapshot_schedules"
		}
		if err := tx.QueryRow(ctx, `SELECT service_id FROM `+table+` WHERE id = $1`, item.id).Scan(&resourceServiceID); err != nil {
			t.Fatal(err)
		}
		if resourceServiceID != serviceID {
			t.Fatalf("legacy %s was not linked to the app service: %s != %s", item.name, resourceServiceID, serviceID)
		}
	}
	for _, item := range []struct {
		table string
		id    uuid.UUID
	}{
		{table: "previews", id: previewID},
		{table: "cron_jobs", id: cronID},
		{table: "app_volumes", id: volumeID},
	} {
		var resourceServiceID uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT service_id FROM `+item.table+` WHERE id = $1`, item.id).Scan(&resourceServiceID); err != nil {
			t.Fatal(err)
		}
		if resourceServiceID != serviceID {
			t.Fatalf("new %s was not linked to the app service: %s != %s", item.table, resourceServiceID, serviceID)
		}
	}
	mismatchedServiceID := uuid.New()
	if _, err := tx.Exec(ctx, `INSERT INTO services (id, org_id, project_id, name, kind) VALUES ($1, $2, $3, 'mismatched-service', 'database')`, mismatchedServiceID, orgID, projectID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE compose_apps SET service_id = $1 WHERE id = $2`, mismatchedServiceID, composeID); err == nil {
		t.Fatal("expected service specification kind constraint to reject a mismatched service")
	}
}

func assertServiceIdentity(t *testing.T, ctx context.Context, tx pgx.Tx, specID uuid.UUID, kind string) {
	t.Helper()
	var serviceID uuid.UUID
	query := `SELECT id FROM apps WHERE service_id = $1`
	if kind == "compose" {
		query = `SELECT service_id FROM compose_apps WHERE id = $1`
	}
	if kind == "database" {
		query = `SELECT service_id FROM databases WHERE id = $1`
	}
	if kind == "app" {
		query = `SELECT service_id FROM apps WHERE id = $1`
	}
	if err := tx.QueryRow(ctx, query, specID).Scan(&serviceID); err != nil {
		t.Fatal(err)
	}
	var serviceKind string
	if err := tx.QueryRow(ctx, `SELECT kind FROM services WHERE id = $1`, serviceID).Scan(&serviceKind); err != nil {
		t.Fatal(err)
	}
	if serviceKind != kind || serviceID == uuid.Nil {
		t.Fatalf("unexpected %s service identity: service=%s kind=%s spec=%s", kind, serviceID, serviceKind, specID)
	}
}

func serviceIdentityTestPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	port := 5432
	if value := os.Getenv("AETHER_TEST_DATABASE_PORT"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err == nil {
			port = parsed
		}
	}
	user := os.Getenv("AETHER_TEST_DATABASE_USER")
	password := os.Getenv("AETHER_TEST_DATABASE_PASSWORD")
	if user == "" && port == 5433 {
		user, password = "postgres", "postgres"
	}
	name := os.Getenv("AETHER_TEST_DATABASE_NAME")
	if name == "" {
		name = "aether_service_identity_migration_test"
	}
	cfg := Config{Host: "127.0.0.1", Port: port, Name: name, User: user, Password: password, SSLMode: "disable", PoolMax: 4, ConnectTimeout: 5}
	if err := EnsureDatabase(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	pool, err := Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, pool, "../../../db/migrations"); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	return pool
}
