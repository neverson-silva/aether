package application

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/databases/domain"
	deploydomain "aether/internal/modules/deployments/domain"
	"aether/internal/platform/worker"
)

type ContainerRuntime interface {
	Run(ctx context.Context, spec worker.RunSpec) (string, error)
	Start(ctx context.Context, containerID string) error
	Stop(ctx context.Context, containerID string) error
	Restart(ctx context.Context, containerID string) error
	Remove(ctx context.Context, containerID string) error
	RemoveByLabel(ctx context.Context, label string) error
	ContainerState(ctx context.Context, containerID string) (string, error)
	LogTail(ctx context.Context, containerID string, lines int) ([]string, error)
	Exec(ctx context.Context, containerID string, env []string, args ...string) (string, string, error)
}

var dbImageRepositories = map[domain.Engine]string{
	domain.EnginePostgres: "docker.io/postgres",
	domain.EngineMysql:    "docker.io/mysql",
	domain.EngineMariaDB:  "docker.io/mariadb",
	domain.EngineRedis:    "docker.io/redis",
	domain.EngineMongoDB:  "docker.io/mongo",
	domain.EngineMSSQL:    "mcr.microsoft.com/mssql/server",
	domain.EngineOracle:   "gvenzl/oracle-free",
}

func dbImage(engine domain.Engine, version string) string {
	repository := dbImageRepositories[engine]
	if repository == "" {
		return ""
	}
	if version == "" {
		version = defaultVersions[engine]
	}
	return repository + ":" + version
}

func dbEnv(db *domain.Database, pass string) []string {
	switch db.Engine {
	case domain.EnginePostgres:
		return []string{
			"POSTGRES_USER=" + db.User,
			"POSTGRES_PASSWORD=" + pass,
			"POSTGRES_DB=" + db.DBName,
		}
	case domain.EngineMysql, domain.EngineMariaDB:
		return []string{
			"MYSQL_ROOT_PASSWORD=" + pass,
			"MYSQL_DATABASE=" + db.DBName,
			"MYSQL_USER=" + db.User,
			"MYSQL_PASSWORD=" + pass,
		}
	case domain.EngineMongoDB:
		return []string{
			"MONGO_INITDB_ROOT_USERNAME=" + db.User,
			"MONGO_INITDB_ROOT_PASSWORD=" + pass,
			"MONGO_INITDB_DATABASE=" + db.DBName,
		}
	case domain.EngineMSSQL:
		return []string{
			"ACCEPT_EULA=Y",
			"MSSQL_SA_PASSWORD=" + pass,
		}
	case domain.EngineOracle:
		return []string{
			"ORACLE_PASSWORD=" + pass,
			"ORACLE_DATABASE=" + db.DBName,
		}
	case domain.EngineRedis:
		return nil
	default:
		return nil
	}
}

func (d *Databases) deploy(ctx context.Context, db *domain.Database) (string, error) {
	if d.Runtime == nil {
		return "", errors.New("database runtime not configured")
	}
	image := dbImage(db.Engine, db.Version)
	if image == "" {
		return "", fmt.Errorf("%w: unsupported engine %s", domain.ErrValidation, db.Engine)
	}
	if puller, ok := d.Runtime.(interface {
		Pull(context.Context, string) (string, error)
	}); ok {
		if _, err := puller.Pull(ctx, image); err != nil {
			return "", fmt.Errorf("pull database image %q: %w", image, err)
		}
	}
	pass, err := d.Passwords.Decrypt(db.PassEnc)
	if err != nil {
		return "", domain.ErrValidation
	}
	if db.ContainerID != "" {
		_ = d.Runtime.Remove(ctx, db.ContainerID)
	}
	_ = d.Runtime.Remove(ctx, "db-"+db.Name)
	containerPort := defaultPorts[db.Engine]
	if containerPort == 0 {
		containerPort = db.Port
	}
	hostPort := d.allocHostPort(db.Port)
	if hostPort == 0 {
		return "", errors.New("no free host port available")
	}
	if hostPort != db.Port {
		if err := d.Store.UpdateDatabasePort(ctx, db.ID, hostPort); err != nil {
			return "", err
		}
		db.Port = hostPort
	}
	serviceID := db.ServiceID
	if serviceID == uuid.Nil {
		serviceID = db.ID
	}
	spec := worker.RunSpec{
		Name:          "db-" + db.Name,
		Image:         image,
		Env:           d.runtimeEnv(ctx, db, pass),
		Port:          hostPort,
		ContainerPort: containerPort,
		Network:       d.Network,
		NetworkAlias:  "db-" + db.ID.String()[:8],
		MemMB:         db.MemMB,
		Labels: map[string]string{
			"aether.owner":        "user",
			"aether.service-type": "database",
			"aether.service-id":   serviceID.String(),
			"aether.service-name": db.Name,
			"aether.project-id":   db.ProjectID.String(),
			"aether.database-id":  db.ID.String(),
		},
	}
	containerID, err := d.Runtime.Run(ctx, spec)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(containerID), nil
}

func (d *Databases) runtimeEnv(ctx context.Context, db *domain.Database, pass string) []string {
	values := map[string]string{}
	if d.Variables != nil && db.ServiceID != uuid.Nil {
		if resolved, err := d.Variables.Effective(ctx, db.ServiceID, db.OrgID); err == nil {
			for key, value := range resolved {
				values[key] = value
			}
		}
	}
	for _, item := range dbEnv(db, pass) {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			values[key] = value
		}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	env := make([]string, 0, len(keys))
	for _, key := range keys {
		env = append(env, key+"="+values[key])
	}
	return env
}

func hostPortFree(port int) bool {
	if port <= 0 {
		return false
	}
	for _, host := range []string{"host.containers.internal", "127.0.0.1"} {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 300*time.Millisecond)
		if err == nil {
			conn.Close()
			return false
		}
	}
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}

const databaseHealthTimeout = 120 * time.Second

func (d *Databases) waitHealthy(ctx context.Context, db *domain.Database, containerPort int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	targets := []struct {
		host string
		port int
	}{
		{host: "db-" + db.ID.String()[:8], port: containerPort},
		{host: "host.docker.internal", port: db.Port},
		{host: "host.containers.internal", port: db.Port},
		{host: "127.0.0.1", port: db.Port},
	}
	for {
		var lastErr error
		for _, target := range targets {
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", target.host, target.port), 2*time.Second)
			if err == nil {
				conn.Close()
				return nil
			}
			lastErr = err
		}
		if time.Now().After(deadline) {
			return lastErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func (d *Databases) allocHostPort(stored int) int {
	start := stored
	if start <= 0 {
		return 0
	}
	limit := start + 1000
	if limit > 65535 {
		limit = 65535
	}
	for p := start; p <= limit; p++ {
		if hostPortFree(p) {
			return p
		}
	}
	return 0
}

func (d *Databases) Deploy(ctx context.Context, id, orgID uuid.UUID) (*domain.Database, error) {
	return d.deployWithTrigger(ctx, id, orgID, "deploy", true)
}

func (d *Databases) Rebuild(ctx context.Context, id, orgID uuid.UUID) (*domain.Database, error) {
	return d.deployWithTrigger(ctx, id, orgID, "rebuild", true)
}

func (d *Databases) DeployForWorker(ctx context.Context, id, orgID uuid.UUID) (*domain.Database, error) {
	return d.deployWithTrigger(ctx, id, orgID, "deploy", false)
}

func (d *Databases) deployWithTrigger(ctx context.Context, id, orgID uuid.UUID, trigger string, record bool) (*domain.Database, error) {
	db, err := d.Get(ctx, id, orgID)
	if err != nil {
		return nil, err
	}
	var dep *deploydomain.Deployment
	if record {
		dep = d.recordDeployment(ctx, db, trigger, deploydomain.StatusStarting, "", "")
	}
	if dep != nil && d.Notifier != nil {
		d.Notifier.NotifyDeploy(ctx, deploydomain.DeployEvent{AppID: dep.AppID, ServiceID: dep.ServiceID, DepID: dep.ID, Status: string(deploydomain.StatusStarting), Detail: "Database deployment started"})
	}
	deploymentID := uuid.Nil
	if dep != nil {
		deploymentID = dep.ID
	}
	d.appendDeployLog(ctx, deploymentID, "Deploying database '"+db.Name+"' ("+string(db.Engine)+")")
	d.appendDeployLog(ctx, deploymentID, "Pulling image "+dbImage(db.Engine, db.Version))
	containerID, err := d.deploy(ctx, db)
	if err != nil {
		d.appendDeployLog(ctx, deploymentID, "Deploy failed: "+err.Error())
		if dep != nil {
			d.finishDeployment(ctx, dep.ID, deploydomain.StatusFailed, "", err.Error())
			d.notifyDeployment(ctx, dep, deploydomain.StatusFailed, err.Error())
		}
		_ = d.Store.UpdateDatabaseStatus(ctx, id, "failed", db.ContainerID)
		return nil, err
	}
	d.appendDeployLog(ctx, deploymentID, "Container started: "+containerID)
	_ = d.Store.UpdateDatabaseStatus(ctx, id, "starting", containerID)
	containerPort := defaultPorts[db.Engine]
	if containerPort == 0 {
		containerPort = db.Port
	}
	if err := d.waitHealthy(ctx, db, containerPort, databaseHealthTimeout); err != nil {
		_ = d.Runtime.Remove(ctx, containerID)
		d.appendDeployLog(ctx, deploymentID, "Health check failed: "+err.Error())
		if dep != nil {
			d.finishDeployment(ctx, dep.ID, deploydomain.StatusFailed, containerID, err.Error())
			d.notifyDeployment(ctx, dep, deploydomain.StatusFailed, err.Error())
		}
		_ = d.Store.UpdateDatabaseStatus(ctx, id, "failed", containerID)
		return nil, fmt.Errorf("database did not become healthy: %w", err)
	}
	d.appendDeployLog(ctx, deploymentID, "Database is healthy on port "+strconv.Itoa(db.Port))
	if dep != nil {
		d.finishDeployment(ctx, dep.ID, deploydomain.StatusReady, containerID, "")
		d.notifyDeployment(ctx, dep, deploydomain.StatusReady, "Database is healthy")
	}
	if err := d.Store.UpdateDatabaseStatus(ctx, id, "running", containerID); err != nil {
		return nil, err
	}
	return d.Get(ctx, id, orgID)
}

func (d *Databases) notifyDeployment(ctx context.Context, dep *deploydomain.Deployment, status deploydomain.Status, detail string) {
	if d.Notifier != nil {
		d.Notifier.NotifyDeploy(ctx, deploydomain.DeployEvent{AppID: dep.AppID, ServiceID: dep.ServiceID, DepID: dep.ID, Status: string(status), Detail: detail})
	}
}

func (d *Databases) recordDeployment(ctx context.Context, db *domain.Database, trigger string, status deploydomain.Status, containerID, errMsg string) *deploydomain.Deployment {
	if d.Deployments == nil {
		return nil
	}
	number, err := d.Deployments.NextNumber(ctx, db.ID)
	if err != nil {
		return nil
	}
	dep := &deploydomain.Deployment{
		AppID: db.ID, ServiceID: db.ServiceID, Number: number, Status: status, Trigger: trigger,
		ContainerID: containerID, Error: errMsg,
	}
	created, err := d.Deployments.CreateDeployment(ctx, dep)
	if err != nil {
		return nil
	}
	return created
}

func (d *Databases) finishDeployment(ctx context.Context, depID uuid.UUID, status deploydomain.Status, containerID, errMsg string) {
	if d.Deployments == nil {
		return
	}
	now := time.Now().UTC()
	_ = d.Deployments.UpdateStatus(ctx, depID, status, errMsg, "", containerID, &now, &now)
}

func (d *Databases) appendDeployLog(ctx context.Context, depID uuid.UUID, line string) {
	worker.EmitDeploymentLog(ctx, line)
	if depID == uuid.Nil {
		return
	}
	if d.LogsDir == "" {
		return
	}
	dir := filepath.Join(d.LogsDir, "deployments")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, depID.String()+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s %s\n", time.Now().UTC().Format(time.RFC3339), line)
}

func (d *Databases) ListDeployments(ctx context.Context, id, orgID uuid.UUID, limit int) ([]deploydomain.Deployment, error) {
	if d.Deployments == nil {
		return nil, nil
	}
	db, err := d.Get(ctx, id, orgID)
	if err != nil {
		return nil, err
	}
	return d.Deployments.ListByApp(ctx, db.ID, limit)
}

func (d *Databases) DeploymentLogs(ctx context.Context, dbID, depID, orgID uuid.UUID, limit int) (string, error) {
	if _, err := d.Get(ctx, dbID, orgID); err != nil {
		return "", err
	}
	if d.Deployments == nil {
		return "", nil
	}
	dep, err := d.Deployments.GetDeployment(ctx, depID)
	if err != nil {
		return "", err
	}
	if dep.AppID != dbID {
		return "", domain.ErrNotFound
	}
	if d.LogsDir != "" {
		if content, err := os.ReadFile(filepath.Join(d.LogsDir, "deployments", depID.String()+".log")); err == nil {
			return string(content), nil
		}
	}
	if dep.ContainerID == "" || d.Runtime == nil {
		return "", nil
	}
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	lines, err := d.Runtime.LogTail(ctx, dep.ContainerID, limit)
	if err != nil {
		return "", nil
	}
	return strings.Join(lines, "\n"), nil
}

func (d *Databases) Start(ctx context.Context, id, orgID uuid.UUID) (*domain.Database, error) {
	db, err := d.Get(ctx, id, orgID)
	if err != nil {
		return nil, err
	}
	if db.ContainerID == "" {
		return d.Deploy(ctx, id, orgID)
	}
	if err := d.Runtime.Start(ctx, db.ContainerID); err != nil {
		return nil, err
	}
	if err := d.Store.UpdateDatabaseStatus(ctx, id, "running", db.ContainerID); err != nil {
		return nil, err
	}
	return d.Get(ctx, id, orgID)
}

func (d *Databases) Stop(ctx context.Context, id, orgID uuid.UUID) (*domain.Database, error) {
	db, err := d.Get(ctx, id, orgID)
	if err != nil {
		return nil, err
	}
	if db.ContainerID != "" {
		if err := d.Runtime.Stop(ctx, db.ContainerID); err != nil {
			return nil, err
		}
	}
	if err := d.Store.UpdateDatabaseStatus(ctx, id, "stopped", db.ContainerID); err != nil {
		return nil, err
	}
	return d.Get(ctx, id, orgID)
}

func (d *Databases) State(ctx context.Context, id, orgID uuid.UUID) (string, error) {
	db, err := d.Get(ctx, id, orgID)
	if err != nil {
		return "", err
	}
	if db.ContainerID == "" {
		return "no_container", nil
	}
	return d.Runtime.ContainerState(ctx, db.ContainerID)
}

func (d *Databases) ContainerID(ctx context.Context, id, orgID uuid.UUID) (string, error) {
	db, err := d.Get(ctx, id, orgID)
	if err != nil {
		return "", err
	}
	if db.ContainerID == "" {
		return "", domain.ErrNotFound
	}
	return db.ContainerID, nil
}
