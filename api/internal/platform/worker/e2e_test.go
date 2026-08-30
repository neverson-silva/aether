package worker

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"aether/internal/modules/apps/domain"
	appsInfra "aether/internal/modules/apps/infra"
	deployApp "aether/internal/modules/deployments/application"
	deployInfra "aether/internal/modules/deployments/infra"
	"aether/internal/platform/database"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	port := 5432
	if v := os.Getenv("AETHER_TEST_DATABASE_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			port = n
		}
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
	cfg := database.Config{
		Host: "127.0.0.1", Port: port, Name: "aether_api_test_worker",
		User: user, Password: password, SSLMode: "disable",
		PoolMax: 8, ConnectTimeout: 5,
	}
	if err := database.EnsureDatabase(ctx, cfg); err != nil {
		t.Fatalf("criar banco de teste: %v", err)
	}
	pool, err := database.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("postgres de teste indisponível: %v", err)
	}
	if err := database.Migrate(ctx, pool, "../../../db/migrations"); err != nil {
		pool.Close()
		t.Fatalf("migrate: %v", err)
	}
	for _, q := range []string{
		"TRUNCATE deployments, app_env, env_variables, apps, environments, projects RESTART IDENTITY CASCADE",
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

func TestWorkerDeploysRealContainer(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	appsStore := appsInfra.NewStore(pool)
	deployStore := deployInfra.NewStore(pool)
	defer appsStore.Close()
	defer deployStore.Close()

	orgID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, orgID, "Worker Org", "worker-org"); err != nil {
		t.Fatalf("criar org: %v", err)
	}
	project, err := appsStore.CreateProject(ctx, orgID, "WorkerProj", "worker-proj", "", "")
	if err != nil {
		t.Fatalf("criar project: %v", err)
	}
	app, err := appsStore.CreateApp(ctx, &domain.App{
		OrgID: orgID, ProjectID: project.ID, Name: "web", SourceType: "image",
		Image: "nginx:alpine", Port: 38080,
		HealthCheck: domain.HealthCheck{Enabled: true, Path: "/", TimeoutMS: 30000, Retries: 20},
	})
	if err != nil {
		t.Fatalf("criar app: %v", err)
	}

	svc := &deployApp.Deployments{Store: deployStore, Apps: appsStore}
	dep, err := svc.Deploy(ctx, app.ID, orgID, deployApp.DeployOpts{Trigger: "test"})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}

	runtime, err := NewDockerRuntime("")
	if err != nil {
		t.Fatalf("create Docker runtime: %v", err)
	}
	defer runtime.Close()
	w := &Worker{Store: deployStore, Runtime: runtime}
	w.deploy(ctx, dep)

	got, err := deployStore.GetDeployment(ctx, dep.ID)
	if err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	if got.Status != "ready" {
		t.Fatalf("deployment deveria estar ready, got %s (error=%q)", got.Status, got.Error)
	}
	if got.ContainerID == "" {
		t.Fatalf("container id deveria estar persistido")
	}

	time.Sleep(time.Second)
	_ = runtime.Remove(ctx, got.ContainerID)
	t.Logf("container real deployado: %s", got.ContainerID)
}
