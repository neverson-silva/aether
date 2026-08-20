package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/uuid"

	appsdomain "aether/internal/apps/domain"
	"aether/internal/databases/domain"
	deploydomain "aether/internal/deployments/domain"
	"aether/internal/hostinfo"
)

type Databases struct {
	Store       domain.Store
	Apps        AppStore
	Passwords   domain.PasswordCipher
	Runtime     ContainerRuntime
	Network     string
	LogsDir     string
	Deployments deploydomain.Store
	OnCreated   func(ctx context.Context, serviceID uuid.UUID, serviceType, name string, orgID uuid.UUID)
}

type AppStore interface {
	GetProject(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.Project, error)
}

var defaultVersions = map[domain.Engine]string{
	domain.EnginePostgres: "16", domain.EngineMysql: "8.4", domain.EngineMariaDB: "11",
	domain.EngineRedis: "7", domain.EngineMongoDB: "6", domain.EngineMSSQL: "2022", domain.EngineOracle: "21c",
}

var defaultPorts = map[domain.Engine]int{
	domain.EnginePostgres: 5432, domain.EngineMysql: 3306, domain.EngineMariaDB: 3306,
	domain.EngineRedis: 6379, domain.EngineMongoDB: 27017, domain.EngineMSSQL: 1433, domain.EngineOracle: 1521,
}

func (d *Databases) Create(ctx context.Context, orgID, projectID uuid.UUID, name string, engine domain.Engine, version, user, password string, memMB, storageMB int) (*domain.Database, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 {
		return nil, domain.ErrValidation
	}
	if !engine.Valid() {
		return nil, domain.ErrValidation
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
		OrgID: orgID, ProjectID: projectID, Name: name, Engine: engine,
		Version: version, Port: defaultPorts[engine], DBName: name, User: user,
		PassEnc: passEnc, MemMB: memMB, StorageMB: storageMB, Status: "creating",
	})
	if err != nil {
		return nil, err
	}
	if d.OnCreated != nil {
		d.OnCreated(ctx, db.ID, "db", db.Name, orgID)
	}
	return db, nil
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

func (d *Databases) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	db, err := d.Get(ctx, id, orgID)
	if err != nil {
		return err
	}
	if d.Runtime != nil {
		_ = d.Runtime.RemoveByLabel(ctx, "aether.database-id="+id.String())
		if db.ContainerID != "" {
			_ = d.Runtime.Remove(ctx, db.ContainerID)
		}
		_ = d.Runtime.Remove(ctx, "db-"+db.Name)
	}
	return d.Store.DeleteDatabase(ctx, id, orgID)
}

func (d *Databases) ConnectionString(ctx context.Context, id, orgID uuid.UUID) (string, error) {
	db, err := d.Get(ctx, id, orgID)
	if err != nil {
		return "", err
	}
	pass, err := d.Passwords.Decrypt(db.PassEnc)
	if err != nil {
		return "", err
	}
	host := hostinfo.PublicIP()
	switch db.Engine {
	case domain.EnginePostgres:
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", db.User, pass, host, db.Port, db.DBName), nil
	case domain.EngineMysql, domain.EngineMariaDB:
		return fmt.Sprintf("mysql://%s:%s@%s:%d/%s", db.User, pass, host, db.Port, db.DBName), nil
	case domain.EngineRedis:
		return fmt.Sprintf("redis://:%s@%s:%d/0", pass, host, db.Port), nil
	case domain.EngineMongoDB:
		return fmt.Sprintf("mongodb://%s:%s@%s:%d/%s", db.User, pass, host, db.Port, db.DBName), nil
	case domain.EngineMSSQL:
		return fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s", db.User, pass, host, db.Port, db.DBName), nil
	default:
		return fmt.Sprintf("%s://%s:%s@%s:%d/%s", db.Engine, db.User, pass, host, db.Port, db.DBName), nil
	}
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
