package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	appsdomain "aether/internal/apps/domain"
	appsInfra "aether/internal/apps/infra"
	"aether/internal/variables/domain"
	"aether/internal/variables/infra"
)

type env struct {
	ctx    context.Context
	svc    *Variables
	orgID  uuid.UUID
	projID uuid.UUID
	userID uuid.UUID
	envID  uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	appsStore := appsInfra.NewStore(pool)
	e := &env{ctx: context.Background(), svc: &Variables{Store: store, Apps: appsStore}, orgID: uuid.New()}
	t.Cleanup(func() {
		_ = store.Close()
		_ = appsStore.Close()
	})
	if _, err := pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, e.orgID, "Var Org", "var-org"); err != nil {
		t.Fatalf("criar org: %v", err)
	}
	e.userID = uuid.New()
	if _, err := pool.Exec(e.ctx, `INSERT INTO users (id, email, name, password_hash) VALUES ($1, $2, $3, 'x')`, e.userID, "var@test.com", "Var User"); err != nil {
		t.Fatalf("criar user: %v", err)
	}
	project, err := appsStore.CreateProject(e.ctx, e.orgID, "VarProj", "var-proj", "", "")
	if err != nil {
		t.Fatalf("criar project: %v", err)
	}
	e.projID = project.ID
	env, err := appsStore.CreateEnvironment(e.ctx, project.ID, "prod", "prod", "", "", true)
	if err != nil {
		t.Fatalf("criar env: %v", err)
	}
	e.envID = env.ID
	return e
}

func TestVariableLifecycle(t *testing.T) {
	e := newEnv(t)
	user := e.userID
	variable, err := e.svc.Set(e.ctx, e.projID, e.orgID, nil, user, "DATABASE_URL", "postgres://x", false)
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	if variable.Key != "DATABASE_URL" {
		t.Fatalf("variable inesperada: %+v", variable)
	}
	if _, err := e.svc.Set(e.ctx, e.projID, e.orgID, nil, user, "CHAVE=IGUAL", "x", false); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("key com = deveria falhar: %v", err)
	}

	list, err := e.svc.List(e.ctx, e.projID, e.orgID, nil)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}

	if err := e.svc.Delete(e.ctx, e.projID, e.orgID, nil, user, "DATABASE_URL"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, _ = e.svc.List(e.ctx, e.projID, e.orgID, nil)
	if len(list) != 0 {
		t.Fatalf("deveria estar vazio")
	}

	events, err := e.svc.Audit(e.ctx, e.projID, e.orgID, 50)
	if err != nil || len(events) == 0 {
		t.Fatalf("audit: %v %d", err, len(events))
	}
}

func TestVariableSecretMasked(t *testing.T) {
	e := newEnv(t)
	_, _ = e.svc.Set(e.ctx, e.projID, e.orgID, nil, e.userID, "API_KEY", "supersecret", true)
	list, _ := e.svc.List(e.ctx, e.projID, e.orgID, nil)
	if list[0].Value != "******" {
		t.Fatalf("secret deveria estar mascarada")
	}
}

func TestVariableImport(t *testing.T) {
	e := newEnv(t)
	err := e.svc.Import(e.ctx, e.projID, e.orgID, nil, e.userID, map[string]string{"A": "1", "B": "2"})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	list, _ := e.svc.List(e.ctx, e.projID, e.orgID, nil)
	if len(list) != 2 {
		t.Fatalf("esperava 2, got %d", len(list))
	}
}

func TestDefaultEnvironment(t *testing.T) {
	e := newEnv(t)
	other, err := e.svc.Apps.CreateEnvironment(e.ctx, e.projID, "dev", "dev", "", "", false)
	if err != nil {
		t.Fatalf("criar env dev: %v", err)
	}
	if err := e.svc.SetDefaultEnvironment(e.ctx, e.projID, e.orgID, other.ID); err != nil {
		t.Fatalf("set default: %v", err)
	}
	envs, _ := e.svc.Apps.ListEnvironments(e.ctx, e.projID)
	seen := map[uuid.UUID]bool{}
	for i := range envs {
		if envs[i].IsDefault {
			seen[envs[i].ID] = true
		}
	}
	if !seen[other.ID] || seen[e.envID] {
		t.Fatalf("default deveria ser o dev, prod desligado")
	}
}

func TestReplaceVariables(t *testing.T) {
	e := newEnv(t)
	saved, err := e.svc.Replace(e.ctx, e.projID, e.orgID, nil, e.userID, map[string]domain.VariableInput{
		"FOO":   {Value: "bar", Secret: false},
		"TOKEN": {Value: "abc", Secret: true},
	})
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	if saved != 2 {
		t.Fatalf("esperava 2, got %d", saved)
	}
	vars, err := e.svc.List(e.ctx, e.projID, e.orgID, nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(vars) != 2 {
		t.Fatalf("esperava 2 vars, got %d", len(vars))
	}
}

func TestVariableCrossOrg(t *testing.T) {
	e := newEnv(t)
	otherOrg := uuid.New()
	if _, err := e.svc.List(e.ctx, e.projID, otherOrg, nil); !errors.Is(err, appsdomain.ErrNotFound) {
		t.Fatalf("cross-org list deveria falhar: %v", err)
	}
	if _, err := e.svc.Set(e.ctx, e.projID, otherOrg, nil, e.userID, "X", "y", false); !errors.Is(err, appsdomain.ErrNotFound) {
		t.Fatalf("cross-org set deveria falhar: %v", err)
	}
	if err := e.svc.Delete(e.ctx, e.projID, otherOrg, nil, e.userID, "X"); !errors.Is(err, appsdomain.ErrNotFound) {
		t.Fatalf("cross-org delete deveria falhar: %v", err)
	}
	// env fora do projeto não é aceito
	if _, err := e.svc.List(e.ctx, e.projID, e.orgID, ptr(uuid.New())); !errors.Is(err, appsdomain.ErrNotFound) {
		t.Fatalf("env de outro projeto deveria falhar: %v", err)
	}
}

func ptr(id uuid.UUID) *uuid.UUID { return &id }
