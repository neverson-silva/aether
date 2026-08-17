package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	appsdomain "aether/internal/apps/domain"
	appsInfra "aether/internal/apps/infra"
	"aether/internal/jobs/domain"
	"aether/internal/jobs/infra"
)

type fakeWorkerRuntime struct{}

func (fakeWorkerRuntime) Run(ctx context.Context, name, image, command string, env []string) (string, error) {
	return "container-" + name, nil
}
func (fakeWorkerRuntime) Stop(ctx context.Context, containerID string) error { return nil }
func (fakeWorkerRuntime) Remove(ctx context.Context, containerID string) error {
	return nil
}

type env struct {
	ctx   context.Context
	svc   *Jobs
	orgID uuid.UUID
	appID uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	appsStore := appsInfra.NewStore(pool)
	e := &env{ctx: context.Background(), svc: &Jobs{Store: store, Apps: appsStore, Runtime: fakeWorkerRuntime{}}, orgID: uuid.New()}
	t.Cleanup(func() {
		_ = store.Close()
		_ = appsStore.Close()
	})
	if _, err := pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, e.orgID, "Jobs Org", "jobs-org"); err != nil {
		t.Fatalf("criar org: %v", err)
	}
	project, err := appsStore.CreateProject(e.ctx, e.orgID, "JobsProj", "jobs-proj", "", "")
	if err != nil {
		t.Fatalf("criar project: %v", err)
	}
	app, err := appsStore.CreateApp(e.ctx, &appsdomain.App{
		OrgID: e.orgID, ProjectID: project.ID, Name: "api", SourceType: "image",
		Image: "nginx", Port: 80,
	})
	if err != nil {
		t.Fatalf("criar app: %v", err)
	}
	e.appID = app.ID
	return e
}

func TestCronJobLifecycle(t *testing.T) {
	e := newEnv(t)
	job, err := e.svc.CreateCronJob(e.ctx, e.appID, e.orgID, "backup", "0 3 * * *", "backup.sh")
	if err != nil {
		t.Fatalf("create cron: %v", err)
	}
	if !job.Enabled || job.Schedule != "0 3 * * *" {
		t.Fatalf("cron inesperado: %+v", job)
	}

	if _, err := e.svc.CreateCronJob(e.ctx, e.appID, e.orgID, "bad", "not-a-cron", "x"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("schedule inválido deveria falhar: %v", err)
	}

	list, err := e.svc.ListCronJobs(e.ctx, e.appID, e.orgID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}

	disabled := false
	updated, err := e.svc.UpdateCronJob(e.ctx, job.ID, e.orgID, nil, nil, &disabled)
	if err != nil {
		t.Fatalf("update cron: %v", err)
	}
	if updated.Enabled {
		t.Fatalf("cron deveria estar desabilitado")
	}

	all, err := e.svc.ListAllCronJobs(e.ctx, e.orgID)
	if err != nil || len(all) != 1 {
		t.Fatalf("list all: %v %d", err, len(all))
	}

	if err := e.svc.DeleteCronJob(e.ctx, job.ID, e.orgID); err != nil {
		t.Fatalf("delete cron: %v", err)
	}
}

func TestCronIsolation(t *testing.T) {
	e := newEnv(t)
	job, _ := e.svc.CreateCronJob(e.ctx, e.appID, e.orgID, "job", "0 * * * *", "x")
	if _, err := e.svc.UpdateCronJob(e.ctx, job.ID, uuid.New(), nil, nil, nil); !errors.Is(err, appsdomain.ErrNotFound) {
		t.Fatalf("outra org deveria falhar: %v", err)
	}
}

func TestWorkerLifecycle(t *testing.T) {
	e := newEnv(t)
	worker, err := e.svc.CreateWorker(e.ctx, e.appID, e.orgID, "mailer", "node mailer.js", 2)
	if err != nil {
		t.Fatalf("create worker: %v", err)
	}
	if worker.Status != "stopped" || worker.Replicas != 2 {
		t.Fatalf("worker inesperado: %+v", worker)
	}

	if err := e.svc.StartWorker(e.ctx, worker.ID, e.orgID); err != nil {
		t.Fatalf("start: %v", err)
	}
	workers, _ := e.svc.ListWorkers(e.ctx, e.appID, e.orgID)
	if workers[0].Status != "running" {
		t.Fatalf("worker deveria estar running")
	}

	if err := e.svc.StopWorker(e.ctx, worker.ID, e.orgID); err != nil {
		t.Fatalf("stop: %v", err)
	}
	workers, _ = e.svc.ListWorkers(e.ctx, e.appID, e.orgID)
	if workers[0].Status != "stopped" {
		t.Fatalf("worker deveria estar stopped")
	}

	if err := e.svc.DeleteWorker(e.ctx, worker.ID, e.orgID); err != nil {
		t.Fatalf("delete worker: %v", err)
	}
}

func TestWorkerValidation(t *testing.T) {
	e := newEnv(t)
	if _, err := e.svc.CreateWorker(e.ctx, e.appID, e.orgID, "", "x", 1); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("nome vazio deveria falhar: %v", err)
	}
	if _, err := e.svc.CreateWorker(e.ctx, e.appID, e.orgID, "w", "x", 0); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("replicas inválidas deveriam falhar: %v", err)
	}
}

func TestPolicyLifecycle(t *testing.T) {
	e := newEnv(t)
	policy, err := e.svc.GetPolicy(e.ctx, e.appID, e.orgID)
	if err != nil {
		t.Fatalf("get policy default: %v", err)
	}
	if policy.Enabled || policy.CPUMax != 4 {
		t.Fatalf("policy default inesperada: %+v", policy)
	}

	policy.Enabled = true
	policy.CPUMin = 0.5
	policy.CPUMax = 8
	policy.MemMaxMB = 4096
	saved, err := e.svc.SavePolicy(e.ctx, e.appID, e.orgID, policy)
	if err != nil {
		t.Fatalf("save policy: %v", err)
	}
	if !saved.Enabled || saved.CPUMax != 8 {
		t.Fatalf("policy salva divergente: %+v", saved)
	}

	got, err := e.svc.GetPolicy(e.ctx, e.appID, e.orgID)
	if err != nil || !got.Enabled || got.CPUMax != 8 {
		t.Fatalf("policy relida divergente: %+v", got)
	}
}

func TestPolicyValidation(t *testing.T) {
	e := newEnv(t)
	bad := &domain.Policy{CPUMin: 2, CPUMax: 1, MemMinMB: 0, MemMaxMB: 10, ScaleUpPct: 80, ScaleDownPct: 15, CooldownMin: 5}
	if _, err := e.svc.SavePolicy(e.ctx, e.appID, e.orgID, bad); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("cpu_max < cpu_min deveria falhar: %v", err)
	}
}

func TestPolicyEvents(t *testing.T) {
	e := newEnv(t)
	events, err := e.svc.PolicyEvents(e.ctx, e.appID, e.orgID, 50)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("sem eventos deveria estar vazio")
	}
}
