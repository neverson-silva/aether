package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"aether/internal/modules/gitops/domain"
	"aether/internal/modules/gitops/infra"
)

type env struct {
	ctx   context.Context
	svc   *GitOps
	orgID uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	e := &env{ctx: context.Background(), svc: &GitOps{Store: store}, orgID: uuid.New()}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, e.orgID, "Git Org", "git-org"); err != nil {
		t.Fatalf("criar org: %v", err)
	}
	return e
}

func TestGitOpsLifecycle(t *testing.T) {
	e := newEnv(t)
	g, err := e.svc.Create(e.ctx, e.orgID, "infra", "https://github.com/acme/infra.git", "", "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if g.Branch != "main" || g.Path != "aether.yml" || g.ApplyMode != "manual" || g.LastStatus != "pending" {
		t.Fatalf("defaults inesperados: %+v", g)
	}

	list, err := e.svc.List(e.ctx, e.orgID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}

	synced, err := e.svc.Sync(e.ctx, g.ID, e.orgID, "abc123", 2, 1, 3)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if synced.LastStatus != "synced" || synced.DriftAdded != 2 || synced.LastSHA != "abc123" {
		t.Fatalf("sync inesperado: %+v", synced)
	}

	if _, err := e.svc.Sync(e.ctx, g.ID, uuid.New(), "", 0, 0, 0); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("outra org deveria falhar: %v", err)
	}

	if err := e.svc.Delete(e.ctx, g.ID, e.orgID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestGitOpsValidation(t *testing.T) {
	e := newEnv(t)
	if _, err := e.svc.Create(e.ctx, e.orgID, "", "repo", "", "", ""); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("name vazio deveria falhar: %v", err)
	}
	if _, err := e.svc.Create(e.ctx, e.orgID, "x", "", "", "", ""); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("repo vazio deveria falhar: %v", err)
	}
	if _, err := e.svc.Create(e.ctx, e.orgID, "x", "repo", "", "", "auto"); err != nil {
		t.Fatalf("auto deveria ser aceito: %v", err)
	}
	if _, err := e.svc.Create(e.ctx, e.orgID, "x", "repo", "", "", "weird"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("apply_mode inválido deveria falhar: %v", err)
	}
}
