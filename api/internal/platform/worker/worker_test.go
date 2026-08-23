package worker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	deploydomain "aether/internal/modules/deployments/domain"
	memoryqueue "aether/internal/platform/druntime/adapter/memory"
	"aether/internal/platform/druntime/queue"
)

type fakeRuntime struct {
	pulled         []string
	ran            []RunSpec
	removed        []string
	runErr         error
	portVal        string
	healthOK       bool
	built          []string
	buildErr       error
	exposedPort    int
	containerState string
	containerErr   error
}

func (f *fakeRuntime) Pull(ctx context.Context, image string) (string, error) {
	f.pulled = append(f.pulled, image)
	return "pulled " + image, nil
}

func (f *fakeRuntime) Build(ctx context.Context, dir, dockerfile, tag string) (string, error) {
	f.built = append(f.built, tag)
	if f.buildErr != nil {
		return "", f.buildErr
	}
	return "built " + tag, nil
}

func (f *fakeRuntime) ExposedPort(ctx context.Context, image string) (int, error) {
	return f.exposedPort, nil
}

func (f *fakeRuntime) Run(ctx context.Context, spec RunSpec) (string, error) {
	f.ran = append(f.ran, spec)
	if f.runErr != nil {
		return "", f.runErr
	}
	return "container-1", nil
}

func (f *fakeRuntime) Port(ctx context.Context, containerID string) (string, error) {
	return f.portVal, nil
}

func (f *fakeRuntime) HealthCheck(ctx context.Context, hostPort, path string) error {
	if f.healthOK {
		return nil
	}
	return errors.New("health check failed")
}

func (f *fakeRuntime) Remove(ctx context.Context, containerID string) error {
	f.removed = append(f.removed, containerID)
	return nil
}

func (f *fakeRuntime) RemoveByLabel(ctx context.Context, label string) error { return nil }

func (f *fakeRuntime) FollowLogs(ctx context.Context, containerID string, writer io.Writer) error {
	return nil
}

func (f *fakeRuntime) LogTail(ctx context.Context, containerID string, lines int) ([]string, error) {
	return nil, nil
}

type fakeStore struct {
	updates []deploydomain.Deployment
	dep     *deploydomain.Deployment
}

func (f *fakeStore) ListQueued(ctx context.Context) ([]deploydomain.Deployment, error) {
	return nil, nil
}

func (f *fakeStore) ListReady(ctx context.Context) ([]deploydomain.Deployment, error) {
	if f.dep != nil && f.dep.Status == deploydomain.StatusReady {
		return []deploydomain.Deployment{*f.dep}, nil
	}
	return nil, nil
}

func (f *fakeStore) GetDeployment(ctx context.Context, id uuid.UUID) (*deploydomain.Deployment, error) {
	if f.dep != nil && f.dep.ID == id {
		return f.dep, nil
	}
	return nil, nil
}

func (f *fakeStore) ListByApp(ctx context.Context, appID uuid.UUID, limit int) ([]deploydomain.Deployment, error) {
	return nil, nil
}

func (f *fakeStore) UpdateStatus(ctx context.Context, id uuid.UUID, status deploydomain.Status, errMsg, imageRef, containerID string, startedAt, finishedAt *time.Time) error {
	f.updates = append(f.updates, deploydomain.Deployment{
		ID: id, Status: status, Error: errMsg, ImageRef: imageRef, ContainerID: containerID,
		StartedAt: startedAt, FinishedAt: finishedAt,
	})
	return nil
}

func newDeployment(t *testing.T, hc bool) *deploydomain.Deployment {
	t.Helper()
	spec, _ := json.Marshal(map[string]any{
		"name": "api", "image": "nginx:alpine", "port": 80, "mem_mb": 256,
		"health_check": map[string]any{"enabled": hc, "path": "/", "timeout_ms": 5000, "retries": 3},
	})
	env, _ := json.Marshal(map[string]string{"ENV": "prod"})
	return &deploydomain.Deployment{
		ID: uuid.New(), AppID: uuid.New(), Number: 1, Status: deploydomain.StatusQueued,
		ImageRef: "nginx:alpine", EnvSnapshot: env, DeploySpec: spec,
	}
}

func TestWorkerDeploySuccess(t *testing.T) {
	rt := &fakeRuntime{healthOK: true, portVal: "8081"}
	store := &fakeStore{}
	w := &Worker{Store: store, Runtime: rt, Logger: slog.Default()}

	dep := newDeployment(t, true)
	w.deploy(context.Background(), dep)

	if len(rt.pulled) != 1 || rt.pulled[0] != "nginx:alpine" {
		t.Fatalf("pull inesperado: %v", rt.pulled)
	}
	if len(rt.ran) != 1 {
		t.Fatalf("run inesperado: %d", len(rt.ran))
	}
	if rt.ran[0].Image != "nginx:alpine" || rt.ran[0].Port != 80 {
		t.Fatalf("run spec inesperado: %+v", rt.ran[0])
	}
	if len(store.updates) != 4 {
		t.Fatalf("esperava 4 transições (building/starting/health_checking/ready), got %d", len(store.updates))
	}
	if store.updates[3].Status != deploydomain.StatusReady {
		t.Fatalf("status final deveria ser ready: %v", store.updates[3].Status)
	}
	if store.updates[3].ContainerID != "container-1" {
		t.Fatalf("container id não persistido")
	}
}

func TestWorkerDeployWithoutHealthCheck(t *testing.T) {
	rt := &fakeRuntime{}
	store := &fakeStore{}
	w := &Worker{Store: store, Runtime: rt}

	dep := newDeployment(t, false)
	w.deploy(context.Background(), dep)

	if len(store.updates) != 4 {
		t.Fatalf("sem health check: esperava 4 transições, got %d", len(store.updates))
	}
	if store.updates[3].Status != deploydomain.StatusReady {
		t.Fatalf("status final deveria ser ready: %v", store.updates[3].Status)
	}
}

func TestWorkerDeployHealthFail(t *testing.T) {
	rt := &fakeRuntime{healthOK: false, portVal: "8082"}
	store := &fakeStore{}
	w := &Worker{Store: store, Runtime: rt}

	dep := newDeployment(t, true)
	w.deploy(context.Background(), dep)

	last := store.updates[len(store.updates)-1]
	if last.Status != deploydomain.StatusFailed {
		t.Fatalf("status final deveria ser failed: %v", last)
	}
	if len(rt.removed) != 1 {
		t.Fatalf("container falho deveria ser removido")
	}
}

func TestWorkerDeployRunFailure(t *testing.T) {
	rt := &fakeRuntime{runErr: errors.New("run failed")}
	store := &fakeStore{}
	w := &Worker{Store: store, Runtime: rt}

	dep := newDeployment(t, false)
	w.deploy(context.Background(), dep)

	last := store.updates[len(store.updates)-1]
	if last.Status != deploydomain.StatusFailed || last.Error == "" {
		t.Fatalf("falha de run deveria levar a failed: %+v", last)
	}
}

func (f *fakeRuntime) ContainerState(ctx context.Context, containerID string) (string, error) {
	if f.containerErr != nil {
		return "", f.containerErr
	}
	if f.containerState != "" {
		return f.containerState, nil
	}
	return "running", nil
}

func TestWatcherIgnoresTransientContainerStateError(t *testing.T) {
	rt := &fakeRuntime{containerErr: errors.New("podman socket unavailable")}
	dep := newDeployment(t, false)
	dep.Status = deploydomain.StatusReady
	dep.ContainerID = "container-1"
	store := &fakeStore{dep: dep}
	w := &Watcher{Store: store, Runtime: rt}

	w.check(context.Background(), map[uuid.UUID]string{})

	if len(store.updates) != 0 {
		t.Fatalf("transient container state error should not fail deployment: %+v", store.updates)
	}
}

func (f *fakeRuntime) Start(ctx context.Context, containerID string) error {
	return nil
}

func (f *fakeRuntime) Stop(ctx context.Context, containerID string) error {
	return nil
}

func (f *fakeRuntime) Restart(ctx context.Context, containerID string) error {
	return nil
}

func (f *fakeRuntime) Stats(ctx context.Context, containerID string) (ContainerStats, error) {
	return ContainerStats{CPUPercent: 1.5, MemUsage: 1024, MemLimit: 2048}, nil
}

func (f *fakeRuntime) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	return nil, nil
}

func (f *fakeRuntime) Exec(ctx context.Context, containerID string, env []string, args ...string) (string, string, error) {
	return "", "", nil
}

func TestWorkerDeployGitBuild(t *testing.T) {
	rt := &fakeRuntime{healthOK: true, portVal: "8082"}
	store := &fakeStore{}
	repo := makeLocalGitRepo(t)
	spec, _ := json.Marshal(map[string]any{
		"name": "api", "git_url": repo, "git_branch": "main",
		"dockerfile": "Dockerfile", "port": 8080,
		"health_check": map[string]any{"enabled": true, "path": "/", "timeout_ms": 5000, "retries": 3},
	})
	dep := &deploydomain.Deployment{
		ID: uuid.New(), AppID: uuid.New(), Number: 1, Status: deploydomain.StatusQueued,
		DeploySpec: spec, EnvSnapshot: []byte(`{}`),
	}
	w := &Worker{Store: store, Runtime: rt, BuildsDir: t.TempDir()}
	w.deploy(context.Background(), dep)
	if len(rt.built) != 1 {
		t.Fatalf("deveria buildar a imagem: %d", len(rt.built))
	}
	if len(rt.pulled) != 0 {
		t.Fatalf("imagem buildada é local, não deveria puxar: %v", rt.pulled)
	}
	last := store.updates[len(store.updates)-1]
	if last.Status != deploydomain.StatusReady {
		t.Fatalf("status: %+v", last)
	}
}

func TestWorkerDeployGitBuildFails(t *testing.T) {
	rt := &fakeRuntime{buildErr: errors.New("build failed")}
	store := &fakeStore{}
	repo := makeLocalGitRepo(t)
	spec, _ := json.Marshal(map[string]any{
		"name": "api", "git_url": repo, "git_branch": "main",
		"dockerfile": "Dockerfile", "port": 8080,
	})
	dep := &deploydomain.Deployment{
		ID: uuid.New(), AppID: uuid.New(), Number: 1, Status: deploydomain.StatusQueued,
		DeploySpec: spec, EnvSnapshot: []byte(`{}`),
	}
	w := &Worker{Store: store, Runtime: rt, BuildsDir: t.TempDir()}
	w.deploy(context.Background(), dep)
	last := store.updates[len(store.updates)-1]
	if last.Status != deploydomain.StatusFailed || last.Error == "" {
		t.Fatalf("build falho deveria levar a failed: %+v", last)
	}
}

func TestWorkerQueueRun(t *testing.T) {
	mq := memoryqueue.NewQueue()
	dep := newDeployment(t, true)
	rt := &fakeRuntime{healthOK: true, portVal: "8085"}
	store := &fakeStore{dep: dep}
	w := &Worker{Store: store, Runtime: rt, Queue: mq, Logger: slog.Default()}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := mq.Enqueue(ctx, "deployments", queue.Job{DeploymentID: dep.ID.String(), AppID: dep.AppID.String()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	go w.runQueue(ctx)

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if len(rt.ran) == 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(rt.ran) != 1 {
		t.Fatalf("deploy via fila não executado")
	}
	if len(store.updates) == 0 || store.updates[len(store.updates)-1].Status != deploydomain.StatusReady {
		t.Fatalf("deploy via fila não concluiu ready: %+v", store.updates)
	}
}

func TestWorkerQueueSkipsNonQueued(t *testing.T) {
	mq := memoryqueue.NewQueue()
	dep := newDeployment(t, true)
	dep.Status = deploydomain.StatusBuilding
	rt := &fakeRuntime{healthOK: true}
	store := &fakeStore{dep: dep}
	w := &Worker{Store: store, Runtime: rt, Queue: mq, Logger: slog.Default()}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := mq.Enqueue(ctx, "deployments", queue.Job{DeploymentID: dep.ID.String(), AppID: dep.AppID.String()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	go w.runQueue(ctx)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		n, _ := mq.Len(ctx, "deployments")
		if n == 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(rt.ran) != 0 {
		t.Fatalf("deploy não-queued não deveria rodar")
	}
}

type fakeWatcherNotifier struct {
	deploys []deploydomain.DeployEvent
	states  map[uuid.UUID]string
}

func (f *fakeWatcherNotifier) NotifyDeploy(_ context.Context, event deploydomain.DeployEvent) {
	f.deploys = append(f.deploys, event)
}

func (f *fakeWatcherNotifier) NotifyAppState(_ context.Context, appID uuid.UUID, state string) {
	if f.states == nil {
		f.states = map[uuid.UUID]string{}
	}
	f.states[appID] = state
}

func TestWatcherMarksStoppedContainerFailed(t *testing.T) {
	rt := &fakeRuntime{containerState: "exited"}
	dep := newDeployment(t, false)
	dep.Status = deploydomain.StatusReady
	dep.ContainerID = "container-1"
	store := &fakeStore{dep: dep}
	n := &fakeWatcherNotifier{}
	w := &Watcher{Store: store, Runtime: rt, Notifier: n}

	w.check(context.Background(), map[uuid.UUID]string{})

	if len(store.updates) != 1 || store.updates[0].Status != deploydomain.StatusFailed {
		t.Fatalf("deploy ready deveria ir a failed: %+v", store.updates)
	}
	if len(n.deploys) != 1 || n.deploys[0].Status != "failed" {
		t.Fatalf("evento deploy.failed não emitido: %+v", n.deploys)
	}
	if n.states[dep.AppID] != "error" {
		t.Fatalf("app.state deveria ser error: %+v", n.states)
	}
}

func TestWatcherEmitsStateOnce(t *testing.T) {
	rt := &fakeRuntime{containerState: "running"}
	dep := newDeployment(t, false)
	dep.Status = deploydomain.StatusReady
	dep.ContainerID = "container-1"
	store := &fakeStore{dep: dep}
	n := &fakeWatcherNotifier{}
	w := &Watcher{Store: store, Runtime: rt, Notifier: n}
	last := map[uuid.UUID]string{}

	w.check(context.Background(), last)
	w.check(context.Background(), last)

	if n.states[dep.AppID] != "running" {
		t.Fatalf("estado não emitido: %+v", n.states)
	}
	if len(n.states) != 1 {
		t.Fatalf("estado deveria ser emitido apenas uma vez, got %d", len(n.states))
	}
	if len(store.updates) != 0 {
		t.Fatalf("container running não deveria marcar failed")
	}
}

func makeLocalGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runCmd(t, dir, "git", "init", "-q")
	runCmd(t, dir, "git", "config", "user.email", "t@t.com")
	runCmd(t, dir, "git", "config", "user.name", "t")
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM nginx:alpine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runCmd(t, dir, "git", "add", "-A")
	runCmd(t, dir, "git", "commit", "-qm", "init")
	runCmd(t, dir, "git", "branch", "-M", "main")
	return dir
}

func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %s", name, args, out)
	}
}

func TestWorkerDeployGitRollbackUsesImage(t *testing.T) {
	rt := &fakeRuntime{healthOK: true, portVal: "8083"}
	store := &fakeStore{}
	repo := makeLocalGitRepo(t)
	spec, _ := json.Marshal(map[string]any{
		"name": "api", "git_url": repo, "git_branch": "main",
		"dockerfile": "Dockerfile", "port": 8080,
		"health_check": map[string]any{"enabled": true, "path": "/", "timeout_ms": 5000, "retries": 3},
	})
	dep := &deploydomain.Deployment{
		ID: uuid.New(), AppID: uuid.New(), Number: 2, Status: deploydomain.StatusQueued,
		ImageRef: "aether/abc12345:1", DeploySpec: spec, EnvSnapshot: []byte(`{}`),
	}
	w := &Worker{Store: store, Runtime: rt, BuildsDir: t.TempDir()}
	w.deploy(context.Background(), dep)
	if len(rt.built) != 0 {
		t.Fatalf("rollback com ImageRef não deveria rebuildar: %v", rt.built)
	}
	if len(rt.pulled) != 1 || rt.pulled[0] != "aether/abc12345:1" {
		t.Fatalf("rollback deveria puxar a imagem do ImageRef: %v", rt.pulled)
	}
	last := store.updates[len(store.updates)-1]
	if last.Status != deploydomain.StatusReady {
		t.Fatalf("status: %+v", last)
	}
}
