package application

import (
	"context"
	"errors"
	"io"
)

var ErrUnsupportedEngine = errors.New("unsupported engine")

type BackupEngine string

const (
	EnginePostgres BackupEngine = "postgres"
	EngineMySQL    BackupEngine = "mysql"
	EngineMariaDB  BackupEngine = "mariadb"
	EngineMSSQL    BackupEngine = "mssql"
	EngineOracle   BackupEngine = "oracle"
	EngineMongoDB  BackupEngine = "mongodb"
)

type DBDescriptor struct {
	Engine      string
	ContainerID string
	User        string
	Password    string
	DBName      string
	Version     string
	Format      string
}

type BackupAdapter interface {
	Engine() BackupEngine
	Format() string
	ContentType() string
	Validate(ctx context.Context, db DBDescriptor) error
	CreateBackup(ctx context.Context, db DBDescriptor, dest io.Writer) error
	Restore(ctx context.Context, db DBDescriptor, src io.Reader) error
}

var backupAdapters = map[BackupEngine]BackupAdapter{}

func RegisterBackupAdapter(a BackupAdapter) {
	backupAdapters[a.Engine()] = a
}

func BackupAdapterFor(engine BackupEngine) (BackupAdapter, error) {
	a, ok := backupAdapters[engine]
	if !ok {
		return nil, ErrUnsupportedEngine
	}
	return a, nil
}

var engineByString = map[string]BackupEngine{
	"postgres": EnginePostgres,
	"mysql":    EngineMySQL,
	"mariadb":  EngineMariaDB,
	"mssql":    EngineMSSQL,
	"oracle":   EngineOracle,
	"mongodb":  EngineMongoDB,
}

func ParseBackupEngine(v string) (BackupEngine, bool) {
	e, ok := engineByString[v]
	return e, ok
}

func adapterForEngine(engine string) (BackupAdapter, error) {
	e, ok := ParseBackupEngine(engine)
	if !ok {
		return nil, ErrUnsupportedEngine
	}
	return BackupAdapterFor(e)
}
