package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	appsdomain "aether/internal/apps/domain"
	appsInfra "aether/internal/apps/infra"
	"aether/internal/orgs/domain"
	"aether/internal/orgs/infra"
)

type env struct {
	ctx    context.Context
	pool   *pgxpool.Pool
	svc    *Organizations
	owner  uuid.UUID
	orgID  uuid.UUID
	projID uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	appsStore := appsInfra.NewStore(pool)
	e := &env{ctx: context.Background(), pool: pool, svc: &Organizations{Store: store, Apps: appsStore}, owner: uuid.New()}
	t.Cleanup(func() {
		_ = store.Close()
		_ = appsStore.Close()
	})
	if _, err := pool.Exec(e.ctx, `INSERT INTO users (id, email, name, password_hash) VALUES ($1, $2, $3, 'x')`, e.owner, "owner@test.com", "Owner"); err != nil {
		t.Fatalf("criar user: %v", err)
	}
	org, err := e.svc.Create(e.ctx, e.owner, "Acme")
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	e.orgID = org.ID
	project, err := appsStore.CreateProject(e.ctx, e.orgID, "Proj", "proj", "", "")
	if err != nil {
		t.Fatalf("criar project: %v", err)
	}
	e.projID = project.ID
	return e
}

func TestOrgLifecycle(t *testing.T) {
	e := newEnv(t)
	orgs, err := e.svc.List(e.ctx, e.owner)
	if err != nil || len(orgs) != 1 {
		t.Fatalf("list: %v %d", err, len(orgs))
	}
	got, err := e.svc.Get(e.ctx, e.orgID, e.owner)
	if err != nil || got.Role != domain.RoleOwner {
		t.Fatalf("get: %v role=%s", err, got.Role)
	}
	updated, err := e.svc.Update(e.ctx, e.orgID, e.owner, "Acme Corp", "nova desc", nil, nil)
	if err != nil || updated.Name != "Acme Corp" {
		t.Fatalf("update: %v %+v", err, updated)
	}
	if err := e.svc.Delete(e.ctx, e.orgID, e.owner); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := e.svc.Get(e.ctx, e.orgID, e.owner); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("org deletada ainda existe: %v", err)
	}
}

func TestOrgPermission(t *testing.T) {
	e := newEnv(t)
	other := uuid.New()
	if _, err := e.svc.Get(e.ctx, e.orgID, other); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("nao-membro deveria falhar: %v", err)
	}
	if _, err := e.svc.Update(e.ctx, e.orgID, other, "x", "", nil, nil); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("nao-membro nao deveria atualizar: %v", err)
	}
}

func TestMemberManagement(t *testing.T) {
	e := newEnv(t)
	member := uuid.New()
	if _, err := e.pool.Exec(e.ctx, `INSERT INTO users (id, email, name, password_hash) VALUES ($1, $2, $3, 'x')`, member, "dev@test.com", "Dev"); err != nil {
		t.Fatalf("criar dev: %v", err)
	}
	if err := e.svc.Store.SetAssignment(e.ctx, e.orgID, member, e.projID); err != nil {
		t.Fatalf("assign: %v", err)
	}
	if err := e.svc.Store.SetAssignment(e.ctx, e.orgID, member, e.projID); err != nil {
		t.Fatalf("re-assign: %v", err)
	}
	assignments, err := e.svc.Assignments(e.ctx, e.orgID, e.owner)
	if err != nil || len(assignments) != 1 {
		t.Fatalf("assignments: %v %d", err, len(assignments))
	}
	if err := e.svc.Store.RemoveAssignment(e.ctx, e.orgID, member, e.projID); err != nil {
		t.Fatalf("remove assignment: %v", err)
	}
}

func TestAuditTrail(t *testing.T) {
	e := newEnv(t)
	events, err := e.svc.Audit(e.ctx, e.orgID, e.owner, 50)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if len(events) == 0 {
		t.Fatalf("audit deveria ter eventos (org.create)")
	}
}

func TestOrgExportImport(t *testing.T) {
	e := newEnv(t)
	appsStore := appsInfra.NewStore(e.pool)
	app, err := appsStore.CreateApp(e.ctx, &appsdomain.App{
		OrgID: e.orgID, ProjectID: e.projID, Name: "web", SourceType: "git",
		GitURL: "https://github.com/a/b", GitBranch: "main", Port: 80,
	})
	if err != nil {
		t.Fatalf("criar app: %v", err)
	}
	if err := appsStore.UpsertEnvVar(e.ctx, app.ID, "PORT", "80", false); err != nil {
		t.Fatalf("env var: %v", err)
	}
	data, err := e.svc.Export(e.ctx, e.orgID)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if !strings.Contains(string(data), "web") || !strings.Contains(string(data), "github.com/a/b") {
		t.Fatalf("export incompleto: %s", string(data))
	}

	otherOrg := uuid.New()
	if _, err := e.pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, otherOrg, "Other", "other"); err != nil {
		t.Fatalf("criar org2: %v", err)
	}
	if err := e.svc.Import(e.ctx, otherOrg, data); err != nil {
		t.Fatalf("import: %v", err)
	}
	projects, _ := appsStore.ListProjects(e.ctx, otherOrg)
	if len(projects) != 1 {
		t.Fatalf("import deveria criar projeto: %d", len(projects))
	}
}
