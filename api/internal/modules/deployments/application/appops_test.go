package application

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	appsdomain "aether/internal/modules/apps/domain"
	deploydomain "aether/internal/modules/deployments/domain"
)

type fakeOpsRuntime struct {
	state string
	err   error
	label string
}

func (f fakeOpsRuntime) ContainerState(ctx context.Context, containerID string) (string, error) {
	return f.state, f.err
}
func (f fakeOpsRuntime) Start(ctx context.Context, containerID string) error { return nil }
func (f fakeOpsRuntime) Stop(ctx context.Context, containerID string) error  { return nil }
func (f fakeOpsRuntime) Restart(ctx context.Context, containerID string) error {
	return nil
}

func (f fakeOpsRuntime) Remove(ctx context.Context, containerID string) error { return nil }

func (f fakeOpsRuntime) RemoveByLabel(ctx context.Context, label string) error {
	f.label = label
	return nil
}

type fakeOpsAppStore struct {
	app *appsdomain.App
}

func (f fakeOpsAppStore) GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error) {
	if f.app != nil && f.app.ID == id {
		return f.app, nil
	}
	return nil, deploydomain.ErrNotFound
}
func (f fakeOpsAppStore) ListEnvVars(ctx context.Context, appID uuid.UUID) ([]appsdomain.EnvVar, error) {
	return nil, nil
}
func (f fakeOpsAppStore) ListAppsByOrg(ctx context.Context, orgID uuid.UUID) ([]appsdomain.App, error) {
	return nil, nil
}

type fakeOpsDeployStore struct {
	deps []deploydomain.Deployment
}

func (f fakeOpsDeployStore) ListByApp(ctx context.Context, appID uuid.UUID, limit int) ([]deploydomain.Deployment, error) {
	return f.deps, nil
}
func (f fakeOpsDeployStore) GetDeployment(ctx context.Context, id uuid.UUID) (*deploydomain.Deployment, error) {
	return nil, nil
}
func (f fakeOpsDeployStore) GetByApp(ctx context.Context, appID uuid.UUID, number int) (*deploydomain.Deployment, error) {
	return nil, nil
}
func (f fakeOpsDeployStore) ListQueued(ctx context.Context) ([]deploydomain.Deployment, error) {
	return nil, nil
}

func (f fakeOpsDeployStore) ListReady(ctx context.Context) ([]deploydomain.Deployment, error) {
	return nil, nil
}
func (f fakeOpsDeployStore) NextNumber(ctx context.Context, appID uuid.UUID) (int, error) {
	return 1, nil
}
func (f fakeOpsDeployStore) CreateDeployment(ctx context.Context, dep *deploydomain.Deployment) (*deploydomain.Deployment, error) {
	return dep, nil
}
func (f fakeOpsDeployStore) LastReady(ctx context.Context, appID uuid.UUID) (*deploydomain.Deployment, error) {
	return nil, nil
}
func (f fakeOpsDeployStore) CreateRollback(ctx context.Context, dep *deploydomain.Deployment, from uuid.UUID) (*deploydomain.Deployment, error) {
	return dep, nil
}
func (f fakeOpsDeployStore) MarkRolledBack(ctx context.Context, id uuid.UUID) error { return nil }
func (f fakeOpsDeployStore) UpdateStatus(ctx context.Context, id uuid.UUID, status deploydomain.Status, errMsg, imageRef, containerID string, startedAt, finishedAt *time.Time) error {
	return nil
}

func TestAppOpsState(t *testing.T) {
	appID := uuid.New()
	orgID := uuid.New()
	app := &appsdomain.App{ID: appID, OrgID: orgID, Name: "web"}

	noContainer := &AppOps{Deployments: &Deployments{Apps: fakeOpsAppStore{app: app}, Store: fakeOpsDeployStore{}}, Runtime: fakeOpsRuntime{state: "running"}}
	if state, _ := noContainer.State(context.Background(), appID, orgID); state != "no_container" {
		t.Fatalf("sem container deveria ser no_container: %s", state)
	}

	withContainer := &AppOps{
		Deployments: &Deployments{
			Apps:  fakeOpsAppStore{app: app},
			Store: fakeOpsDeployStore{deps: []deploydomain.Deployment{{AppID: appID, ContainerID: "c1", Status: deploydomain.StatusReady}}},
		},
		Runtime: fakeOpsRuntime{state: "running"},
	}
	if state, _ := withContainer.State(context.Background(), appID, orgID); state != "running" {
		t.Fatalf("estado: %s", state)
	}

	unknown := &AppOps{
		Deployments: &Deployments{
			Apps:  fakeOpsAppStore{app: app},
			Store: fakeOpsDeployStore{deps: []deploydomain.Deployment{{AppID: appID, ContainerID: "c1", Status: deploydomain.StatusReady}}},
		},
		Runtime: fakeOpsRuntime{state: "running", err: deploydomain.ErrNotFound},
	}
	if state, _ := unknown.State(context.Background(), appID, orgID); state != "unknown" {
		t.Fatalf("erro runtime deveria ser unknown: %s", state)
	}
}

func TestAppOpsDeleteServiceUsesCanonicalServiceLabel(t *testing.T) {
	appID := uuid.New()
	serviceID := uuid.New()
	orgID := uuid.New()
	runtime := &recordingOpsRuntime{state: "running"}
	ops := &AppOps{
		Deployments: &Deployments{
			Apps:  fakeOpsAppStore{app: &appsdomain.App{ID: appID, OrgID: orgID}},
			Store: fakeOpsDeployStore{deps: []deploydomain.Deployment{{AppID: appID, ContainerID: "c1"}}},
		},
		Runtime: runtime,
	}
	if err := ops.DeleteService(context.Background(), appID, serviceID, orgID); err != nil {
		t.Fatal(err)
	}
	if runtime.label != "aether.service-id="+serviceID.String() {
		t.Fatalf("unexpected service label: %s", runtime.label)
	}
}

type recordingOpsRuntime struct {
	state string
	label string
}

func (r *recordingOpsRuntime) ContainerState(ctx context.Context, containerID string) (string, error) {
	return r.state, nil
}
func (r *recordingOpsRuntime) Start(ctx context.Context, containerID string) error   { return nil }
func (r *recordingOpsRuntime) Stop(ctx context.Context, containerID string) error    { return nil }
func (r *recordingOpsRuntime) Restart(ctx context.Context, containerID string) error { return nil }
func (r *recordingOpsRuntime) Remove(ctx context.Context, containerID string) error  { return nil }
func (r *recordingOpsRuntime) RemoveByLabel(ctx context.Context, label string) error {
	r.label = label
	return nil
}
