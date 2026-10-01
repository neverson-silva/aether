package domain

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound            = errors.New("not found")
	ErrValidation          = errors.New("invalid input")
	ErrConflict            = errors.New("conflict")
	ErrForbidden           = errors.New("access denied")
	ErrDatabaseUnavailable = errors.New("database unavailable")
)

type Engine string

const (
	EnginePostgres Engine = "postgres"
	EngineMysql    Engine = "mysql"
	EngineMariaDB  Engine = "mariadb"
	EngineRedis    Engine = "redis"
	EngineMongoDB  Engine = "mongodb"
	EngineMSSQL    Engine = "mssql"
	EngineOracle   Engine = "oracle"
)

func (e Engine) Valid() bool {
	switch e {
	case EnginePostgres, EngineMysql, EngineMariaDB, EngineRedis, EngineMongoDB, EngineMSSQL, EngineOracle:
		return true
	}
	return false
}

func InternalHost(name string, id uuid.UUID) string {
	var slug strings.Builder
	for _, character := range strings.ToLower(name) {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' {
			slug.WriteRune(character)
			continue
		}
		if slug.Len() > 0 && slug.String()[slug.Len()-1] != '-' {
			slug.WriteByte('-')
		}
	}
	value := strings.Trim(slug.String(), "-")
	if len(value) > 48 {
		value = strings.TrimRight(value[:48], "-")
	}
	identifier := strings.ReplaceAll(id.String(), "-", "")
	if len(identifier) > 8 {
		identifier = identifier[:8]
	}
	if value == "" {
		return "db-" + identifier
	}
	return "db-" + value + "-" + identifier
}

type TableColumn struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Primary  bool   `json:"primary"`
	Default  string `json:"default"`
}

type CreateTableInput struct {
	Schema  string        `json:"schema"`
	Table   string        `json:"table"`
	Columns []TableColumn `json:"columns"`
}

type Database struct {
	ID               uuid.UUID
	ServiceID        uuid.UUID
	OrgID            uuid.UUID
	ProjectID        uuid.UUID
	EnvironmentID    *uuid.UUID
	Name             string
	Engine           Engine
	Version          string
	Port             int
	InternalPort     int
	PublicAccess     bool
	ExternalPort     int
	DataVolume       string
	DataVolumeTarget string
	DBName           string
	User             string
	PassEnc          string
	CPUs             string
	MemMB            int
	StorageMB        int
	Status           string
	ContainerID      string
	CreatedAt        time.Time
}

type PasswordCipher interface {
	Encrypt(plain string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type Store interface {
	CreateDatabase(ctx context.Context, db *Database) (*Database, error)
	GetDatabase(ctx context.Context, id uuid.UUID) (*Database, error)
	ListDatabasesByOrg(ctx context.Context, orgID uuid.UUID) ([]Database, error)
	UpdateDatabaseStatus(ctx context.Context, id uuid.UUID, status, containerID string) error
	UpdateDatabaseNetwork(ctx context.Context, id uuid.UUID, publicAccess bool, externalPort int) error
	UpdateDatabaseDataVolume(ctx context.Context, id uuid.UUID, volume, target string) error
	DeleteDatabase(ctx context.Context, id, orgID uuid.UUID) error
}

type OrganizationStorageUsage interface {
	OrganizationStorageMB(ctx context.Context, orgID uuid.UUID) (int, error)
}
