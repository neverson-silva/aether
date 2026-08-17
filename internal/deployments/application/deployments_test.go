package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	appsdomain "aether/internal/apps/domain"
	appsInfra "aether/internal/apps/infra"
	deploydomain "aether/internal/deployments/domain"
	deployInfra "aether/internal/deployments/infra"
	"aether/internal/druntime/queue"
)

type env struct {
	ctx   context.Context
	svc   *Deployments
	apps  *appsInfra.Store
	orgID uuid.UUID
	appID uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	appsStore := appsInfra.NewStore(pool)
	deployStore := deployInfra.NewStore(pool)
	e := &env{
		ctx:   context.Background(),
		svc:   &Deployments{Store: deployStore, Apps: appsStore},
		apps:  appsStore,
		orgID: uuid.New(),
	}
	t.Cleanup(func() {
		_ = appsStore.Close()
		_ = deployStore.Close()
	})
	if _, err := pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, e.orgID, "Deploy Org", "deploy-org"); err != nil {
		t.Fatalf("criar org: %v", err)
	}
	project, err := appsStore.CreateProject(e.ctx, e.orgID, "DeployProj", "deploy-proj", "", "")
	if err != nil {
		t.Fatalf("criar project: %v", err)
	}
	app, err := appsStore.CreateApp(e.ctx, &appsdomain.App{
		OrgID: e.orgID, ProjectID: project.ID, Name: "api", SourceType: "image",
		Image: "nginx:alpine", Port: 80,
	})
	if err != nil {
		t.Fatalf("criar app: %v", err)
	}
	e.appID = app.ID
	return e
}

func TestDeployLifecycle(t *testing.T) {
	e := newEnv(t)
	dep, err := e.svc.Deploy(e.ctx, e.appID, e.orgID, DeployOpts{Trigger: "api", TriggeredBy: "user"})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if dep.Number != 1 || dep.Status != deploydomain.StatusQueued {
		t.Fatalf("deploy inesperado: number=%d status=%s", dep.Number, dep.Status)
	}

	if _, err := e.svc.Deploy(e.ctx, e.appID, e.orgID, DeployOpts{Trigger: "webhook"}); err != nil {
		t.Fatalf("deploy 2: %v", err)
	}
	list, err := e.svc.List(e.ctx, e.appID, e.orgID, 25)
	if err != nil || len(list) != 2 {
		t.Fatalf("list: %v %d", err, len(list))
	}
	if list[0].Number != 2 {
		t.Fatalf("ordenacao deveria ser DESC, got %d primeiro", list[0].Number)
	}
}

func TestDeployIsolationByOrg(t *testing.T) {
	e := newEnv(t)
	if _, err := e.svc.Deploy(e.ctx, e.appID, uuid.New(), DeployOpts{}); !errors.Is(err, appsdomain.ErrNotFound) {
		t.Fatalf("org errada deveria falhar com not found: %v", err)
	}
}

func TestStatusTransitions(t *testing.T) {
	e := newEnv(t)
	dep, err := e.svc.Deploy(e.ctx, e.appID, e.orgID, DeployOpts{})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}

	steps := []deploydomain.Status{
		deploydomain.StatusBuilding,
		deploydomain.StatusStarting,
		deploydomain.StatusHealthChecking,
		deploydomain.StatusReady,
	}
	for _, step := range steps {
		dep, err = e.svc.Transition(e.ctx, e.appID, e.orgID, dep.ID, step)
		if err != nil {
			t.Fatalf("transition para %s: %v", step, err)
		}
		if dep.Status != step {
			t.Fatalf("status deveria ser %s, got %s", step, dep.Status)
		}
	}
	if dep.StartedAt == nil || dep.FinishedAt == nil {
		t.Fatalf("timestamps de build/terminal deveriam estar setados: started=%v finished=%v", dep.StartedAt, dep.FinishedAt)
	}

	if _, err := e.svc.Transition(e.ctx, e.appID, e.orgID, dep.ID, deploydomain.StatusStarting); !errors.Is(err, deploydomain.ErrInvalidTransition) {
		t.Fatalf("transition de terminal deveria falhar: %v", err)
	}
}

func TestInvalidTransition(t *testing.T) {
	e := newEnv(t)
	dep, _ := e.svc.Deploy(e.ctx, e.appID, e.orgID, DeployOpts{})
	if _, err := e.svc.Transition(e.ctx, e.appID, e.orgID, dep.ID, deploydomain.StatusReady); !errors.Is(err, deploydomain.ErrInvalidTransition) {
		t.Fatalf("queued->ready direto deveria falhar: %v", err)
	}
}

func TestRollback(t *testing.T) {
	e := newEnv(t)
	dep, _ := e.svc.Deploy(e.ctx, e.appID, e.orgID, DeployOpts{ImageRef: "nginx:1.25"})
	for _, s := range []deploydomain.Status{deploydomain.StatusBuilding, deploydomain.StatusStarting, deploydomain.StatusHealthChecking, deploydomain.StatusReady} {
		dep, _ = e.svc.Transition(e.ctx, e.appID, e.orgID, dep.ID, s)
	}

	if _, err := e.svc.Rollback(e.ctx, e.appID, e.orgID, "user"); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	old, err := e.svc.Get(e.ctx, dep.ID, e.orgID)
	if err != nil {
		t.Fatalf("get antigo: %v", err)
	}
	if old.Status != deploydomain.StatusRolledBack {
		t.Fatalf("deployment anterior deveria ser rolled_back, got %s", old.Status)
	}

	list, _ := e.svc.List(e.ctx, e.appID, e.orgID, 10)
	if len(list) != 2 || list[0].Status != deploydomain.StatusQueued {
		t.Fatalf("novo deployment deveria ser queued: %+v", list[0])
	}
	if list[0].ImageRef != "nginx:1.25" {
		t.Fatalf("rollback deveria reusar image_ref anterior: %s", list[0].ImageRef)
	}
}

func TestRollbackWithoutReady(t *testing.T) {
	e := newEnv(t)
	if _, err := e.svc.Rollback(e.ctx, e.appID, e.orgID, "user"); !errors.Is(err, deploydomain.ErrNotFound) {
		t.Fatalf("rollback sem deployment ready deveria falhar: %v", err)
	}
}

type fakeQueue struct {
	jobs []queue.Job
}

func (f *fakeQueue) Enqueue(_ context.Context, stream string, job queue.Job) error {
	f.jobs = append(f.jobs, job)
	return nil
}

func (f *fakeQueue) NewConsumer(_ context.Context, stream, group, consumerID string) (queue.Consumer, error) {
	return nil, nil
}

func (f *fakeQueue) Len(_ context.Context, stream string) (int64, error) { return 0, nil }

func (f *fakeQueue) Pending(_ context.Context, stream, group string) (int64, error) { return 0, nil }

func (f *fakeQueue) DeadLetterLen(_ context.Context, stream string) (int64, error) { return 0, nil }

func (f *fakeQueue) Cancel(_ context.Context, stream, jobID string) error { return nil }

type notifierFunc func(ctx context.Context, ev deploydomain.DeployEvent)

func (f notifierFunc) NotifyDeploy(ctx context.Context, ev deploydomain.DeployEvent) { f(ctx, ev) }

func TestDeployEnqueuesAndNotifies(t *testing.T) {
	e := newEnv(t)
	q := &fakeQueue{}
	var notified []deploydomain.DeployEvent
	e.svc.Queue = q
	e.svc.Notifier = notifierFunc(func(_ context.Context, ev deploydomain.DeployEvent) {
		notified = append(notified, ev)
	})

	dep, err := e.svc.Deploy(e.ctx, e.appID, e.orgID, DeployOpts{Trigger: "api", TriggeredBy: "user"})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if len(q.jobs) != 1 || q.jobs[0].DeploymentID != dep.ID.String() {
		t.Fatalf("job não enfileirado: %+v", q.jobs)
	}
	if len(notified) != 1 || notified[0].Status != "queued" || notified[0].DepID != dep.ID {
		t.Fatalf("notifier: %+v", notified)
	}
}
