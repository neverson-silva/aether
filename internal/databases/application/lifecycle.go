package application

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/google/uuid"

	"aether/internal/databases/domain"
	"aether/internal/worker"
)

type ContainerRuntime interface {
	Run(ctx context.Context, spec worker.RunSpec) (string, error)
	Start(ctx context.Context, containerID string) error
	Stop(ctx context.Context, containerID string) error
	Restart(ctx context.Context, containerID string) error
	Remove(ctx context.Context, containerID string) error
	ContainerState(ctx context.Context, containerID string) (string, error)
}

var dbImages = map[domain.Engine]string{
	domain.EnginePostgres: "docker.io/postgres:16",
	domain.EngineMysql:    "docker.io/mysql:8.4",
	domain.EngineMariaDB:  "docker.io/mariadb:11",
	domain.EngineRedis:    "docker.io/redis:7",
	domain.EngineMongoDB:  "docker.io/mongo:7",
	domain.EngineMSSQL:    "mcr.microsoft.com/mssql/server:2022",
	domain.EngineOracle:   "gvenzl/oracle-free:23",
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
	image := dbImages[db.Engine]
	if image == "" {
		return "", fmt.Errorf("%w: unsupported engine %s", domain.ErrValidation, db.Engine)
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
	spec := worker.RunSpec{
		Name:          "db-" + db.Name,
		Image:         image,
		Env:           dbEnv(db, pass),
		Port:          hostPort,
		ContainerPort: containerPort,
		Network:       d.Network,
		NetworkAlias:  "db-" + db.ID.String()[:8],
		MemMB:         db.MemMB,
		Labels: map[string]string{
			"aether.resource":    "database",
			"aether.database-id": db.ID.String(),
			"aether.project-id":  db.ProjectID.String(),
		},
	}
	containerID, err := d.Runtime.Run(ctx, spec)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(containerID), nil
}

func hostPortFree(port int) bool {
	if port <= 0 {
		return false
	}
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}

func (d *Databases) allocHostPort(stored int) int {
	base := d.PortBase
	if base <= 0 || base > 65500 {
		base = 20000
	}
	limit := base + 1000
	start := stored
	if start < base || start >= limit {
		start = base
	}
	for p := start; p < limit; p++ {
		if hostPortFree(p) {
			return p
		}
	}
	return 0
}

func (d *Databases) Deploy(ctx context.Context, id, orgID uuid.UUID) (*domain.Database, error) {
	db, err := d.Get(ctx, id, orgID)
	if err != nil {
		return nil, err
	}
	containerID, err := d.deploy(ctx, db)
	if err != nil {
		_ = d.Store.UpdateDatabaseStatus(ctx, id, "failed", db.ContainerID)
		return nil, err
	}
	if err := d.Store.UpdateDatabaseStatus(ctx, id, "running", containerID); err != nil {
		return nil, err
	}
	return d.Get(ctx, id, orgID)
}

func (d *Databases) Rebuild(ctx context.Context, id, orgID uuid.UUID) (*domain.Database, error) {
	return d.Deploy(ctx, id, orgID)
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
