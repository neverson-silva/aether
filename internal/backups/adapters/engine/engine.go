// Package engine imports every database engine adapter so their init functions register them.
package engine

import (
	_ "aether/internal/backups/adapters/engine/mariadb"
	_ "aether/internal/backups/adapters/engine/mongodb"
	_ "aether/internal/backups/adapters/engine/mssql"
	_ "aether/internal/backups/adapters/engine/mysql"
	_ "aether/internal/backups/adapters/engine/oracle"
	_ "aether/internal/backups/adapters/engine/postgres"
)
