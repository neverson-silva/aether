package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"

	appsdomain "aether/internal/modules/apps/domain"
	"aether/internal/modules/databases/domain"
	deploydomain "aether/internal/modules/deployments/domain"
	"aether/internal/platform/hostinfo"
)

type Databases struct {
	Store       domain.Store
	Apps        AppStore
	Passwords   domain.PasswordCipher
	Runtime     ContainerRuntime
	LogsDir     string
	Deployments deploydomain.Store
	Audit       interface {
		Record(context.Context, uuid.UUID, string, string, string, string)
	}
	Variables interface {
		Effective(context.Context, uuid.UUID, uuid.UUID) (map[string]string, error)
	}
	Notifier interface {
		NotifyDeploy(context.Context, deploydomain.DeployEvent)
	}
}

type AppStore interface {
	GetProject(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.Project, error)
	GetEnvironment(ctx context.Context, id, projectID uuid.UUID) (*appsdomain.Environment, error)
	GetAppByName(ctx context.Context, orgID uuid.UUID, name string) (*appsdomain.App, error)
	DefaultEnvironment(ctx context.Context, projectID uuid.UUID) (uuid.UUID, error)
}

var defaultVersions = map[domain.Engine]string{
	domain.EnginePostgres: "16", domain.EngineMysql: "8.4", domain.EngineMariaDB: "11",
	domain.EngineRedis: "7", domain.EngineMongoDB: "6", domain.EngineMSSQL: "2022", domain.EngineOracle: "21c",
}

var defaultPorts = map[domain.Engine]int{
	domain.EnginePostgres: 5432, domain.EngineMysql: 3306, domain.EngineMariaDB: 3306,
	domain.EngineRedis: 6379, domain.EngineMongoDB: 27017, domain.EngineMSSQL: 1433, domain.EngineOracle: 1521,
}

const (
	maxDatabaseMemoryMB      = 2048
	maxDatabaseStorageMB     = 102400
	maxOrganizationStorageMB = 512000
)

func (d *Databases) Create(ctx context.Context, orgID, projectID uuid.UUID, name string, engine domain.Engine, version, user, password, cpus string, memMB, storageMB int) (*domain.Database, error) {
	var environmentID *uuid.UUID
	if id, err := d.Apps.DefaultEnvironment(ctx, projectID); err == nil {
		environmentID = &id
	}
	return d.create(ctx, orgID, projectID, environmentID, name, engine, version, user, password, cpus, memMB, storageMB)
}

func (d *Databases) CreateInEnvironment(ctx context.Context, orgID, projectID, environmentID uuid.UUID, name string, engine domain.Engine, version, user, password, cpus string, memMB, storageMB int) (*domain.Database, error) {
	if _, err := d.Apps.GetProject(ctx, projectID, orgID); err != nil {
		return nil, err
	}
	if _, err := d.Apps.GetEnvironment(ctx, environmentID, projectID); err != nil {
		return nil, err
	}
	return d.create(ctx, orgID, projectID, &environmentID, name, engine, version, user, password, cpus, memMB, storageMB)
}

func (d *Databases) create(ctx context.Context, orgID, projectID uuid.UUID, environmentID *uuid.UUID, name string, engine domain.Engine, version, user, password, cpus string, memMB, storageMB int) (*domain.Database, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 {
		return nil, domain.ErrValidation
	}
	if !engine.Valid() {
		return nil, domain.ErrValidation
	}
	if memMB < 0 || memMB > maxDatabaseMemoryMB || storageMB < 0 || storageMB > maxDatabaseStorageMB {
		return nil, domain.ErrValidation
	}
	cpus = strings.TrimSpace(cpus)
	if cpus == "" {
		cpus = "0.5"
	}
	if !validDatabaseCPUs(cpus) {
		return nil, domain.ErrValidation
	}
	if usageStore, ok := d.Store.(domain.OrganizationStorageUsage); ok {
		used, err := usageStore.OrganizationStorageMB(ctx, orgID)
		if err != nil {
			return nil, err
		}
		if used < 0 || used > maxOrganizationStorageMB-storageMB {
			return nil, domain.ErrConflict
		}
	}
	user = strings.TrimSpace(user)
	if user == "" {
		user = "aether"
	}
	if !validDBUser(user) {
		return nil, domain.ErrValidation
	}
	if _, err := d.Apps.GetProject(ctx, projectID, orgID); err != nil {
		return nil, err
	}
	if _, err := d.Apps.GetAppByName(ctx, orgID, name); err == nil {
		return nil, domain.ErrConflict
	} else if !errors.Is(err, appsdomain.ErrNotFound) {
		return nil, err
	}
	list, err := d.Store.ListDatabasesByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for _, existing := range list {
		if strings.EqualFold(existing.Name, name) {
			return nil, domain.ErrConflict
		}
	}
	if version == "" {
		version = defaultVersions[engine]
	}
	password = strings.TrimSpace(password)
	if password != "" && (len(password) < 8 || len(password) > 128) {
		return nil, domain.ErrValidation
	}
	if password == "" {
		password, err = randomPassword()
		if err != nil {
			return nil, err
		}
	}
	passEnc, err := d.Passwords.Encrypt(password)
	if err != nil {
		return nil, err
	}
	db, err := d.Store.CreateDatabase(ctx, &domain.Database{
		OrgID: orgID, ProjectID: projectID, EnvironmentID: environmentID, Name: name, Engine: engine,
		Version: version, Port: defaultPorts[engine], InternalPort: defaultPorts[engine], DBName: name, User: user,
		PassEnc: passEnc, CPUs: cpus, MemMB: memMB, StorageMB: storageMB, Status: "creating",
	})
	if err != nil {
		return nil, err
	}
	if d.Audit != nil {
		d.Audit.Record(ctx, orgID, "database.created", "database", db.ID.String(), string(engine))
	}
	return db, nil
}

func validDatabaseCPUs(value string) bool {
	cpus, err := strconv.ParseFloat(value, 64)
	if err != nil || cpus < 0.25 || cpus > 2 || math.Abs(cpus*4-math.Round(cpus*4)) > 1e-9 {
		return false
	}
	return true
}

func (d *Databases) List(ctx context.Context, orgID uuid.UUID) ([]domain.Database, error) {
	return d.Store.ListDatabasesByOrg(ctx, orgID)
}

func (d *Databases) Get(ctx context.Context, id, orgID uuid.UUID) (*domain.Database, error) {
	db, err := d.Store.GetDatabase(ctx, id)
	if err != nil {
		return nil, err
	}
	if db.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	return db, nil
}

func (d *Databases) GetByServiceID(ctx context.Context, serviceID, orgID uuid.UUID) (*domain.Database, error) {
	databases, err := d.Store.ListDatabasesByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for index := range databases {
		if databases[index].ServiceID == serviceID {
			return &databases[index], nil
		}
	}
	return nil, domain.ErrNotFound
}

func (d *Databases) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	db, err := d.Get(ctx, id, orgID)
	if err != nil {
		return err
	}
	if d.Runtime != nil {
		runtime, hasVolumeRemoval := d.Runtime.(interface {
			RemoveWithVolumes(context.Context, string) error
		})
		if db.ContainerID != "" {
			if hasVolumeRemoval {
				_ = runtime.RemoveWithVolumes(ctx, db.ContainerID)
			} else {
				_ = d.Runtime.Remove(ctx, db.ContainerID)
			}
		}
		if hasVolumeRemoval {
			_ = runtime.RemoveWithVolumes(ctx, "db-"+db.Name)
		} else {
			_ = d.Runtime.Remove(ctx, "db-"+db.Name)
		}
		_ = d.Runtime.RemoveByLabel(ctx, "aether.database-id="+id.String())
	}
	return d.Store.DeleteDatabase(ctx, id, orgID)
}

func (d *Databases) ConnectionString(ctx context.Context, id, orgID uuid.UUID) (string, error) {
	db, err := d.Get(ctx, id, orgID)
	if err != nil {
		return "", err
	}
	return d.connectionString(db)
}

func (d *Databases) ConnectionStringByServiceID(ctx context.Context, serviceID, orgID uuid.UUID) (string, error) {
	db, err := d.GetByServiceID(ctx, serviceID, orgID)
	if err != nil {
		return "", err
	}
	return d.connectionString(db)
}

func (d *Databases) connectionString(db *domain.Database) (string, error) {
	return d.connectionStringFor(db, "internal", "")
}

func (d *Databases) ConnectionDetails(ctx context.Context, id, orgID uuid.UUID, scope string) (ConnectionDetails, error) {
	db, err := d.Get(ctx, id, orgID)
	if err != nil {
		return ConnectionDetails{}, err
	}
	if scope != "internal" && scope != "external" {
		return ConnectionDetails{}, domain.ErrValidation
	}
	if scope == "external" && !db.PublicAccess {
		return ConnectionDetails{}, domain.ErrForbidden
	}
	dsn, err := d.connectionStringFor(db, scope, hostinfo.PublicIP())
	if err != nil {
		return ConnectionDetails{}, err
	}
	password, err := d.Passwords.Decrypt(db.PassEnc)
	if err != nil {
		return ConnectionDetails{}, err
	}
	port := db.InternalPort
	host := domain.InternalHost(db.Name, db.ID)
	if scope == "external" {
		port = db.ExternalPort
		host = hostinfo.PublicIP()
	}
	username := db.User
	if db.Engine == domain.EngineRedis {
		username = ""
	} else if db.Engine == domain.EngineMSSQL {
		username = "sa"
	} else if db.Engine == domain.EngineOracle {
		username = "system"
	}
	return ConnectionDetails{Host: host, Port: port, Database: db.DBName, Username: username, Password: password, URL: dsn}, nil
}

type ConnectionDetails struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	URL      string `json:"url"`
}

func (d *Databases) UpdateNetworkAccess(ctx context.Context, id, orgID uuid.UUID, publicAccess bool, externalPort int) (*domain.Database, error) {
	db, err := d.Get(ctx, id, orgID)
	if err != nil {
		return nil, err
	}
	previousPublicAccess, previousExternalPort := db.PublicAccess, db.ExternalPort
	if publicAccess && (externalPort < 1024 || externalPort > 65535) {
		return nil, domain.ErrValidation
	}
	if !publicAccess {
		externalPort = 0
	}
	if publicAccess && (!db.PublicAccess || db.ExternalPort != externalPort) {
		checker, ok := d.Runtime.(interface {
			PortInUse(context.Context, int, string) (bool, error)
		})
		if !ok {
			return nil, errors.New("published port validation is unavailable")
		}
		inUse, checkErr := checker.PortInUse(ctx, externalPort, db.ContainerID)
		if checkErr != nil {
			return nil, checkErr
		}
		if inUse {
			return nil, fmt.Errorf("%w: port %d is already in use on this server", domain.ErrConflict, externalPort)
		}
	}
	if err := d.Store.UpdateDatabaseNetwork(ctx, id, publicAccess, externalPort); err != nil {
		return nil, err
	}
	if d.Audit != nil && (previousPublicAccess != publicAccess || previousExternalPort != externalPort) {
		action := "database.public_access.disabled"
		details := ""
		if publicAccess && !previousPublicAccess {
			action = "database.public_access.enabled"
			details = fmt.Sprintf("external_port=%d", externalPort)
		} else if publicAccess && previousExternalPort != externalPort {
			action = "database.external_port.changed"
			details = fmt.Sprintf("external_port=%d", externalPort)
		}
		d.Audit.Record(ctx, orgID, action, "database", id.String(), details)
	}
	db.PublicAccess = publicAccess
	db.ExternalPort = externalPort
	return db, nil
}

func (d *Databases) connectionStringFor(db *domain.Database, scope, publicHost string) (string, error) {
	pass, err := d.Passwords.Decrypt(db.PassEnc)
	if err != nil {
		return "", err
	}
	host, port := domain.InternalHost(db.Name, db.ID), db.InternalPort
	if scope == "external" {
		host, port = publicHost, db.ExternalPort
	}
	username := db.User
	if db.Engine == domain.EngineRedis {
		username = ""
	} else if db.Engine == domain.EngineMSSQL {
		username = "sa"
	} else if db.Engine == domain.EngineOracle {
		username = "system"
	}
	scheme := string(db.Engine)
	path := db.DBName
	query := ""
	switch db.Engine {
	case domain.EnginePostgres:
		scheme = "postgresql"
	case domain.EngineMysql, domain.EngineMariaDB:
		scheme = "mysql"
	case domain.EngineRedis:
		path = "0"
	case domain.EngineMongoDB:
		query = "authSource=admin"
	case domain.EngineMSSQL:
		scheme = "sqlserver"
		path = ""
		query = "database=" + url.QueryEscape(db.DBName)
	}
	connection := url.URL{Scheme: scheme, User: url.UserPassword(username, pass), Host: net.JoinHostPort(host, strconv.Itoa(port)), Path: path}
	if query != "" {
		connection.RawQuery = query
	}
	return connection.String(), nil
}

func randomPassword() (string, error) {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func validDBUser(user string) bool {
	if len(user) == 0 || len(user) > 63 {
		return false
	}
	for i, r := range user {
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		if i == 0 && r >= 'A' && r <= 'Z' {
			continue
		}
		return false
	}
	return true
}
