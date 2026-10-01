package application

import (
	"context"
	"errors"
	"fmt"
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

func databaseDataTarget(engine domain.Engine, version string) string {
	switch engine {
	case domain.EnginePostgres:
		versionNumber := strings.TrimFunc(version, func(value rune) bool { return value < '0' || value > '9' })
		major, err := strconv.Atoi(strings.SplitN(versionNumber, ".", 2)[0])
		if err == nil && major >= 18 || version == "latest" {
			return "/var/lib/postgresql"
		}
		return "/var/lib/postgresql/data"
	case domain.EngineMysql, domain.EngineMariaDB:
		return "/var/lib/mysql"
	case domain.EngineMongoDB:
		return "/data/db"
	case domain.EngineRedis:
		return "/data"
	case domain.EngineMSSQL:
		return "/var/opt/mssql"
	case domain.EngineOracle:
		return "/opt/oracle/oradata"
	default:
		return ""
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
	containerPort := db.InternalPort
	if containerPort == 0 {
		containerPort = defaultPorts[db.Engine]
	}
	if containerPort == 0 {
		return "", fmt.Errorf("%w: missing internal port for %s", domain.ErrValidation, db.Engine)
	}
	if db.PublicAccess && (db.ExternalPort < 1024 || db.ExternalPort > 65535) {
		return "", fmt.Errorf("%w: invalid external port", domain.ErrValidation)
	}
	environmentID := db.ProjectID
	if db.EnvironmentID != nil {
		environmentID = *db.EnvironmentID
	}
	environmentNetwork := worker.EnvironmentNetworkName(environmentID)
	networks, ok := d.Runtime.(worker.NetworkRuntime)
	if !ok {
		return "", errors.New("environment network runtime is not configured")
	}
	if err := networks.EnsureNetwork(ctx, environmentNetwork, map[string]string{"io.aether.component": "environment", "io.aether.environment-id": environmentID.String()}); err != nil {
		return "", err
	}
	oldContainerID := db.ContainerID
	if oldContainerID == "" {
		finder, ok := d.Runtime.(interface {
			ContainerIDsByLabel(context.Context, string) ([]string, error)
		})
		if ok {
			ids, findErr := finder.ContainerIDsByLabel(ctx, "aether.database-id="+db.ID.String())
			if findErr != nil {
				return "", findErr
			}
			if len(ids) > 1 {
				return "", errors.New("cannot safely redeploy database with multiple existing containers")
			}
			if len(ids) == 1 {
				oldContainerID = ids[0]
			}
		} else if db.Status == "running" || db.Status == "stopped" {
			return "", errors.New("cannot safely redeploy database without locating its existing container")
		}
	}
	if db.PublicAccess {
		checker, ok := d.Runtime.(interface {
			PortInUse(context.Context, int, string) (bool, error)
		})
		if !ok {
			return "", errors.New("published port validation is unavailable")
		}
		inUse, checkErr := checker.PortInUse(ctx, db.ExternalPort, oldContainerID)
		if checkErr != nil {
			return "", checkErr
		}
		if inUse {
			return "", fmt.Errorf("%w: port %d is already in use on this server", domain.ErrConflict, db.ExternalPort)
		}
	}
	dataTarget := db.DataVolumeTarget
	if dataTarget == "" {
		dataTarget = databaseDataTarget(db.Engine, db.Version)
	}
	if dataTarget == "" {
		return "", fmt.Errorf("%w: no data volume target for %s", domain.ErrValidation, db.Engine)
	}
	dataVolume := db.DataVolume
	if dataVolume == "" && oldContainerID != "" {
		inspector, ok := d.Runtime.(interface {
			ContainerMountSources(context.Context, string) (map[string]string, error)
		})
		if !ok {
			return "", errors.New("cannot safely redeploy database without inspecting its existing data volume")
		}
		sources, inspectErr := inspector.ContainerMountSources(ctx, oldContainerID)
		if inspectErr != nil {
			return "", inspectErr
		}
		dataVolume = sources[dataTarget]
		if dataVolume == "" && db.Engine == domain.EnginePostgres {
			for _, candidate := range []string{"/var/lib/postgresql/data", "/var/lib/postgresql"} {
				if source := sources[candidate]; source != "" {
					dataTarget, dataVolume = candidate, source
					break
				}
			}
		}
		if dataVolume == "" {
			return "", errors.New("cannot safely redeploy database because its existing data volume was not found")
		}
	}
	if dataVolume == "" {
		dataVolume = "aether-db-" + strings.ReplaceAll(db.ID.String(), "-", "")
		volumes, ok := d.Runtime.(worker.VolumeRuntime)
		if !ok {
			return "", errors.New("database volume runtime is not configured")
		}
		if err := volumes.CreateVolume(ctx, dataVolume, map[string]string{"aether.owner": "aether", "aether.database-id": db.ID.String()}); err != nil {
			return "", err
		}
	}
	if dataVolume != db.DataVolume || dataTarget != db.DataVolumeTarget {
		if err := d.Store.UpdateDatabaseDataVolume(ctx, db.ID, dataVolume, dataTarget); err != nil {
			return "", err
		}
		db.DataVolume, db.DataVolumeTarget = dataVolume, dataTarget
	}
	if oldContainerID != "" {
		if err := d.Runtime.Remove(ctx, oldContainerID); err != nil && !errors.Is(err, worker.ErrContainerNotFound) {
			return "", err
		}
	}
	serviceID := db.ServiceID
	if serviceID == uuid.Nil {
		serviceID = db.ID
	}
	spec := worker.RunSpec{
		Name:          "db-" + strings.ReplaceAll(db.ID.String(), "-", ""),
		Image:         image,
		Env:           d.runtimeEnv(ctx, db, pass),
		Port:          0,
		ContainerPort: containerPort,
		HostIP:        "0.0.0.0",
		Network:       environmentNetwork,
		NetworkAlias:  domain.InternalHost(db.Name, db.ID),
		MemMB:         db.MemMB,
		CPUs:          db.CPUs,
		StorageMB:     db.StorageMB,
		Mounts:        []worker.MountSpec{{Source: dataVolume, Target: dataTarget}},
		Labels: map[string]string{
			"aether.owner":        "aether",
			"aether.service-type": "database",
			"aether.service-id":   serviceID.String(),
			"aether.service-name": db.Name,
			"aether.project-id":   db.ProjectID.String(),
			"aether.database-id":  db.ID.String(),
		},
	}
	if db.Engine == domain.EngineRedis {
		spec.Command = []string{"redis-server", "--requirepass", pass, "--appendonly", "yes"}
	}
	if db.PublicAccess {
		spec.Port = db.ExternalPort
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

const databaseHealthTimeout = 120 * time.Second

func (d *Databases) waitHealthy(ctx context.Context, db *domain.Database, containerID string, containerPort int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	password, err := d.Passwords.Decrypt(db.PassEnc)
	if err != nil {
		return domain.ErrValidation
	}
	var command []string
	var env []string
	switch db.Engine {
	case domain.EnginePostgres:
		command = []string{"pg_isready", "-q", "-h", "127.0.0.1", "-p", strconv.Itoa(containerPort)}
	case domain.EngineMysql, domain.EngineMariaDB:
		command = []string{"mysqladmin", "ping", "-h", "127.0.0.1", "-u", "root"}
		env = []string{"MYSQL_PWD=" + password}
	case domain.EngineRedis:
		command = []string{"redis-cli", "ping"}
		env = []string{"REDISCLI_AUTH=" + password}
	case domain.EngineMongoDB:
		command = []string{"sh", "-c", "mongosh --quiet --username \"$MONGO_INITDB_ROOT_USERNAME\" --password \"$MONGO_INITDB_ROOT_PASSWORD\" --authenticationDatabase admin --eval \"db.adminCommand('ping').ok\""}
		env = []string{"MONGO_INITDB_ROOT_USERNAME=" + db.User, "MONGO_INITDB_ROOT_PASSWORD=" + password}
	case domain.EngineMSSQL:
		command = []string{"sh", "-c", "for p in /opt/mssql-tools18/bin/sqlcmd /opt/mssql-tools/bin/sqlcmd; do if [ -x \"$p\" ]; then \"$p\" -C -S localhost -U sa -P \"$MSSQL_SA_PASSWORD\" -Q 'SELECT 1'; exit $?; fi; done; exit 1"}
		env = []string{"MSSQL_SA_PASSWORD=" + password}
	case domain.EngineOracle:
		command = []string{"bash", "-lc", "echo 'SELECT 1 FROM DUAL;' | sqlplus -s system/$ORACLE_PASSWORD@localhost/FREEPDB1"}
		env = []string{"ORACLE_PASSWORD=" + password}
	default:
		return fmt.Errorf("%w: unsupported database health check", domain.ErrValidation)
	}
	for {
		_, stderr, err := d.Runtime.Exec(ctx, containerID, env, command...)
		if err == nil {
			return nil
		}
		lastErr = err
		if stderr != "" {
			lastErr = fmt.Errorf("database health check: %s", strings.TrimSpace(stderr))
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
	containerPort := db.InternalPort
	if containerPort == 0 {
		containerPort = defaultPorts[db.Engine]
	}
	if err := d.waitHealthy(ctx, db, containerID, containerPort, databaseHealthTimeout); err != nil {
		if lines, logErr := d.Runtime.LogTail(ctx, containerID, 40); logErr == nil {
			for _, line := range lines {
				d.appendDeployLog(ctx, deploymentID, "Container: "+line)
			}
		}
		_ = d.Runtime.Remove(ctx, containerID)
		d.appendDeployLog(ctx, deploymentID, "Health check failed: "+err.Error())
		if dep != nil {
			d.finishDeployment(ctx, dep.ID, deploydomain.StatusFailed, containerID, err.Error())
			d.notifyDeployment(ctx, dep, deploydomain.StatusFailed, err.Error())
		}
		_ = d.Store.UpdateDatabaseStatus(ctx, id, "failed", containerID)
		return nil, fmt.Errorf("database did not become healthy: %w", err)
	}
	d.appendDeployLog(ctx, deploymentID, "Database is healthy on internal port "+strconv.Itoa(containerPort))
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
	f, err := os.OpenFile(filepath.Join(dir, depID.String()+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
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
