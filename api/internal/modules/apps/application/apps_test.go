package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"aether/internal/modules/apps/domain"
	"aether/internal/modules/apps/infra"
)

type env struct {
	ctx   context.Context
	svc   *Apps
	pool  *pgxpool.Pool
	orgID uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	secrets, err := infra.NewSecretCipher([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	svc := &Apps{Store: store, Secrets: secrets}
	ctx := context.Background()
	orgID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, orgID, "Test Org", "test-org"); err != nil {
		store.Close()
		t.Fatalf("criar org: %v", err)
	}
	e := &env{ctx: ctx, svc: svc, pool: pool, orgID: orgID}
	t.Cleanup(func() { _ = store.Close() })
	return e
}

func TestProjectLifecycle(t *testing.T) {
	e := newEnv(t)
	p, err := e.svc.CreateProject(e.ctx, e.orgID, "Meu Projeto", "desc", "#00ff00")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if p.Slug != "meu-projeto" || p.OrgID != e.orgID {
		t.Fatalf("project inesperado: %+v", p)
	}

	got, err := e.svc.GetProject(e.ctx, p.ID, e.orgID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Meu Projeto" {
		t.Fatalf("nome divergente")
	}

	list, err := e.svc.ListProjects(e.ctx, e.orgID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}

	if _, err := e.svc.GetProject(e.ctx, uuid.New(), e.orgID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}

	updated, err := e.svc.UpdateProject(e.ctx, p.ID, e.orgID, "Novo Nome", "nova", "#0000ff")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Novo Nome" || updated.Slug != "novo-nome" {
		t.Fatalf("update divergente: %+v", updated)
	}

	if err := e.svc.DeleteProject(e.ctx, p.ID, e.orgID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := e.svc.GetProject(e.ctx, p.ID, e.orgID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("projeto deletado ainda visível: %v", err)
	}
}

func TestProjectValidation(t *testing.T) {
	e := newEnv(t)
	if _, err := e.svc.CreateProject(e.ctx, e.orgID, "", "", ""); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("nome vazio deveria falhar: %v", err)
	}
}

func TestEnvironmentLifecycle(t *testing.T) {
	e := newEnv(t)
	p, _ := e.svc.CreateProject(e.ctx, e.orgID, "Proj", "", "")
	envs, err := e.svc.ListEnvironments(e.ctx, p.ID)
	if err != nil || len(envs) != 1 {
		t.Fatalf("projeto deveria criar a env production por padrão: %v %d", err, len(envs))
	}
	env := envs[0]
	if env.Slug != "production" || !env.IsDefault {
		t.Fatalf("env inesperado: %+v", env)
	}
	if _, err := e.svc.CreateEnvironment(e.ctx, p.ID, "production", "", "", false); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("slug duplicado deveria falhar: %v", err)
	}
	if err := e.svc.DeleteEnvironment(e.ctx, env.ID, p.ID); err != nil {
		t.Fatalf("delete env: %v", err)
	}
}

func TestAppLifecycle(t *testing.T) {
	e := newEnv(t)
	p, _ := e.svc.CreateProject(e.ctx, e.orgID, "AppProj", "", "")
	env, _ := e.svc.CreateEnvironment(e.ctx, p.ID, "prod", "", "", true)

	created, err := e.svc.CreateApp(e.ctx, e.orgID, p.ID, &domain.App{
		Name: "api", SourceType: "image", Image: "nginx:alpine", Port: 80,
		MemMB: 256, HealthCheck: domain.HealthCheck{Enabled: true, Path: "/health", IntervalMS: 3000, TimeoutMS: 1000, Retries: 2},
		EnvironmentID: &env.ID,
	})
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	if created.Port != 80 || !created.HealthCheck.Enabled {
		t.Fatalf("app inesperado: %+v", created)
	}

	got, err := e.svc.GetApp(e.ctx, created.ID, e.orgID)
	if err != nil || got.Name != "api" {
		t.Fatalf("get app: %v", err)
	}

	if _, err := e.svc.GetApp(e.ctx, created.ID, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("app deveria ser isolado por org: %v", err)
	}

	updated, err := e.svc.UpdateApp(e.ctx, created.ID, e.orgID, &domain.App{
		Name: "api", SourceType: "image", Image: "nginx:1.25", Port: 8080,
	})
	if err != nil {
		t.Fatalf("update app: %v", err)
	}
	if updated.Image != "nginx:1.25" || updated.Port != 8080 {
		t.Fatalf("update divergente: %+v", updated)
	}

	appList, err := e.svc.ListApps(e.ctx, e.orgID, &p.ID)
	if err != nil || len(appList) != 1 {
		t.Fatalf("list apps: %v %d", err, len(appList))
	}

	if err := e.svc.DeleteApp(e.ctx, created.ID, e.orgID); err != nil {
		t.Fatalf("delete app: %v", err)
	}
	if _, err := e.svc.GetApp(e.ctx, created.ID, e.orgID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("app deveria ter sido removido: %v", err)
	}
}

func TestAppValidation(t *testing.T) {
	e := newEnv(t)
	p, _ := e.svc.CreateProject(e.ctx, e.orgID, "ValProj", "", "")
	_, err := e.svc.CreateApp(e.ctx, e.orgID, p.ID, &domain.App{Name: "api", SourceType: "image", Image: ""})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("app sem imagem deveria falhar: %v", err)
	}
	_, err = e.svc.CreateApp(e.ctx, e.orgID, p.ID, &domain.App{Name: "Nome Inválido!", SourceType: "image", Image: "x"})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("nome inválido deveria falhar: %v", err)
	}
	_, err = e.svc.CreateApp(e.ctx, e.orgID, p.ID, &domain.App{Name: "api", SourceType: "image", Image: "x", Port: 0})
	if err != nil {
		t.Fatalf("port 0 deveria ser defaultado: %v", err)
	}
	_, err = e.svc.CreateApp(e.ctx, e.orgID, p.ID, &domain.App{Name: "api", SourceType: "image", Image: "x", Port: 70000})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("porta inválida deveria falhar: %v", err)
	}
}

func TestEnvVars(t *testing.T) {
	e := newEnv(t)
	p, _ := e.svc.CreateProject(e.ctx, e.orgID, "EnvProj", "", "")
	app, err := e.svc.CreateApp(e.ctx, e.orgID, p.ID, &domain.App{Name: "api", SourceType: "image", Image: "nginx", Port: 80})
	if err != nil {
		t.Fatalf("create app: %v", err)
	}

	if err := e.svc.SetEnv(e.ctx, app.ID, e.orgID, "DATABASE_URL", "postgres://x", false); err != nil {
		t.Fatalf("set env: %v", err)
	}
	if err := e.svc.SetEnv(e.ctx, app.ID, e.orgID, "API_KEY", "supersecret", true); err != nil {
		t.Fatalf("set secret: %v", err)
	}
	if err := e.svc.SetEnv(e.ctx, app.ID, e.orgID, "DATABASE_URL", "postgres://y", false); err != nil {
		t.Fatalf("upsert env: %v", err)
	}

	vars, err := e.svc.ListEnv(e.ctx, app.ID, e.orgID)
	if err != nil {
		t.Fatalf("list env: %v", err)
	}
	if len(vars) != 2 {
		t.Fatalf("esperava 2 vars, got %d", len(vars))
	}
	for _, v := range vars {
		if v.Name == "API_KEY" && v.Value == "supersecret" {
			t.Fatalf("secret vazado no list")
		}
	}

	if err := e.svc.SetEnv(e.ctx, app.ID, e.orgID, "CHAVE=IGUAL", "x", false); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("nome com = deveria falhar: %v", err)
	}

	if err := e.svc.DeleteEnv(e.ctx, app.ID, e.orgID, "API_KEY"); err != nil {
		t.Fatalf("delete env: %v", err)
	}
	vars, _ = e.svc.ListEnv(e.ctx, app.ID, e.orgID)
	if len(vars) != 1 {
		t.Fatalf("esperava 1 var após delete, got %d", len(vars))
	}
}

func TestIsolationAcrossOrgs(t *testing.T) {
	e := newEnv(t)
	other := uuid.New()
	if _, err := e.pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, other, "Other Org", "other-org"); err != nil {
		t.Fatalf("criar org B: %v", err)
	}
	p1, err := e.svc.CreateProject(e.ctx, e.orgID, "Org A", "", "")
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	p2, err := e.svc.CreateProject(e.ctx, other, "Org B", "", "")
	if err != nil {
		t.Fatalf("create B: %v", err)
	}
	if p1.OrgID == p2.OrgID {
		t.Fatalf("orgs deveriam ser diferentes")
	}
	list, err := e.svc.ListProjects(e.ctx, e.orgID)
	if err != nil || len(list) != 1 {
		t.Fatalf("org A deveria ver só 1 projeto: %v %d", err, len(list))
	}
}

func TestAppDefaults(t *testing.T) {
	e := newEnv(t)
	p, _ := e.svc.CreateProject(e.ctx, e.orgID, "DefProj", "", "")
	created, err := e.svc.CreateApp(e.ctx, e.orgID, p.ID, &domain.App{
		Name: "defaults", SourceType: "image", Image: "nginx",
	})
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	if created.Port != 80 {
		t.Fatalf("port deveria ser 80: %d", created.Port)
	}
	if created.Dockerfile != "Dockerfile" {
		t.Fatalf("dockerfile deveria ser Dockerfile: %q", created.Dockerfile)
	}
	updated, err := e.svc.UpdateApp(e.ctx, created.ID, e.orgID, &domain.App{Name: "defaults"})
	if err != nil {
		t.Fatalf("update app: %v", err)
	}
	if updated.Port != 80 {
		t.Fatalf("update preservou port? %d", updated.Port)
	}
	resized, err := e.svc.UpdateApp(e.ctx, created.ID, e.orgID, &domain.App{CPUs: "1.0", MemMB: 1024})
	if err != nil {
		t.Fatalf("update resources: %v", err)
	}
	if resized.CPUs != "1.0" || resized.MemMB != 1024 {
		t.Fatalf("resources não aplicadas: %+v", resized)
	}
	if resized.Port != 80 {
		t.Fatalf("update de resources deveria preservar port: %d", resized.Port)
	}
}

func TestSetWebhookEncryptsSecret(t *testing.T) {
	e := newEnv(t)
	p, _ := e.svc.CreateProject(e.ctx, e.orgID, "WhkProj", "", "")
	app, err := e.svc.CreateApp(e.ctx, e.orgID, p.ID, &domain.App{Name: "web", SourceType: "image", Image: "nginx", Port: 80})
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	if err := e.svc.SetWebhook(e.ctx, app.ID, e.orgID, "super-secret"); err != nil {
		t.Fatalf("set webhook: %v", err)
	}
	got, err := e.svc.GetApp(e.ctx, app.ID, e.orgID)
	if err != nil {
		t.Fatalf("get app: %v", err)
	}
	if got.WebhookSecret == "super-secret" {
		t.Fatalf("secret deveria estar criptografado no banco")
	}
	if got.WebhookSecret == "" {
		t.Fatalf("secret não foi salvo")
	}
	dec, err := e.svc.Secrets.Decrypt(got.WebhookSecret)
	if err != nil || dec != "super-secret" {
		t.Fatalf("decrypt: %v %q", err, dec)
	}
	if err := e.svc.SetWebhook(e.ctx, app.ID, e.orgID, ""); err != nil {
		t.Fatalf("limpar webhook: %v", err)
	}
	cleared, _ := e.svc.GetApp(e.ctx, app.ID, e.orgID)
	if cleared.WebhookSecret != "" {
		t.Fatalf("webhook limpo deveria ficar vazio")
	}
}
