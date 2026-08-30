package engine

import (
	"aether/internal/modules/backups/adapters/container"
	"aether/internal/modules/backups/adapters/engine/mariadb"
	"aether/internal/modules/backups/adapters/engine/mongodb"
	"aether/internal/modules/backups/adapters/engine/mssql"
	"aether/internal/modules/backups/adapters/engine/mysql"
	"aether/internal/modules/backups/adapters/engine/oracle"
	"aether/internal/modules/backups/adapters/engine/postgres"
	"aether/internal/modules/backups/application"
)

func Register(exec container.Executor) {
	application.RegisterBackupAdapter(mariadb.New(exec))
	application.RegisterBackupAdapter(mongodb.New(exec))
	application.RegisterBackupAdapter(mssql.New(exec))
	application.RegisterBackupAdapter(mysql.New(exec))
	application.RegisterBackupAdapter(oracle.New(exec))
	application.RegisterBackupAdapter(postgres.New(exec))
}

func init() {
	Register(container.Unavailable{})
}
