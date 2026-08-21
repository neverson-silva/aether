package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("invalid input")
	ErrConflict   = errors.New("conflict")
	ErrForbidden  = errors.New("access denied")
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

type TableColumn struct {
	Name     string `json:"name"`
	Type    string `json:"type"`
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
	ID          uuid.UUID
	OrgID       uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	Engine      Engine
	Version     string
	Port        int
	DBName      string
	User        string
	PassEnc     string
	MemMB       int
	StorageMB   int
	Status      string
	ContainerID string
	CreatedAt   time.Time
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
	UpdateDatabasePort(ctx context.Context, id uuid.UUID, port int) error
	DeleteDatabase(ctx context.Context, id, orgID uuid.UUID) error
}
