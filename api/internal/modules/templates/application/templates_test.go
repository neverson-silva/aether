package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	appsInfra "aether/internal/modules/apps/infra"
	"aether/internal/modules/templates/domain"
	"aether/internal/modules/templates/infra"
	composeengine "aether/internal/platform/compose"
)

type env struct {
	ctx   context.Context
	pool  *pgxpool.Pool
	svc   *Templates
	orgID uuid.UUID
	proj  uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	appsStore := appsInfra.NewStore(pool)
	e := &env{ctx: context.Background(), pool: pool, svc: &Templates{Store: store, Apps: appsStore}, orgID: uuid.New()}
	t.Cleanup(func() {
		_ = store.Close()
		_ = appsStore.Close()
	})
	if _, err := pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, e.orgID, "Tpl Org", "tpl-org"); err != nil {
		t.Fatalf("criar org: %v", err)
	}
	project, err := appsStore.CreateProject(e.ctx, e.orgID, "TplProj", "tpl-proj", "", "")
	if err != nil {
		t.Fatalf("criar project: %v", err)
	}
	e.proj = project.ID
	return e
}

func seedTemplates(_ *testing.T, e *env) {
	e.svc.Catalog = testCatalog{templates: []domain.Template{
		{ID: uuid.New(), Name: "nginx", Category: "web", Definition: `{"services":[{"name":"web","image":"nginx:alpine","port":80}]}`, Tags: []string{"proxy", "http"}},
		{ID: uuid.New(), Name: "postgres", Category: "database", Definition: `{"services":[{"name":"db","image":"postgres:16"}]}`, Tags: []string{"db"}},
	}}
}

type testCatalog struct {
	templates []domain.Template
}

func (c testCatalog) List(context.Context) ([]domain.Template, error) {
	return c.templates, nil
}

func (c testCatalog) Get(_ context.Context, id uuid.UUID) (*domain.Template, error) {
	for _, template := range c.templates {
		if template.ID == id {
			copy := template
			return &copy, nil
		}
	}
	return nil, domain.ErrNotFound
}

func TestTemplateListAndFilters(t *testing.T) {
	e := newEnv(t)
	seedTemplates(t, e)
	list, err := e.svc.List(e.ctx, domain.Filter{})
	if err != nil || len(list) != 2 {
		t.Fatalf("list: %v %d", err, len(list))
	}

	web, err := e.svc.List(e.ctx, domain.Filter{Category: "web"})
	if err != nil || len(web) != 1 || web[0].Name != "nginx" {
		t.Fatalf("filtro categoria: %v %d", err, len(web))
	}

	search, err := e.svc.List(e.ctx, domain.Filter{Search: "postgres"})
	if err != nil || len(search) != 1 {
		t.Fatalf("filtro busca: %v %d", err, len(search))
	}
}

func TestTemplateInstall(t *testing.T) {
	e := newEnv(t)
	seedTemplates(t, e)
	_, err := e.svc.Install(e.ctx, uuid.New(), e.orgID, e.proj, "", nil)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("template inexistente deveria falhar: %v", err)
	}

	web, _ := e.svc.List(e.ctx, domain.Filter{Category: "web"})
	installed, err := e.svc.Install(e.ctx, web[0].ID, e.orgID, e.proj, "myweb", nil)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if installed.Installs != 1 || installed.ComposeYAML == "" {
		t.Fatalf("install inesperado: installs=%d compose=%q", installed.Installs, installed.ComposeYAML)
	}
	if !contains(installed.ComposeYAML, "nginx:alpine") {
		t.Fatalf("compose deveria conter a imagem: %s", installed.ComposeYAML)
	}

	apps, err := e.svc.ListCompose(e.ctx, e.orgID)
	if err != nil || len(apps) != 1 || apps[0].Name != "myweb" {
		t.Fatalf("compose apps: %v %d", err, len(apps))
	}
	if apps[0].EnvironmentID == nil {
		t.Fatal("template service must use the project default environment")
	}

	if err := e.svc.DeleteCompose(e.ctx, apps[0].ID, e.orgID); err != nil {
		t.Fatalf("delete compose: %v", err)
	}
}

func TestTemplateInstallOverride(t *testing.T) {
	e := newEnv(t)
	seedTemplates(t, e)
	web, _ := e.svc.List(e.ctx, domain.Filter{Category: "web"})
	installed, err := e.svc.Install(e.ctx, web[0].ID, e.orgID, e.proj, "web", map[string]string{"image": "nginx:1.25"})
	if err != nil {
		t.Fatalf("install override: %v", err)
	}
	if !contains(installed.ComposeYAML, "nginx:1.25") {
		t.Fatalf("override deveria trocar imagem: %s", installed.ComposeYAML)
	}
}

type staticCatalog struct {
	template domain.Template
}

func (c staticCatalog) List(context.Context) ([]domain.Template, error) {
	return []domain.Template{c.template}, nil
}

func (c staticCatalog) Get(context.Context, uuid.UUID) (*domain.Template, error) {
	template := c.template
	return &template, nil
}

func TestTemplateInstallNormalizesCatalogRestartPolicy(t *testing.T) {
	e := newEnv(t)
	template := domain.Template{
		ID:   uuid.New(),
		Name: "RustFS",
		ComposeYAML: `version: "3.8"
services:
  rustfs:
    image: rustfs/rustfs@sha256:41fe89380f4120a337790c02af192c3fe7bb55c3edc2e6e9357b487b47c6ab21
    volumes:
      - rustfs-data:/data
    restart: unless-stopped
volumes:
  rustfs-data: {}`,
	}
	e.svc.Catalog = staticCatalog{template: template}

	installed, err := e.svc.Install(e.ctx, template.ID, e.orgID, e.proj, "rustfs", nil)
	if err != nil {
		t.Fatalf("catalog install: %v", err)
	}
	if err := composeengine.ValidatePolicy(installed.ComposeYAML); err != nil {
		t.Fatalf("installed compose violates policy: %v", err)
	}
	if !contains(installed.ComposeYAML, "restart: no") {
		t.Fatalf("installed compose should use the managed restart policy: %s", installed.ComposeYAML)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
