package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"aether/internal/domain"
	"aether/internal/druntime/queue"
	"aether/internal/events"
	"aether/internal/git"
	netx "aether/internal/net"
	"aether/internal/obs"
	"aether/internal/runtime"
	"aether/internal/runtime/compose"
)

type DeployOpts struct {
	Trigger       string
	TriggeredBy   string
	ImageOverride string
	Commit        string
	SkipBuild     bool
}

func (c *Core) Deploy(appID string, opts DeployOpts) (*domain.Deployment, error) {
	app, err := c.Store.GetApp(appID)
	if err != nil {
		return nil, err
	}
	if opts.Trigger == "" {
		opts.Trigger = "manual"
	}
	number, err := c.Store.NextDeploymentNumber(appID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	envSnap := ""
	if env, err := c.EnsureAppEnv(appID); err == nil {
		envSnap = strings.Join(env, "\n")
	}
	dep := &domain.Deployment{
		ID:          domain.NewID(),
		AppID:       appID,
		Number:      number,
		Status:      domain.DeploymentQueued,
		Trigger:     opts.Trigger,
		TriggeredBy: opts.TriggeredBy,
		EnvSnapshot: envSnap,
		ImageRef:    opts.ImageOverride,
		CreatedAt:   now,
		StartedAt:   now,
	}
	// Deployment Spec First: gera e armazena o docker-compose.yml no momento
	// do deploy (histórico para rollback/diff/auditoria).
	if composeYAML, cerr := c.GenerateCompose(app); cerr == nil {
		dep.ComposeYAML = composeYAML
		dep.ComposeHash = c.ComposeHash(composeYAML)
	}
	if spec, serr := c.AppToSpec(app, dep.Number); serr == nil {
		if b, jerr := json.Marshal(spec); jerr == nil {
			dep.DeploySpec = string(b)
		}
	}
	if err := c.Store.CreateDeployment(dep); err != nil {
		return nil, err
	}
	payload := map[string]any{
		"app_id":        appID,
		"deployment_id": dep.ID,
		"number":        number,
		"trigger":       opts.Trigger,
		"image":         opts.ImageOverride,
		"commit":        opts.Commit,
		"skip_build":    opts.SkipBuild,
	}
	if err := c.Bus.Publish(context.Background(), "deployment", dep.ID, "deployment.created", payload, nil); err != nil {
		return nil, err
	}
	return dep, nil
}

func (c *Core) Rollback(appID string) (*domain.Deployment, error) {
	return c.RollbackBy(appID, "")
}

func (c *Core) RollbackBy(appID, triggeredBy string) (*domain.Deployment, error) {
	dep, err := c.Store.LastReadyDeployment(appID, 1<<62)
	if err != nil {
		return nil, errors.New("nenhum deployment pronto para rollback")
	}
	prev, err := c.Store.LastReadyDeployment(appID, dep.Number)
	if err != nil {
		return nil, errors.New("nenhum deployment anterior pronto")
	}
	return c.Deploy(appID, DeployOpts{
		Trigger:       "rollback",
		TriggeredBy:   triggeredBy,
		ImageOverride: prev.ImageRef,
		Commit:        prev.Commit,
		SkipBuild:     true,
	})
}

func (c *Core) onDeploymentCreated(ctx context.Context, e events.Event) {
	if e.Type != "deployment.created" {
		return
	}
	var payload struct {
		DeploymentID string `json:"deployment_id"`
		AppID        string `json:"app_id"`
	}
	if err := jsonUnmarshal(e.Payload, &payload); err != nil || payload.DeploymentID == "" {
		return
	}
	if err := c.RT.Queue.Enqueue(context.Background(), deployQueueStream, queue.Job{
		Type:         "deploy",
		AppID:        payload.AppID,
		DeploymentID: payload.DeploymentID,
	}); err != nil {
		log.Printf("[deploy] enqueue %s falhou: %v", payload.DeploymentID, err)
		go c.runPipeline(payload.DeploymentID)
		return
	}
	c.ensureDeployWorkers()
}

const deployQueueStream = "deploys"
const deployWorkerCount = 4

func (c *Core) ensureDeployWorkers() {
	c.deployWkMu.Lock()
	defer c.deployWkMu.Unlock()
	if c.deployWorkersStarted {
		return
	}
	c.deployWorkersStarted = true
	c.deployCtx, c.deployCancel = context.WithCancel(context.Background())
	for w := 0; w < deployWorkerCount; w++ {
		c.deployWg.Add(1)
		go c.runDeployWorker(c.deployCtx, strconv.Itoa(w))
	}
	log.Printf("[deploy] worker pool iniciado (%d workers, queue=%s)", deployWorkerCount, deployQueueStream)
}

func (c *Core) runDeployWorker(ctx context.Context, id string) {
	defer c.deployWg.Done()
	consumer, err := c.RT.Queue.NewConsumer(ctx, deployQueueStream, "workers", "deploy-"+id)
	if err != nil {
		log.Printf("[deploy] worker %s: %v", id, err)
		return
	}
	for {
		job, err := consumer.Next(ctx)
		if err != nil {
			return
		}
		if job.DeploymentID == "" {
			_ = consumer.Ack(ctx, job)
			continue
		}
		c.runPipeline(job.DeploymentID)
		_ = consumer.Ack(ctx, job)
	}
}

func (c *Core) stopDeployWorkers() {
	c.deployWkMu.Lock()
	defer c.deployWkMu.Unlock()
	if c.deployCancel != nil {
		c.deployCancel()
	}
}

func (c *Core) runPipeline(deploymentID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	dep, err := c.Store.GetDeployment(deploymentID)
	if err != nil {
		return
	}
	if dep.Status == domain.DeploymentReady || dep.Status == domain.DeploymentFailed || dep.Status == domain.DeploymentCancelled {
		return
	}
	app, err := c.Store.GetApp(dep.AppID)
	if err != nil {
		return
	}
	lockName := "lock:app:" + app.ID
	lockCtx, lockCancel := context.WithTimeout(context.Background(), 15*time.Second)
	lock, locked, lerr := c.RT.Locks.Acquire(lockCtx, lockName, lockDeployTTL)
	lockCancel()
	if lerr != nil {
		log.Printf("[deploy] lock %s: %v", lockName, lerr)
		return
	}
	if !locked {
		log.Printf("[deploy] app %s já em deploy em outra instância — cancelando %s", app.ID, deploymentID)
		c.setDeploymentStatus(dep, domain.DeploymentCancelled)
		return
	}
	defer func() {
		rctx, rcancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer rcancel()
		_ = c.RT.Locks.Release(rctx, lock)
	}()
	if err := c.cancelOtherActive(app.ID, dep.ID); err != nil {
		log.Printf("[deploy] cancelamento de deployments anteriores falhou: %v", err)
	}
	c.activeMu.Lock()
	c.active[app.ID] = dep.ID
	c.activeMu.Unlock()
	defer func() {
		c.activeMu.Lock()
		if c.active[app.ID] == dep.ID {
			delete(c.active, app.ID)
		}
		c.activeMu.Unlock()
	}()

	liveLog, err := c.openLiveLog(app, dep)
	if err != nil {
		c.failDeployment(dep, err)
		return
	}

	if target, err := c.placeOnAgent(app); target != "" {
		if err != nil {
			c.failDeploymentLog(dep, liveLog, err)
			liveLog.Close()
			return
		}
		dep.ServerID = target
		_ = c.Store.UpdateDeployment(dep)
		c.FireDeployStarted(app, dep)
		liveLog.Write([]byte("[scheduler] deploy remoto -> " + target + "\n"))
		c.runRemoteDeploy(ctx, app, dep, target, liveLog)
		return
	}
	c.setDeploymentStatus(dep, domain.DeploymentBuilding)
	c.FireDeployStarted(app, dep)
	c.publishDeployEvent(app, dep, "deployment.building", map[string]any{"build_method": app.BuildType})
	if err := c.resolveImage(ctx, app, dep, liveLog); err != nil {
		c.failDeploymentLog(dep, liveLog, err)
		liveLog.Close()
		return
	}

	c.setDeploymentStatus(dep, domain.DeploymentStarting)
	c.publishDeployEvent(app, dep, "deployment.starting", nil)
	// para o deployment anterior ANTES de iniciar o novo: com o bind na porta
	// configurada, o container anterior segura a porta e causaria conflito.
	if err := c.stopPrevious(ctx, app, dep); err != nil {
		log.Printf("[deploy] parada do deployment anterior falhou: %v", err)
	}
	containerID, err := c.schedule(ctx, app, dep, liveLog)
	if err != nil {
		c.failDeploymentLog(dep, liveLog, err)
		liveLog.Close()
		return
	}
	dep.ContainerID = containerID
	c.Store.UpdateDeployment(dep)

	c.setDeploymentStatus(dep, domain.DeploymentHealthChecking)
	c.publishDeployEvent(app, dep, "deployment.healthcheck", map[string]any{"path": app.HealthCheck.Path})
	if err := c.verifyContainerAlive(ctx, dep, liveLog); err != nil {
		c.cleanupContainer(ctx, containerID)
		c.failDeploymentLog(dep, liveLog, err)
		liveLog.Close()
		return
	}
	if err := c.healthCheck(ctx, app, dep, liveLog); err != nil {
		c.cleanupContainer(ctx, containerID)
		c.failDeploymentLog(dep, liveLog, err)
		liveLog.Close()
		return
	}

	c.setDeploymentStatus(dep, domain.DeploymentReady)
	c.attachRoutes(ctx, app, dep)
	c.closeOlderLogs(app)
	c.PublishAppState(app.ID, "running")
	c.RegistryPushAfterBuild(dep.ImageRef)
	c.startContainerLogCollector(app, dep)
	c.FireWebhookEvent(context.Background(), app.OrgID, EvDeployReady, map[string]any{
		"app":     app.Name,
		"app_id":  app.ID,
		"project": app.ProjectID,
		"build":   dep.Number,
		"image":   dep.ImageRef,
	})
	c.Bus.Publish(context.Background(), "app", app.ID, "app.deployed", map[string]any{
		"deployment_id": dep.ID,
		"number":        dep.Number,
		"image":         dep.ImageRef,
	}, nil)
	if dep.Trigger == "rollback" {
		c.Bus.Publish(context.Background(), "app", app.ID, "app.rolled_back", map[string]any{
			"deployment_id": dep.ID,
			"number":        dep.Number,
		}, nil)
	}
}

func (c *Core) cancelOtherActive(appID, exceptDeploymentID string) error {
	rows, err := c.DB.Query(`SELECT id, container_id FROM deployments WHERE app_id=? AND status IN ('queued','building','starting','health_checking') AND id<>?`, appID, exceptDeploymentID)
	if err != nil {
		return err
	}
	defer rows.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var cancelled []*domain.Deployment
	for rows.Next() {
		var id, cid string
		if err := rows.Scan(&id, &cid); err != nil {
			return err
		}
		// carrega o deployment completo para não zerar colunas (image_ref,
		// commit, env_snapshot) no UpdateDeployment.
		d, err := c.Store.GetDeployment(id)
		if err != nil {
			continue
		}
		d.ContainerID = cid
		cancelled = append(cancelled, d)
	}
	for _, d := range cancelled {
		if d.ContainerID != "" {
			c.Driver.Remove(ctx, d.ContainerID, true)
		}
		d.Status = domain.DeploymentCancelled
		d.FinishedAt = time.Now().UTC()
		c.Store.UpdateDeployment(d)
	}
	return rows.Err()
}

func (c *Core) openLiveLog(app *domain.App, dep *domain.Deployment) (*obs.LiveLog, error) {
	dir := filepath.Join(c.Cfg.LogsDir, "apps", app.Name)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, fmt.Sprintf("%d.log", dep.Number))
	ll, err := obs.NewLiveLog(path)
	if err != nil {
		return nil, err
	}
	ll.SetTailBuffer(256 * 1024)
	channel := "logs:deployment:" + dep.ID
	appChannel := "logs:app:" + app.ID
	ll.SetPublisher(func(chunk []byte) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.RT.PubSub.Publish(ctx, channel, chunk)
		_ = c.RT.PubSub.Publish(ctx, appChannel, chunk)
	})
	c.liveMu.Lock()
	c.live[dep.ID] = ll
	c.liveMu.Unlock()
	return ll, nil
}

func (c *Core) closeOlderLogs(app *domain.App) {
	deploys, err := c.Store.ListDeployments(app.ID, 10)
	if err != nil {
		return
	}
	if len(deploys) == 0 {
		return
	}
	keep := map[string]bool{}
	for i, d := range deploys {
		if i < 2 {
			keep[d.ID] = true
		}
	}
	c.liveMu.Lock()
	for id, ll := range c.live {
		if strings.HasPrefix(ll.Path(), filepath.Join(c.Cfg.LogsDir, "apps", app.Name)) && !keep[id] {
			ll.Close()
			delete(c.live, id)
		}
	}
	c.liveMu.Unlock()
}

func (c *Core) LiveLog(deploymentID string) *obs.LiveLog {
	c.liveMu.Lock()
	defer c.liveMu.Unlock()
	return c.live[deploymentID]
}

// ReadDeploymentLog lê do disco o log persistido de um deployment
// (funciona mesmo após o LiveLog em memória ser fechado ou o servidor reiniciar).
func (c *Core) ReadDeploymentLog(appName string, number int64) ([]byte, error) {
	path := filepath.Join(c.Cfg.LogsDir, "apps", appName, fmt.Sprintf("%d.log", number))
	return os.ReadFile(path)
}

func (c *Core) resolveImage(ctx context.Context, app *domain.App, dep *domain.Deployment, ll *obs.LiveLog) error {
	if app.SourceType == domain.SourceImage {
		image := dep.ImageRef
		if image == "" {
			image = app.Image
		}
		dep.ImageRef = image
		ll.Write([]byte("[deploy] imagem: " + image + "\n"))
		if err := c.Driver.Pull(ctx, image); err != nil {
			return fmt.Errorf("pull %s: %w", image, err)
		}
		return c.Store.UpdateDeployment(dep)
	}
	srcDir := filepath.Join(c.Cfg.BuildsDir, "sources", app.Name, strconv.FormatInt(dep.Number, 10))
	if err := os.RemoveAll(srcDir); err != nil {
		return err
	}
	var commit string
	if app.UploadID != "" {
		uploadDir := filepath.Join(c.Cfg.BuildsDir, "uploads", app.UploadID)
		if err := copyDir(uploadDir, srcDir); err != nil {
			return fmt.Errorf("upload: %w", err)
		}
		flattenSingleRoot(srcDir)
		ll.Write([]byte("[deploy] fonte: upload " + app.UploadID + "\n"))
		dep.Commit = "upload"
	} else {
		ll.Write([]byte("[deploy] clone " + app.GitURL + " (" + app.GitBranch + ")\n"))
		if err := git.Clone(ctx, app.GitURL, app.GitBranch, srcDir); err != nil {
			return fmt.Errorf("git clone: %w", err)
		}
		var err error
		commit, err = git.CommitHEAD(ctx, srcDir)
		if err != nil {
			return err
		}
		dep.Commit = commit
		ll.Write([]byte("[deploy] commit " + commit + "\n"))
	}
	if dep.ImageRef == "" {
		dep.ImageRef = appImageRepo(app) + ":" + strconv.FormatInt(dep.Number, 10)
	}
	if err := c.buildSource(ctx, app, srcDir, dep.ImageRef, ll); err != nil {
		return err
	}
	ll.Write([]byte("[build] concluído: " + dep.ImageRef + "\n"))
	return c.Store.UpdateDeployment(dep)
}

func (c *Core) schedule(ctx context.Context, app *domain.App, dep *domain.Deployment, ll *obs.LiveLog) (string, error) {
	env, err := c.EnsureAppEnv(app.ID)
	if err != nil {
		return "", err
	}
	network := "aether-" + app.ProjectID
	if err := c.Driver.NetworkCreate(ctx, network); err != nil {
		if !strings.Contains(err.Error(), "already exists") && !strings.Contains(err.Error(), "exists") {
			return "", fmt.Errorf("network: %w", err)
		}
	}
	var volumes []runtime.VolumeMount
	for _, v := range app.Volumes {
		volName := "aether-" + app.Name + "-" + v.Name
		if err := c.Driver.VolumeCreate(ctx, volName, app.Resources.StorageMB); err != nil {
			if !strings.Contains(err.Error(), "already exists") && !strings.Contains(err.Error(), "exists") {
				return "", fmt.Errorf("volume: %w", err)
			}
		}
		volumes = append(volumes, runtime.VolumeMount{Source: volName, Target: v.MountPath})
	}
	hostPort := strconv.Itoa(app.Port)
	if app.Port <= 0 {
		hostPort = c.randomFreePort()
	}
	containerPort := c.containerInternalPort(app, dep)
	containerName := "aether-" + app.Name + "-" + strconv.FormatInt(dep.Number, 10)
	labels := map[string]string{"aether.app": app.ID, "aether.deployment": dep.ID}
	if c.testMode {
		labels["aether.test"] = "1"
	}
	spec := runtime.RunSpec{
		Name:         containerName,
		Image:        dep.ImageRef,
		Env:          env,
		Ports:        []runtime.PortBinding{{HostPort: hostPort, ContainerPort: containerPort}},
		Networks:     []string{network},
		NetworkAlias: compose.ServiceName(app.Name),
		Volumes:      volumes,
		MemMB:        app.Resources.MemMB,
		CPUs:         app.Resources.CPUs,
		Restart:      "on-failure",
		Labels:       labels,
	}
	var containerID string
	if c.quadlet {
		serviceName := containerName + ".service"
		unitName := containerName
		if _, err := c.unitMgr.WriteUnit(unitName, spec, network); err != nil {
			return "", fmt.Errorf("unit: %w", err)
		}
		if err := c.unitMgr.Start(ctx, serviceName); err != nil {
			return "", err
		}
		containerID = containerName
	} else {
		// Deployment Spec First: executa via docker compose up.
		// Fallback para o driver (docker run) se o compose não estiver disponível.
		if compSpec, cerr := c.DeploymentCompose(app, dep); cerr == nil {
			cid, cname, err := c.composeUp(ctx, app, dep, compSpec)
			if err == nil {
				containerID = cid
				hostPort = firstHostPort(compSpec)
				dep.ContainerID = cid
				dep.ComposeYAML, _ = compose.GenerateWith(compSpec, cname)
				dep.ComposeHash = c.ComposeHash(dep.ComposeYAML)
				_ = c.Store.UpdateDeployment(dep)
			} else {
				ll.Write([]byte("[compose] falhou (" + err.Error() + ") — fallback driver\n"))
				id, usedPort, rerr := c.runContainerRetry(ctx, spec, containerName)
				if rerr != nil {
					return "", fmt.Errorf("run: %w", rerr)
				}
				containerID = id
				hostPort = usedPort
			}
		} else {
			id, usedPort, rerr := c.runContainerRetry(ctx, spec, containerName)
			if rerr != nil {
				return "", fmt.Errorf("run: %w", rerr)
			}
			containerID = id
			hostPort = usedPort
		}
	}
	ll.Write([]byte("[deploy] container " + containerID + " iniciado (127.0.0.1:" + hostPort + ")\n"))
	go c.streamContainerLogs(ctx, dep.ID, containerID, ll)
	return containerID, nil
}

// runContainerRetry tenta iniciar o container no bind configurado; se a porta
// já estiver alocada, tenta portas livres aleatórias (até 10 tentativas).
func (c *Core) runContainerRetry(ctx context.Context, spec runtime.RunSpec, containerName string) (string, string, error) {
	attempts := []string{spec.Ports[0].HostPort}
	for i := 0; i < 10; i++ {
		if i > 0 {
			spec.Ports[0].HostPort = c.randomFreePort()
			attempts = append(attempts, spec.Ports[0].HostPort)
		}
		c.Driver.Remove(ctx, containerName, true)
		id, err := c.Driver.Run(ctx, spec)
		if err == nil {
			return id, spec.Ports[0].HostPort, nil
		}
		lower := strings.ToLower(err.Error())
		if !strings.Contains(lower, "already allocated") && !strings.Contains(lower, "port is already") {
			return "", "", err
		}
	}
	return "", "", fmt.Errorf("portas indisponíveis (tentadas: %s)", strings.Join(attempts, ", "))
}

// randomFreePort retorna uma porta livre aleatória na faixa 20000-39999.
func (c *Core) randomFreePort() string {
	for i := 0; i < 100; i++ {
		p := 20000 + rand.Intn(20000)
		if netx.PortFree(p) {
			return strconv.Itoa(p)
		}
	}
	return "0"
}

// containerInternalPort resolve a porta em que o processo escuta DENTRO do
// container: nginx (web estático/SPA) → 80; senão o EXPOSE do Dockerfile do
// source (se existir); por último a porta configurada.
func (c *Core) containerInternalPort(app *domain.App, dep *domain.Deployment) string {
	if plan, err := c.Store.GetDeploymentPlan(app.ID); err == nil && plan.WebServer == "nginx" {
		return "80"
	}
	srcDir := filepath.Join(c.Cfg.BuildsDir, "sources", app.Name, strconv.FormatInt(dep.Number, 10))
	if p := readExposePort(filepath.Join(srcDir, app.Dockerfile)); p != "" {
		return p
	}
	return strconv.Itoa(app.Port)
}

// readExposePort extrai a primeira porta do EXPOSE no Dockerfile.
func readExposePort(dfPath string) string {
	data, err := os.ReadFile(dfPath)
	if err != nil {
		return ""
	}
	m := regexp.MustCompile(`(?m)^\s*EXPOSE\s+(\d{2,5})`).FindStringSubmatch(string(data))
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func (c *Core) startContainerLogCollector(app *domain.App, dep *domain.Deployment) {
	if dep.ContainerID == "" {
		return
	}
	channel := "logs:deployment:" + dep.ID
	appChannel := "logs:app:" + app.ID
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			stream, err := c.Driver.Logs(ctx, dep.ContainerID, true)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Second):
					continue
				}
			}
			buf := make([]byte, 32*1024)
			for {
				n, rerr := stream.Read(buf)
				if n > 0 {
					_ = c.RT.PubSub.Publish(ctx, channel, buf[:n])
					_ = c.RT.PubSub.Publish(ctx, appChannel, buf[:n])
				}
				if rerr != nil {
					stream.Close()
					break
				}
			}
			if ctx.Err() != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}()
	c.collectorMu.Lock()
	if prev, ok := c.collectors[app.ID]; ok {
		prev.cancel()
		<-prev.done
	}
	c.collectors[app.ID] = &logCollector{cancel: cancel, done: done}
	c.collectorMu.Unlock()
}

func (c *Core) StopAppLogCollector(appID string) {
	c.stopContainerLogCollector(appID)
}

func (c *Core) stopContainerLogCollector(appID string) {
	c.collectorMu.Lock()
	prev, ok := c.collectors[appID]
	delete(c.collectors, appID)
	c.collectorMu.Unlock()
	if ok {
		prev.cancel()
		<-prev.done
	}
}

type logCollector struct {
	cancel context.CancelFunc
	done   <-chan struct{}
}

func (c *Core) reconcileLogCollectors(ctx context.Context) {
	rows, err := c.DB.Query(`SELECT id, app_id, container_id FROM deployments WHERE status='ready' AND container_id<>''`)
	if err != nil {
		return
	}
	defer rows.Close()
	type row struct{ id, appID, cid string }
	var found []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.appID, &r.cid); err != nil {
			continue
		}
		found = append(found, r)
	}
	for _, r := range found {
		app, err := c.Store.GetApp(r.appID)
		if err != nil {
			continue
		}
		c.startContainerLogCollector(app, &domain.Deployment{ID: r.id, ContainerID: r.cid})
	}
}

func (c *Core) streamContainerLogs(ctx context.Context, deploymentID, containerID string, ll *obs.LiveLog) {
	stream, err := c.Driver.Logs(ctx, containerID, true)
	if err != nil {
		ll.Write([]byte("[logs] stream indisponível: " + err.Error() + "\n"))
		return
	}
	defer stream.Close()
	buf := make([]byte, 32*1024)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			ll.Write(buf[:n])
		}
		if err != nil {
			if err != io.EOF {
				ll.Write([]byte("[logs] fim: " + err.Error() + "\n"))
			}
			return
		}
	}
}

func (c *Core) verifyContainerAlive(ctx context.Context, dep *domain.Deployment, ll *obs.LiveLog) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1200 * time.Millisecond):
	}
	info, err := c.Driver.Inspect(ctx, dep.ContainerID)
	if err != nil {
		return fmt.Errorf("inspeção do container falhou: %w", err)
	}
	if info.State != "running" {
		ll.Write([]byte(fmt.Sprintf("[health] container parou após iniciar (estado: %s)\n", info.State)))
		return fmt.Errorf("container parou após iniciar (estado: %s)", info.State)
	}
	return nil
}

func (c *Core) healthCheck(ctx context.Context, app *domain.App, dep *domain.Deployment, ll *obs.LiveLog) error {
	if !app.HealthCheck.Enabled {
		return nil
	}
	port := c.containerPort(ctx, dep)
	if port == "" {
		return errors.New("health check: porta do container não resolvida")
	}
	url := "http://127.0.0.1:" + port + app.HealthCheck.Path
	client := &http.Client{Timeout: time.Duration(app.HealthCheck.TimeoutMS) * time.Millisecond}
	interval := time.Duration(app.HealthCheck.IntervalMS) * time.Millisecond
	if interval < 500*time.Millisecond {
		interval = 500 * time.Millisecond
	}
	maxAttempts := app.HealthCheck.Retries
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ll.Write([]byte(fmt.Sprintf("[health] tentativa %d/%d %s\n", attempt, maxAttempts, url)))
		resp, err := client.Get(url)
		if err == nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				ll.Write([]byte(fmt.Sprintf("[health] ok status=%d\n", resp.StatusCode)))
				return nil
			}
			ll.Write([]byte(fmt.Sprintf("[health] status=%d body=%s\n", resp.StatusCode, strings.TrimSpace(string(body)))))
		} else {
			ll.Write([]byte("[health] erro: " + err.Error() + "\n"))
		}
		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(interval):
			}
		}
	}
	return errors.New("health check falhou após " + strconv.Itoa(maxAttempts) + " tentativas")
}

func (c *Core) stopPrevious(ctx context.Context, app *domain.App, dep *domain.Deployment) error {
	prev, err := c.Store.LastReadyDeployment(app.ID, dep.Number)
	if err != nil {
		return nil
	}
	if prev.ContainerID == "" {
		return nil
	}
	return c.Driver.Remove(ctx, prev.ContainerID, true)
}

func (c *Core) cleanupContainer(ctx context.Context, containerID string) {
	if containerID == "" {
		return
	}
	c.Driver.Remove(ctx, containerID, true)
}

func (c *Core) setDeploymentStatus(dep *domain.Deployment, status domain.DeploymentStatus) {
	dep.Status = status
	if status == domain.DeploymentReady || status == domain.DeploymentFailed ||
		status == domain.DeploymentCancelled || status == domain.DeploymentRolledBack {
		dep.FinishedAt = time.Now().UTC()
	}
	c.Store.UpdateDeployment(dep)
	if status == domain.DeploymentReady || status == domain.DeploymentFailed {
		if app, err := c.Store.GetApp(dep.AppID); err == nil {
			c.Audit(app.OrgID, dep.TriggeredBy, "deployment."+string(status), "deployment", dep.ID, app.Name+" #"+strconv.FormatInt(dep.Number, 10))
		}
	}
}

func (c *Core) failDeployment(dep *domain.Deployment, err error) {
	dep.Status = domain.DeploymentFailed
	dep.Error = err.Error()
	dep.FinishedAt = time.Now().UTC()
	c.Store.UpdateDeployment(dep)
	app, aerr := c.Store.GetApp(dep.AppID)
	if aerr != nil {
		return
	}
	c.Bus.Publish(context.Background(), "app", app.ID, "app.deploy_failed", map[string]any{
		"deployment_id": dep.ID,
		"number":        dep.Number,
		"error":         err.Error(),
	}, nil)
	c.FireWebhookEvent(context.Background(), app.OrgID, EvDeployFailed, map[string]any{
		"app":    app.Name,
		"app_id": app.ID,
		"build":  dep.Number,
		"error":  err.Error(),
	})
	c.NotifyOrg(app.OrgID, "Deploy failed: "+app.Name, fmt.Sprintf("#%d: %s", dep.Number, err.Error()))
	log.Printf("[deploy] %s #%d falhou: %v", app.Name, dep.Number, err)
}

// failDeploymentLog grava o erro no LiveLog do deployment antes de marcá-lo
// como falho, para que o log completo do build apareça na tela do app.
func (c *Core) failDeploymentLog(dep *domain.Deployment, ll *obs.LiveLog, err error) {
	if ll != nil {
		ll.Write([]byte("\n[build] FALHOU: " + err.Error() + "\n"))
	}
	c.failDeployment(dep, err)
}

func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
