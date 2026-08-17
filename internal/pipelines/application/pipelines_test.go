package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	appsdomain "aether/internal/apps/domain"
	appsInfra "aether/internal/apps/infra"
	"aether/internal/pipelines/domain"
	"aether/internal/pipelines/infra"
)

type fakeRunner struct{}

func (fakeRunner) RunStage(ctx context.Context, image string, commands []string) (string, error) {
	return "ok " + image + " " + commands[0], nil
}

type env struct {
	ctx   context.Context
	svc   *Pipelines
	orgID uuid.UUID
	appID uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	appsStore := appsInfra.NewStore(pool)
	e := &env{ctx: context.Background(), svc: &Pipelines{Store: store, Apps: appsStore, StageRunner: fakeRunner{}}, orgID: uuid.New()}
	t.Cleanup(func() {
		_ = store.Close()
		_ = appsStore.Close()
	})
	if _, err := pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, e.orgID, "Pip Org", "pip-org"); err != nil {
		t.Fatalf("criar org: %v", err)
	}
	project, err := appsStore.CreateProject(e.ctx, e.orgID, "PipProj", "pip-proj", "", "")
	if err != nil {
		t.Fatalf("criar project: %v", err)
	}
	app, err := appsStore.CreateApp(e.ctx, &appsdomain.App{
		OrgID: e.orgID, ProjectID: project.ID, Name: "api", SourceType: "image", Image: "nginx", Port: 80,
	})
	if err != nil {
		t.Fatalf("criar app: %v", err)
	}
	e.appID = app.ID
	return e
}

func TestPipelineLifecycle(t *testing.T) {
	e := newEnv(t)
	pipeline, err := e.svc.Create(e.ctx, e.orgID, &e.appID, "deploy", "manual", []domain.Stage{
		{Name: "build", Image: "node:20", Commands: []string{"npm ci"}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !pipeline.Enabled || pipeline.Trigger != "manual" {
		t.Fatalf("pipeline inesperado: %+v", pipeline)
	}
	if _, err := e.svc.Create(e.ctx, e.orgID, &e.appID, "", "manual", nil); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("name vazio deveria falhar: %v", err)
	}

	list, err := e.svc.List(e.ctx, e.orgID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}

	run, err := e.svc.Run(e.ctx, pipeline.ID, e.orgID, "manual")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if run.Status != "success" || run.Log == "" {
		t.Fatalf("run inesperado: %+v", run)
	}

	runs, err := e.svc.ListRuns(e.ctx, pipeline.ID, e.orgID, 30)
	if err != nil || len(runs) != 1 {
		t.Fatalf("list runs: %v %d", err, len(runs))
	}

	if err := e.svc.Delete(e.ctx, pipeline.ID, e.orgID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestPipelineIsolation(t *testing.T) {
	e := newEnv(t)
	pipeline, _ := e.svc.Create(e.ctx, e.orgID, &e.appID, "p", "manual", []domain.Stage{{Name: "s", Image: "alpine", Commands: []string{"true"}}})
	if _, err := e.svc.Run(e.ctx, pipeline.ID, uuid.New(), "manual"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("outra org deveria falhar: %v", err)
	}
}

func TestPipelineRunFailure(t *testing.T) {
	e := newEnv(t)
	bad := &Pipelines{Store: e.svc.Store, Apps: e.svc.Apps, StageRunner: failRunner{}}
	pipeline, _ := e.svc.Create(e.ctx, e.orgID, &e.appID, "p", "manual", []domain.Stage{{Name: "s", Image: "alpine", Commands: []string{"false"}}})
	run, err := bad.Run(e.ctx, pipeline.ID, e.orgID, "manual")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if run.Status != "failed" {
		t.Fatalf("stage falho deveria marcar run failed: %s", run.Status)
	}
}

type failRunner struct{}

func (failRunner) RunStage(ctx context.Context, image string, commands []string) (string, error) {
	return "output", errors.New("stage failed")
}
