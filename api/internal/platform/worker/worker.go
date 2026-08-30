package worker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	appsdomain "aether/internal/modules/apps/domain"
	deploydomain "aether/internal/modules/deployments/domain"
	"aether/internal/platform/druntime/queue"
	"aether/internal/platform/git"
	"aether/internal/platform/observability"
	"aether/internal/platform/planner"
)

type DeploymentStore interface {
	ListReady(ctx context.Context) ([]deploydomain.Deployment, error)
	GetDeployment(ctx context.Context, id uuid.UUID) (*deploydomain.Deployment, error)
	ListByApp(ctx context.Context, appID uuid.UUID, limit int) ([]deploydomain.Deployment, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status deploydomain.Status, errMsg, imageRef, containerID string, startedAt, finishedAt *time.Time) error
}

type AppStore interface {
	GetAppByID(ctx context.Context, id uuid.UUID) (*appsdomain.App, error)
}

type Worker struct {
	Store           DeploymentStore
	Apps            AppStore
	Runtime         Runtime
	Logger          *slog.Logger
	LogsDir         string
	BuildsDir       string
	UploadsDir      string
	IngressNetwork  string
	Notifier        DeployNotifier
	LogNotifier     LogNotifier
	CnbBuilder      string
	DockerHost      string
	BuildDockerHost string
	Images          ImageRuntime
	Builder         ImageBuildRuntime
	Registry        ImageRegistryRuntime
	Queue           queue.Queue
	ServiceDeploy   func(context.Context, string, uuid.UUID, uuid.UUID, uuid.UUID) (string, error)
	ComposeDeploy   interface {
		UpApp(context.Context, uuid.UUID, uuid.UUID) (string, error)
	}
	Metrics           *observability.Metrics
	deploymentTimeout time.Duration

	mu         sync.Mutex
	inFlight   map[uuid.UUID]bool
	cancellers map[uuid.UUID]context.CancelFunc
}

type cancellationWatcher interface {
	WatchCancellations(context.Context, string, func(string)) (func(), error)
}

type interruptedDeploymentStore interface {
	RecoverInterrupted(context.Context, time.Time) error
}

func (w *Worker) RecoverInterrupted(ctx context.Context, cutoff time.Time) error {
	store, ok := w.Store.(interruptedDeploymentStore)
	if !ok {
		return nil
	}
	return store.RecoverInterrupted(ctx, cutoff)
}

const maxDeploymentTimeout = 20 * time.Minute

type DeployNotifier interface {
	NotifyDeploy(ctx context.Context, event deploydomain.DeployEvent)
}

type LogNotifier interface {
	NotifyDeployLog(ctx context.Context, appID, depID uuid.UUID, line string)
}

type ServiceLogNotifier interface {
	NotifyServiceDeployLog(ctx context.Context, serviceID, appID, depID uuid.UUID, line string)
}

func (w *Worker) Run(ctx context.Context, interval time.Duration) {
	if w.Queue != nil {
		w.runQueue(ctx)
		return
	}
	w.log(ctx, "deployment queue is not configured", errors.New("NATS deployment queue is required"))
}

func (w *Worker) CancelDeployment(id uuid.UUID) bool {
	w.mu.Lock()
	cancel, ok := w.cancellers[id]
	w.mu.Unlock()
	if !ok {
		return false
	}
	cancel()
	return true
}

func (w *Worker) runQueue(ctx context.Context) {
	w.mu.Lock()
	w.inFlight = map[uuid.UUID]bool{}
	w.mu.Unlock()
	consumer, err := w.Queue.NewConsumer(ctx, "deployments", "workers", "aether-deploy")
	if err != nil {
		w.log(ctx, "queue consumer", err)
		return
	}
	defer consumer.Close()
	var stopCancellations func()
	if watcher, ok := w.Queue.(cancellationWatcher); ok {
		stopCancellations, err = watcher.WatchCancellations(ctx, "deployments", func(id string) {
			if parsed, parseErr := uuid.Parse(id); parseErr == nil {
				w.CancelDeployment(parsed)
			}
		})
		if err != nil {
			w.log(ctx, "queue cancellation watcher", err)
			return
		}
		defer stopCancellations()
	}
	go w.consumeQueueLoop(ctx, consumer)
	<-ctx.Done()
}

func (w *Worker) consumeQueueLoop(ctx context.Context, consumer queue.Consumer) {
	for {
		job, err := consumer.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			w.log(ctx, "queue next", err)
			continue
		}
		if job.Type == "deployment.execute" {
			w.logInfo(ctx, "deployment job received", "job_id", job.ID, "deployment_id", job.DeploymentID)
			stopProgress := queue.StartProgress(ctx, consumer, job)
			var payload struct {
				Kind         string `json:"kind"`
				ServiceID    string `json:"service_id"`
				SpecID       string `json:"spec_id"`
				OrgID        string `json:"org_id"`
				DeploymentID string `json:"deployment_id"`
			}
			if err := json.Unmarshal(job.Payload, &payload); err != nil {
				w.log(ctx, "decode service deployment job", err)
				stopProgress()
				_ = consumer.Ack(ctx, job)
				continue
			}
			serviceID, serviceErr := uuid.Parse(payload.ServiceID)
			specID, specErr := uuid.Parse(payload.SpecID)
			orgID, orgErr := uuid.Parse(payload.OrgID)
			if serviceErr != nil || specErr != nil || orgErr != nil {
				w.log(ctx, "parse service deployment job identifiers", fmt.Errorf("service=%q spec=%q org=%q", payload.ServiceID, payload.SpecID, payload.OrgID))
				stopProgress()
				_ = consumer.Ack(ctx, job)
				continue
			}
			deploymentID, deploymentErr := uuid.Parse(payload.DeploymentID)
			if deploymentErr != nil {
				stopProgress()
				_ = consumer.Ack(ctx, job)
				continue
			}
			dep, err := w.Store.GetDeployment(ctx, deploymentID)
			if err != nil {
				stopProgress()
				_ = consumer.Nack(ctx, job)
				continue
			}
			if dep == nil || dep.Status != deploydomain.StatusQueued {
				stopProgress()
				_ = consumer.Ack(ctx, job)
				continue
			}
			var processErr error
			if payload.Kind == "app" {
				processErr = w.processQueueJob(ctx, dep)
			} else {
				processErr = w.processServiceQueueJob(ctx, dep, payload.Kind, serviceID, specID, orgID)
			}
			stopProgress()
			if processErr != nil {
				_ = consumer.Nack(ctx, job)
			} else {
				_ = consumer.Ack(ctx, job)
			}
			continue
		}
		depID, err := uuid.Parse(job.DeploymentID)
		if err != nil {
			_ = consumer.Ack(ctx, job)
			continue
		}
		w.mu.Lock()
		if w.inFlight[depID] {
			w.mu.Unlock()
			_ = consumer.Nack(ctx, job)
			continue
		}
		w.inFlight[depID] = true
		w.mu.Unlock()
		dep, err := w.Store.GetDeployment(ctx, depID)
		if err != nil {
			w.mu.Lock()
			delete(w.inFlight, depID)
			w.mu.Unlock()
			_ = consumer.Nack(ctx, job)
			continue
		}
		if dep == nil || dep.Status != deploydomain.StatusQueued {
			w.mu.Lock()
			delete(w.inFlight, depID)
			w.mu.Unlock()
			_ = consumer.Ack(ctx, job)
			continue
		}
		stopProgress := queue.StartProgress(ctx, consumer, job)
		finish := func(bool) {}
		if w.Metrics != nil {
			finish = w.Metrics.StartJob(job.Type)
		}
		processErr := w.processQueueJob(ctx, dep)
		stopProgress()
		if w.Metrics != nil {
			current, getErr := w.Store.GetDeployment(ctx, depID)
			finish(processErr != nil || getErr == nil && current != nil && current.Status == deploydomain.StatusFailed)
		}
		w.mu.Lock()
		delete(w.inFlight, depID)
		w.mu.Unlock()
		if w.queueIsPermanent(processErr) {
			_ = consumer.Ack(ctx, job)
		} else if processErr != nil {
			_ = consumer.Nack(ctx, job)
		} else {
			_ = consumer.Ack(ctx, job)
		}
	}
}

func (w *Worker) notifyServiceDeployment(ctx context.Context, deploymentID, serviceID uuid.UUID, status, detail string) {
	if w.Notifier == nil {
		return
	}
	w.Notifier.NotifyDeploy(ctx, deploydomain.DeployEvent{AppID: uuid.Nil, ServiceID: serviceID, DepID: deploymentID, Status: status, Detail: detail})
}

func (w *Worker) queueIsPermanent(err error) bool {
	return err != nil && queue.IsPermanent(err)
}

func (w *Worker) processQueueJob(ctx context.Context, dep *deploydomain.Deployment) error {
	if err := w.deploy(ctx, dep); err != nil {
		return err
	}
	current, err := w.Store.GetDeployment(ctx, dep.ID)
	if err != nil {
		return err
	}
	if current == nil {
		return errors.New("deployment disappeared during processing")
	}
	if current.Status.Terminal() {
		return nil
	}
	return fmt.Errorf("deployment ended in non-terminal state %s", current.Status)
}

func (w *Worker) processServiceQueueJob(ctx context.Context, dep *deploydomain.Deployment, kind string, serviceID, specID, orgID uuid.UUID) error {
	if w.ServiceDeploy == nil {
		return queue.Permanent(errors.New("service deployment materializer is not configured"))
	}
	current, err := w.Store.GetDeployment(ctx, dep.ID)
	if err != nil {
		return err
	}
	if current == nil || current.Status != deploydomain.StatusQueued {
		return nil
	}
	timeout := w.deploymentTimeout
	if timeout <= 0 || timeout > maxDeploymentTimeout {
		timeout = maxDeploymentTimeout
	}
	deploymentCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	w.mu.Lock()
	if w.cancellers == nil {
		w.cancellers = make(map[uuid.UUID]context.CancelFunc)
	}
	w.cancellers[dep.ID] = cancel
	w.mu.Unlock()
	defer func() {
		w.mu.Lock()
		delete(w.cancellers, dep.ID)
		w.mu.Unlock()
	}()
	if err := w.setStatus(deploymentCtx, dep, deploydomain.StatusBuilding, "", ""); err != nil {
		return err
	}
	if w.deploymentCancelled(dep.ID) {
		return nil
	}
	containerID, err := w.ServiceDeploy(deploymentCtx, kind, serviceID, specID, orgID)
	if err != nil {
		if w.deploymentCancelled(dep.ID) {
			return nil
		}
		if deploymentCtx.Err() != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		w.fail(deploymentCtx, dep, containerID, err)
		return nil
	}
	if w.deploymentCancelled(dep.ID) {
		return nil
	}
	if err := w.setStatus(deploymentCtx, dep, deploydomain.StatusStarting, dep.ImageRef, containerID); err != nil {
		return err
	}
	if err := w.setStatus(deploymentCtx, dep, deploydomain.StatusHealthChecking, dep.ImageRef, containerID); err != nil {
		return err
	}
	if err := w.setStatus(deploymentCtx, dep, deploydomain.StatusReady, dep.ImageRef, containerID); err != nil {
		return err
	}
	return nil
}

func (w *Worker) deploymentCancelled(id uuid.UUID) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dep, err := w.Store.GetDeployment(ctx, id)
	return err == nil && dep != nil && dep.Status == deploydomain.StatusCancelled
}

func (w *Worker) deploy(ctx context.Context, dep *deploydomain.Deployment) error {
	timeout := w.deploymentTimeout
	if timeout <= 0 || timeout > maxDeploymentTimeout {
		timeout = maxDeploymentTimeout
	}
	deploymentCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	w.mu.Lock()
	if w.cancellers == nil {
		w.cancellers = make(map[uuid.UUID]context.CancelFunc)
	}
	w.cancellers[dep.ID] = cancel
	w.mu.Unlock()
	defer func() {
		w.mu.Lock()
		delete(w.cancellers, dep.ID)
		w.mu.Unlock()
	}()
	ctx = deploymentCtx
	if current, err := w.Store.GetDeployment(ctx, dep.ID); err == nil && current != nil && current.Status != deploydomain.StatusQueued {
		return nil
	}
	spec, hc, err := buildSpec(dep)
	if err != nil {
		return w.fail(ctx, dep, "", err)
	}
	if err := w.setStatus(ctx, dep, deploydomain.StatusBuilding, spec.Image, ""); err != nil {
		return err
	}
	if spec.BuildType == "compose" {
		if w.ComposeDeploy == nil {
			return w.fail(ctx, dep, "", errors.New("compose deployment is not configured"))
		}
		app, err := w.Apps.GetAppByID(ctx, dep.AppID)
		if err != nil {
			return w.fail(ctx, dep, "", err)
		}
		containerID, err := w.ComposeDeploy.UpApp(ctx, dep.AppID, app.OrgID)
		if err != nil {
			return w.fail(ctx, dep, "", err)
		}
		return w.setStatus(ctx, dep, deploydomain.StatusReady, "", containerID)
	}
	built := false
	switch {
	case spec.UploadID != "" && dep.ImageRef == "":
		image, err := w.buildUploadSource(ctx, dep, spec)
		if err != nil {
			return w.fail(ctx, dep, "", err)
		}
		spec.Image = image
		built = true
	case spec.GitURL != "" && dep.ImageRef == "":
		image, err := w.buildGitSource(ctx, dep, spec)
		if err != nil {
			return w.fail(ctx, dep, "", err)
		}
		spec.Image = image
		built = true
	}
	if !built {
		w.appendLog(dep, "pulling image "+spec.Image)
		images := w.Images
		if images == nil {
			images = w.Runtime
		}
		out, err := images.Pull(ctx, spec.Image)
		if err != nil {
			w.appendLog(dep, out)
			return w.fail(ctx, dep, "", err)
		}
		if trimmed := strings.TrimSpace(out); trimmed != "" {
			w.appendLog(dep, trimmed)
		}
	}
	w.appendLog(dep, "starting container "+spec.Name)
	containerPort := spec.ContainerPort
	if containerPort == 0 {
		images := w.Images
		if images == nil {
			images = w.Runtime
		}
		containerPort, _ = images.ExposedPort(ctx, spec.Image)
	}
	if containerPort == 0 {
		containerPort = spec.Port
	}
	w.removeOldContainers(ctx, dep.AppID, dep.ID)
	serviceID := dep.AppID
	if provider, ok := w.Apps.(interface {
		GetServiceID(context.Context, uuid.UUID) (uuid.UUID, error)
	}); ok {
		if resolved, resolveErr := provider.GetServiceID(ctx, dep.AppID); resolveErr == nil && resolved != uuid.Nil {
			serviceID = resolved
		}
	}
	labels := map[string]string{
		"aether.owner":        "user",
		"aether.service-type": "app",
		"aether.service-id":   serviceID.String(),
		"aether.spec-id":      dep.AppID.String(),
	}
	if w.Apps != nil {
		if app, err := w.Apps.GetAppByID(ctx, dep.AppID); err == nil {
			labels["aether.service-name"] = app.Name
			labels["aether.project-id"] = app.ProjectID.String()
		}
	}
	containerID, err := w.Runtime.Run(ctx, RunSpec{
		Name: spec.Name, Image: spec.Image, Env: spec.Env, Port: spec.Port, ContainerPort: containerPort,
		Network: w.IngressNetwork, NetworkAlias: "app-" + dep.AppID.String()[:8],
		MemMB: spec.MemMB, CPUs: spec.CPUs, StorageMB: spec.StorageMB, Labels: labels,
	})
	if err != nil {
		return w.fail(ctx, dep, "", err)
	}
	w.appendLog(dep, "container "+containerID+" started")
	if err := w.setStatus(ctx, dep, deploydomain.StatusStarting, spec.Image, containerID); err != nil {
		return err
	}
	if err := w.setStatus(ctx, dep, deploydomain.StatusHealthChecking, spec.Image, containerID); err != nil {
		return err
	}
	if !hc.Enabled {
		w.appendLog(dep, "health check disabled, deploy ready")
		return w.setStatus(ctx, dep, deploydomain.StatusReady, spec.Image, containerID)
	}
	hostPort, err := w.Runtime.Port(ctx, containerID)
	if err != nil {
		return w.fail(ctx, dep, containerID, err)
	}
	w.appendLog(dep, "health check http://127.0.0.1:"+hostPort+hc.Path)
	if err := w.checkHealth(ctx, hostPort, hc); err != nil {
		return w.fail(ctx, dep, containerID, err)
	}
	w.appendLog(dep, "deploy ready")
	if err := w.setStatus(ctx, dep, deploydomain.StatusReady, spec.Image, containerID); err != nil {
		return err
	}
	return nil
}

func (w *Worker) removeOldContainers(ctx context.Context, appID, currentDepID uuid.UUID) {
	deps, err := w.Store.ListByApp(ctx, appID, 50)
	if err != nil {
		return
	}
	for i := range deps {
		d := &deps[i]
		if d.ID == currentDepID || d.ContainerID == "" {
			continue
		}
		_ = w.Runtime.Remove(ctx, d.ContainerID)
	}
}

func (w *Worker) checkHealth(ctx context.Context, hostPort string, hc healthCheck) error {
	timeout := time.Duration(hc.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	deadline := time.Now().Add(timeout)
	retries := hc.Retries
	if retries <= 0 {
		retries = 10
	}
	var lastErr error
	for time.Now().Before(deadline) {
		if err := w.Runtime.HealthCheck(ctx, hostPort, hc.Path); err == nil {
			return nil
		} else {
			lastErr = err
		}
		retries--
		if retries <= 0 {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if lastErr == nil {
		lastErr = context.DeadlineExceeded
	}
	return lastErr
}

func (w *Worker) buildGitSource(ctx context.Context, dep *deploydomain.Deployment, spec runSpec) (string, error) {
	if w.BuildsDir == "" {
		return "", errors.New("builds directory not configured")
	}
	srcDir := filepath.Join(w.BuildsDir, "sources", dep.ID.String())
	if err := os.RemoveAll(srcDir); err != nil {
		return "", err
	}
	w.appendLog(dep, "cloning "+spec.GitURL)
	branch := spec.GitBranch
	if branch == "" {
		branch = "main"
	}
	if err := git.Clone(ctx, spec.GitURL, branch, srcDir); err != nil {
		return "", err
	}
	tag := "aether/" + dep.AppID.String()[:8] + ":" + strconv.Itoa(dep.Number)
	return w.buildFromDir(ctx, dep, spec, srcDir, tag)
}

func (w *Worker) buildUploadSource(ctx context.Context, dep *deploydomain.Deployment, spec runSpec) (string, error) {
	if w.UploadsDir == "" {
		return "", errors.New("uploads directory not configured")
	}
	srcDir := filepath.Join(w.UploadsDir, spec.UploadID)
	if st, err := os.Stat(srcDir); err != nil || !st.IsDir() {
		return "", fmt.Errorf("upload %q not found", spec.UploadID)
	}
	tag := "aether/" + dep.AppID.String()[:8] + ":" + strconv.Itoa(dep.Number)
	return w.buildFromDir(ctx, dep, spec, srcDir, tag)
}

func (w *Worker) buildFromDir(ctx context.Context, dep *deploydomain.Deployment, spec runSpec, srcDir, tag string) (string, error) {
	switch spec.BuildType {
	case "dockerfile":
		return w.buildDockerfile(ctx, dep, spec, srcDir, tag)
	case "custom":
		return w.buildCommandSource(ctx, dep, spec, srcDir, tag)
	default:
		df := spec.Dockerfile
		if df == "" {
			df = "Dockerfile"
		}
		if _, err := os.Stat(filepath.Join(srcDir, df)); err == nil {
			return w.buildDockerfile(ctx, dep, spec, srcDir, tag)
		}
		return w.buildSmartBuild(ctx, dep, spec, srcDir, tag)
	}
}

func (w *Worker) streamCmd(ctx context.Context, dep *deploydomain.Deployment, cmd *exec.Cmd) (string, error) {
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	var out bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
		for sc.Scan() {
			line := strings.TrimRight(sc.Text(), "\r")
			out.WriteString(line + "\n")
			if line != "" {
				w.appendLog(dep, line)
			}
		}
	}()
	err := cmd.Run()
	_ = pw.Close()
	<-done
	return out.String(), err
}

func (w *Worker) buildSmartBuild(ctx context.Context, dep *deploydomain.Deployment, spec runSpec, srcDir, tag string) (string, error) {
	img := "aether/" + dep.ID.String()[:8]
	builder := w.CnbBuilder
	if builder == "" {
		builder = "127.0.0.1:1500/builder:node-spa"
	}
	dockerHost := w.BuildDockerHost
	if dockerHost == "" {
		dockerHost = "unix:///var/run/docker.sock"
	}
	spa := isStaticSPA(srcDir)

	if plan, err := planner.Detect(srcDir); err == nil {
		w.appendLog(dep, fmt.Sprintf("cnb: detected framework=%q type=%q build=%q output=%q",
			plan.Framework, plan.AppType, plan.BuildCommand, plan.OutputDir))
		if spa {
			w.appendLog(dep, "cnb: detected SPA — group aether/spa-static (static server + SPA fallback)")
		} else {
			w.appendLog(dep, "cnb: server application — group aether/node-server (start:prod > start)")
		}
	}

	w.appendLog(dep, "cnb (smartbuild): building "+img+" with builder "+builder)
	w.appendLog(dep, "cnb: docker host "+dockerHost)
	args := []string{"build", img, "-p", srcDir, "-B", builder, "--docker-host=inherit", "--pull-policy=never", "--platform", "linux/" + runtime.GOARCH}
	for _, e := range cnbBuildEnv(srcDir, spec) {
		args = append(args, "--env", e)
	}
	var out string
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		cmd := exec.CommandContext(ctx, "pack", args...)
		cmd.Env = append(os.Environ(), "DOCKER_HOST="+dockerHost, "DOCKER_API_VERSION=1.40", "PACK_VOLUME_KEY=aether-"+dep.ID.String()[:8])
		out, err = w.streamCmd(ctx, dep, cmd)
		if err == nil || !isTransientCNBError(out) || attempt == 1 {
			break
		}
		w.appendLog(dep, "cnb: build runtime connection interrupted; retrying export")
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		if strings.Contains(out, "No buildpack groups passed detection") {
			w.appendLog(dep, "cnb: no buildpack detected the application. Manual configuration required: provide a Dockerfile in the source or use build_type custom (install/build/start).")
			return "", fmt.Errorf("no CNB buildpack detected the application: provide a Dockerfile in the source or use build_type custom")
		}
		w.appendLog(dep, "fail: "+strings.TrimSpace(out)+": "+err.Error())
		return "", fmt.Errorf("%s: %w", strings.TrimSpace(out), err)
	}
	images := w.Registry
	if images == nil {
		if runtime, ok := w.Runtime.(ImageRegistryRuntime); ok {
			images = runtime
		}
	}
	if images == nil {
		return "", errors.New("image registry runtime unavailable")
	}
	if err := images.Tag(ctx, img+":latest", tag); err != nil {
		return "", err
	}
	return tag, nil
}

func isTransientCNBError(output string) bool {
	value := strings.ToLower(output)
	return strings.Contains(value, "connection reset by peer") ||
		strings.Contains(value, "proxy already running") ||
		strings.Contains(value, "payload does not match any of the supported image formats")
}

func cnbBuildEnv(srcDir string, spec runSpec) []string {
	out := []string{"CNB_PLATFORM_API=0.12"}
	for _, e := range spec.Env {
		if strings.HasPrefix(e, "CNB_PLATFORM_API=") {
			continue
		}
		if strings.HasPrefix(e, "BP_") || strings.HasPrefix(e, "CNB_") {
			out = append(out, e)
		}
	}
	return out
}

var staticSPADevStarts = []*regexp.Regexp{
	regexp.MustCompile(`\bng serve\b`),
	regexp.MustCompile(`\bvite\b`),
	regexp.MustCompile(`\breact-scripts start\b`),
	regexp.MustCompile(`\bnext dev\b`),
	regexp.MustCompile(`\bnuxt dev\b`),
	regexp.MustCompile(`\bastro dev\b`),
	regexp.MustCompile(`\bgatsby develop\b`),
	regexp.MustCompile(`\bvue-cli-service serve\b`),
	regexp.MustCompile(`\bwebpack-dev-server\b`),
	regexp.MustCompile(`\bquasar dev\b`),
	regexp.MustCompile(`\bdocusaurus start\b`),
	regexp.MustCompile(`\bsvelte-kit dev\b`),
	regexp.MustCompile(`\bexpo start\b`),
	regexp.MustCompile(`\bionic serve\b`),
}

func isStaticSPA(srcDir string) bool {
	b, err := os.ReadFile(filepath.Join(srcDir, "package.json"))
	if err != nil {
		return false
	}
	var pkg struct {
		Scripts         map[string]string `json:"scripts"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(b, &pkg) != nil {
		return false
	}
	if _, ok := pkg.Dependencies["@analogjs/platform"]; ok {
		return false
	}
	if _, ok := pkg.DevDependencies["@analogjs/platform"]; ok {
		return false
	}
	if _, ok := pkg.Dependencies["@analogjs/vite-plugin-angular"]; ok {
		return false
	}
	if _, ok := pkg.DevDependencies["@analogjs/vite-plugin-angular"]; ok {
		return false
	}
	if _, ok := pkg.Dependencies["@nestjs/core"]; ok {
		return false
	}
	if _, ok := pkg.Dependencies["@nestjs/common"]; ok {
		return false
	}
	if _, ok := pkg.DevDependencies["@nestjs/core"]; ok {
		return false
	}
	if _, ok := pkg.DevDependencies["@nestjs/common"]; ok {
		return false
	}
	_, hasBuild := pkg.Scripts["build"]
	start, hasStart := pkg.Scripts["start"]
	_, hasStartProd := pkg.Scripts["start:prod"]
	if !hasBuild {
		return false
	}
	if !hasStart && !hasStartProd {
		return true
	}
	if !hasStart && hasStartProd {
		return false
	}
	startCmd := strings.ToLower(start)
	for _, re := range staticSPADevStarts {
		if re.MatchString(startCmd) {
			return true
		}
	}
	return false
}

func (w *Worker) buildDockerfile(ctx context.Context, dep *deploydomain.Deployment, spec runSpec, srcDir, tag string) (string, error) {
	dockerfile := spec.Dockerfile
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}
	if !filepath.IsAbs(dockerfile) {
		dockerfile = filepath.Join(srcDir, dockerfile)
	}
	if _, err := os.Stat(dockerfile); err != nil {
		return "", fmt.Errorf("Dockerfile not found em %s", dockerfile)
	}
	w.writeBuildEnv(srcDir, spec.Env)
	w.appendLog(dep, "building image (Dockerfile) "+tag)
	builder := w.Builder
	if builder == nil {
		builder = w.Runtime
	}
	var out string
	var err error
	streamed := false
	if streaming, ok := builder.(StreamingImageBuildRuntime); ok {
		streamed = true
		out, err = streaming.BuildStream(ctx, srcDir, dockerfile, tag, func(line string) { w.appendLog(dep, line) })
	} else {
		out, err = builder.Build(ctx, srcDir, dockerfile, tag)
	}
	if err != nil {
		if !streamed {
			w.appendLog(dep, out)
		}
		return "", err
	}
	if !streamed {
		if trimmed := strings.TrimSpace(out); trimmed != "" {
			w.appendLog(dep, trimmed)
		}
	}
	return tag, nil
}

func (w *Worker) buildCommandSource(ctx context.Context, dep *deploydomain.Deployment, spec runSpec, srcDir, tag string) (string, error) {
	dockerfile, nginxConf := generateCommandDockerfile(spec, srcDir)
	if err := os.WriteFile(filepath.Join(srcDir, "Dockerfile"), []byte(dockerfile), 0o644); err != nil {
		return "", err
	}
	if nginxConf != "" {
		if err := os.WriteFile(filepath.Join(srcDir, "nginx.conf"), []byte(nginxConf), 0o644); err != nil {
			return "", err
		}
	}
	w.writeBuildEnv(srcDir, spec.Env)
	df := filepath.Join(srcDir, "Dockerfile")
	w.appendLog(dep, "building image (install/build + nginx) "+tag)
	builder := w.Builder
	if builder == nil {
		builder = w.Runtime
	}
	var out string
	var err error
	streamed := false
	if streaming, ok := builder.(StreamingImageBuildRuntime); ok {
		streamed = true
		out, err = streaming.BuildStream(ctx, srcDir, df, tag, func(line string) { w.appendLog(dep, line) })
	} else {
		out, err = builder.Build(ctx, srcDir, df, tag)
	}
	if err != nil {
		if !streamed {
			w.appendLog(dep, out)
		}
		return "", err
	}
	if !streamed {
		if trimmed := strings.TrimSpace(out); trimmed != "" {
			w.appendLog(dep, trimmed)
		}
	}
	return tag, nil
}

func (w *Worker) writeBuildEnv(srcDir string, env []string) {
	if len(env) == 0 {
		return
	}
	var sb strings.Builder
	for _, e := range env {
		sb.WriteString(e + "\n")
	}
	_ = os.WriteFile(filepath.Join(srcDir, ".env"), []byte(sb.String()), 0o600)
}

func (w *Worker) setStatus(ctx context.Context, dep *deploydomain.Deployment, status deploydomain.Status, imageRef, containerID string) error {
	if err := dep.Transition(status); err != nil {
		w.log(ctx, "transition "+string(status), err)
		return err
	}
	dep.ImageRef = imageRef
	dep.ContainerID = containerID
	if err := w.Store.UpdateStatus(ctx, dep.ID, dep.Status, dep.Error, dep.ImageRef, dep.ContainerID, dep.StartedAt, dep.FinishedAt); err != nil {
		w.log(ctx, "persist status "+string(status), err)
		return err
	}
	w.notify(ctx, dep, status)
	return nil
}

func (w *Worker) notify(ctx context.Context, dep *deploydomain.Deployment, status deploydomain.Status) {
	if w.Notifier == nil {
		return
	}
	w.Notifier.NotifyDeploy(ctx, deploydomain.DeployEvent{
		AppID: dep.AppID, ServiceID: dep.ServiceID, DepID: dep.ID, Status: string(status), Detail: dep.Error,
	})
}

func (w *Worker) fail(ctx context.Context, dep *deploydomain.Deployment, containerID string, cause error) error {
	if errors.Is(ctx.Err(), context.Canceled) {
		if containerID != "" {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = w.Runtime.Remove(cleanupCtx, containerID)
			cancel()
		}
		return nil
	}
	statusCtx := ctx
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		cause = errors.New("deployment timed out after 20 minutes")
		var cancel context.CancelFunc
		statusCtx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}
	dep.Error = cause.Error()
	w.appendLog(dep, "fail: "+cause.Error())
	if containerID != "" {
		_ = w.Runtime.Remove(statusCtx, containerID)
	}
	return w.setStatus(statusCtx, dep, deploydomain.StatusFailed, dep.ImageRef, "")
}

func (w *Worker) appendLog(dep *deploydomain.Deployment, line string) {
	if line != "" && w.LogNotifier != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if notifier, ok := w.LogNotifier.(ServiceLogNotifier); ok && dep.ServiceID != uuid.Nil {
			notifier.NotifyServiceDeployLog(ctx, dep.ServiceID, dep.AppID, dep.ID, line)
		} else {
			w.LogNotifier.NotifyDeployLog(ctx, dep.AppID, dep.ID, line)
		}
		cancel()
	}
	if w.LogsDir == "" {
		return
	}
	dir := filepath.Join(w.LogsDir, "deployments")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, dep.ID.String()+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s %s\n", time.Now().UTC().Format(time.RFC3339), line)
}

func (w *Worker) log(ctx context.Context, msg string, err error) {
	if w.Logger != nil {
		w.Logger.Error(msg, "err", err)
	}
}

func (w *Worker) logInfo(ctx context.Context, msg string, args ...any) {
	if w.Logger != nil {
		w.Logger.InfoContext(ctx, msg, args...)
	}
}

type healthCheck struct {
	Enabled   bool   `json:"enabled"`
	Path      string `json:"path"`
	TimeoutMS int    `json:"timeout_ms"`
	Retries   int    `json:"retries"`
}

type runSpec struct {
	Name          string      `json:"name"`
	Image         string      `json:"image"`
	GitURL        string      `json:"git_url"`
	GitBranch     string      `json:"git_branch"`
	UploadID      string      `json:"upload_id"`
	BuildType     string      `json:"build_type"`
	Dockerfile    string      `json:"dockerfile"`
	ComposeFile   string      `json:"compose_file"`
	InstallCmd    string      `json:"install_command"`
	BuildCmd      string      `json:"build_command"`
	StartCmd      string      `json:"start_command"`
	RootFolder    string      `json:"root_folder"`
	DistFolder    string      `json:"dist_folder"`
	Env           []string    `json:"env"`
	Port          int         `json:"port"`
	ContainerPort int         `json:"container_port"`
	Network       string      `json:"network"`
	NetworkAlias  string      `json:"network_alias"`
	MemMB         int         `json:"mem_mb"`
	CPUs          string      `json:"cpus"`
	StorageMB     int         `json:"storage_mb"`
	HealthCheck   healthCheck `json:"health_check"`
}

func buildSpec(dep *deploydomain.Deployment) (runSpec, healthCheck, error) {
	var spec runSpec
	if len(dep.DeploySpec) > 0 {
		if err := json.Unmarshal(dep.DeploySpec, &spec); err != nil {
			return runSpec{}, healthCheck{}, err
		}
	}
	if spec.Image == "" {
		spec.Image = dep.ImageRef
	}
	if spec.Image == "" && spec.GitURL == "" && spec.UploadID == "" {
		return runSpec{}, healthCheck{}, deploydomain.ErrValidation
	}
	if spec.HealthCheck.Path == "" {
		spec.HealthCheck.Path = "/"
	}
	if spec.ContainerPort == 0 && spec.BuildType == "custom" {
		spec.ContainerPort = 80
	}
	spec.Name = "aether-" + dep.ID.String()[:8] + "-" + strconv.Itoa(dep.Number)
	var vars map[string]string
	if len(dep.EnvSnapshot) > 0 && json.Unmarshal(dep.EnvSnapshot, &vars) == nil {
		env := make([]string, 0, 8)
		for k, v := range vars {
			env = append(env, k+"="+v)
		}
		spec.Env = env
	}
	if spec.Port > 0 {
		hasPORT := false
		for _, e := range spec.Env {
			if strings.HasPrefix(e, "PORT=") {
				hasPORT = true
				break
			}
		}
		if !hasPORT {
			spec.Env = append(spec.Env, "PORT="+strconv.Itoa(spec.Port))
		}
	}
	return spec, spec.HealthCheck, nil
}

func generateCommandDockerfile(spec runSpec, srcDir string) (string, string) {
	install := strings.TrimSpace(spec.InstallCmd)
	if install == "" {
		install = "npm install"
	}
	build := strings.TrimSpace(spec.BuildCmd)
	if build == "" {
		build = "npm run build"
	}
	dist := strings.TrimSpace(strings.TrimPrefix(spec.DistFolder, "./"))
	if dist == "" {
		dist = "dist"
		if plan, err := planner.Detect(srcDir); err == nil && plan.OutputDir != "" {
			dist = strings.Trim(strings.TrimPrefix(plan.OutputDir, "./"), "/")
		}
	}
	dist = strings.Trim(dist, "/")

	nginx := "server {\n" +
		"    listen 80;\n" +
		"    server_name _;\n" +
		"    root /usr/share/nginx/html;\n" +
		"    index index.html;\n" +
		"    location ~ /\\.env { deny all; }\n" +
		"    location / {\n" +
		"        try_files $uri $uri/ /index.html;\n" +
		"    }\n" +
		"}\n"

	var sb strings.Builder
	sb.WriteString("# syntax=docker/dockerfile:1\n")
	sb.WriteString("FROM node:22-alpine AS build\n")
	sb.WriteString("WORKDIR /app\n")
	sb.WriteString("COPY package*.json ./\n")
	sb.WriteString("RUN " + install + "\n")
	sb.WriteString("COPY . .\n")
	sb.WriteString("RUN " + build + "\n")
	sb.WriteString("\nFROM nginx:alpine\n")
	sb.WriteString("COPY nginx.conf /etc/nginx/conf.d/default.conf\n")
	sb.WriteString("COPY --from=build /app/" + dist + " /usr/share/nginx/html\n")
	sb.WriteString("EXPOSE 80\n")
	sb.WriteString("CMD [\"nginx\", \"-g\", \"daemon off;\"]\n")
	return sb.String(), nginx
}

func shellJoin(cmd string) string {
	return strings.ReplaceAll(cmd, "\"", "\\\"")
}
