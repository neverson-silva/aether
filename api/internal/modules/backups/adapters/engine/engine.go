// Package engine imports every database engine adapter so their init functions register them.
package engine

import (
	_ "aether/internal/modules/backups/adapters/engine/mariadb"
	_ "aether/internal/modules/backups/adapters/engine/mongodb"
	_ "aether/internal/modules/backups/adapters/engine/mssql"
	_ "aether/internal/modules/backups/adapters/engine/mysql"
	_ "aether/internal/modules/backups/adapters/engine/oracle"
	_ "aether/internal/modules/backups/adapters/engine/postgres"
)
