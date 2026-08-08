package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aether/internal/domain"
	"aether/internal/runtime"
)

var dbImages = map[domain.DBEngine]map[string]string{
	domain.EnginePostgres: {
		"": "postgres:16", "16": "postgres:16", "15": "postgres:15", "14": "postgres:14",
	},
	domain.EngineMysql: {
		"": "mysql:8.4", "8": "mysql:8.4", "8.0": "mysql:8.0",
	},
	domain.EngineMariaDB: {
		"": "mariadb:11", "11": "mariadb:11", "10.11": "mariadb:10.11",
	},
	domain.EngineRedis: {
		"": "redis:7", "7": "redis:7", "6": "redis:6",
	},
	domain.EngineMongoDB: {
		"": "mongo:7", "7": "mongo:7", "6": "mongo:6",
	},
	domain.EngineMSSQL: {
		"": "mcr.microsoft.com/mssql/server:2022-latest", "2022": "mcr.microsoft.com/mssql/server:2022-latest",
	},
	domain.EngineOracle: {
		"": "gvenzl/oracle-free:23-slim", "23": "gvenzl/oracle-free:23-slim",
	},
}

var dbPorts = map[domain.DBEngine]int{
	domain.EnginePostgres: 5432,
	domain.EngineMysql:    3306,
	domain.EngineMariaDB:  3306,
	domain.EngineRedis:    6379,
	domain.EngineMongoDB:  27017,
	domain.EngineMSSQL:    1433,
	domain.EngineOracle:   1521,
}

func randomPassword() string {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "aether-" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return "aether-" + hex.EncodeToString(b)
}

func (c *Core) CreateDatabase(orgID, projectID, name string, engine domain.DBEngine, version string, memMB, storageMB int64) (*domain.Database, error) {
	if _, ok := dbImages[engine]; !ok {
		return nil, fmt.Errorf("engine não suportada: %s", engine)
	}
	if existing, err := c.Store.ListDatabases(orgID); err == nil {
		for _, d := range existing {
			if strings.EqualFold(d.Name, name) {
				return nil, fmt.Errorf("database %q já existe nesta organização — escolha outro nome", name)
			}
		}
	}
	if _, err := c.Store.GetProject(projectID); err != nil {
		return nil, err
	}
	pass := randomPassword()
	db := &domain.Database{
		ID:        domain.NewID(),
		OrgID:     orgID,
		ProjectID: projectID,
		Name:      name,
		Engine:    engine,
		Version:   version,
		Port:      dbPorts[engine],
		DBName:    name,
		User:      "aether",
		MemMB:     memMB,
		StorageMB: storageMB,
		Status:    "creating",
		CreatedAt: time.Now().UTC(),
	}
	enc, err := c.Secrets.EncryptString(pass)
	if err != nil {
		return nil, err
	}
	db.PassEnc = enc
	if err := c.Store.CreateDatabase(db); err != nil {
		return nil, err
	}
	go c.provisionDatabase(db)
	return db, nil
}

func (c *Core) DatabasePassword(db *domain.Database) (string, error) {
	return c.Secrets.DecryptString(db.PassEnc)
}

func (c *Core) provisionDatabase(db *domain.Database) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	defer func() {
		if r := recover(); r != nil {
			c.Store.UpdateDatabaseStatus(db.ID, "failed", db.ContainerID)
			log.Printf("[db] provisionamento de %s falhou: %v", db.Name, r)
		}
	}()

	image := dbImages[db.Engine][db.Version]
	if image == "" {
		image = dbImages[db.Engine][""]
	}
	pass, err := c.DatabasePassword(db)
	if err != nil {
		c.Store.UpdateDatabaseStatus(db.ID, "failed", "")
		return
	}
	network := "aether-" + db.ProjectID
	c.Driver.NetworkCreate(ctx, network)
	volume := "aether-db-" + db.Name
	c.Driver.VolumeCreate(ctx, volume, db.StorageMB)

	env := c.dbEnv(db.Engine, db.DBName, db.User, pass)
	spec := runtime.RunSpec{
		Name:     "aether-db-" + db.Name,
		Image:    image,
		Env:      env,
		Networks: []string{network},
		Volumes:  []runtime.VolumeMount{{Source: volume, Target: "/data"}},
		MemMB:    db.MemMB,
		Restart:  "always",
		Labels:   map[string]string{"aether.db": db.ID},
	}
	switch db.Engine {
	case domain.EnginePostgres:
		spec.Volumes[0].Target = "/var/lib/postgresql/data"
	case domain.EngineMSSQL:
		spec.Volumes[0].Target = "/var/opt/mssql"
	case domain.EngineOracle:
		spec.Volumes[0].Target = "/opt/oracle/oradata"
	}
	c.Driver.Remove(ctx, spec.Name, true)
	id, err := c.Driver.Run(ctx, spec)
	if err != nil {
		c.Store.UpdateDatabaseStatus(db.ID, "failed", "")
		log.Printf("[db] run %s falhou: %v", db.Name, err)
		return
	}
	db.ContainerID = id
	c.Store.UpdateDatabaseStatus(db.ID, "starting", id)

	wait := 3 * time.Minute
	if db.Engine == domain.EngineOracle {
		wait = 12 * time.Minute
	}
	if db.Engine == domain.EngineMSSQL {
		wait = 5 * time.Minute
	}
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		if c.dbReady(ctx, db, pass) {
			c.Store.UpdateDatabaseStatus(db.ID, "ready", id)
			log.Printf("[db] %s pronto (%s)", db.Name, image)
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
	c.Store.UpdateDatabaseStatus(db.ID, "failed", id)
}

func (c *Core) dbEnv(engine domain.DBEngine, dbName, user, pass string) []string {
	switch engine {
	case domain.EnginePostgres:
		return []string{"POSTGRES_PASSWORD=" + pass, "POSTGRES_USER=" + user, "POSTGRES_DB=" + dbName}
	case domain.EngineMysql, domain.EngineMariaDB:
		return []string{"MYSQL_ROOT_PASSWORD=" + pass, "MYSQL_DATABASE=" + dbName, "MYSQL_USER=" + user, "MYSQL_PASSWORD=" + pass}
	case domain.EngineRedis:
		return []string{"REDIS_PASSWORD=" + pass}
	case domain.EngineMongoDB:
		return []string{"MONGO_INITDB_ROOT_USERNAME=" + user, "MONGO_INITDB_ROOT_PASSWORD=" + pass, "MONGO_INITDB_DATABASE=" + dbName}
	case domain.EngineMSSQL:
		return []string{"ACCEPT_EULA=Y", "MSSQL_SA_PASSWORD=" + pass, "MSSQL_PID=developer"}
	case domain.EngineOracle:
		return []string{"ORACLE_PASSWORD=" + pass, "ORACLE_DATABASE=" + dbName, "ORACLE_DISABLE_ASYNC_IO=TRUE"}
	}
	return nil
}

func (c *Core) dbReady(ctx context.Context, db *domain.Database, pass string) bool {
	var cmd []string
	switch db.Engine {
	case domain.EnginePostgres:
		cmd = []string{"pg_isready", "-U", db.User, "-d", db.DBName}
	case domain.EngineMysql, domain.EngineMariaDB:
		cmd = []string{"mysqladmin", "ping", "-u" + db.User, "-p" + pass}
	case domain.EngineRedis:
		cmd = []string{"redis-cli", "-a", pass, "ping"}
	case domain.EngineMongoDB:
		cmd = []string{"mongosh", "--quiet", "--eval", "db.runCommand({ping:1}).ok", "-u", db.User, "-p", pass, "--authenticationDatabase", "admin"}
	case domain.EngineMSSQL:
		cmd = []string{"sh", "-c", "SQLCMD=/opt/mssql-tools18/bin/sqlcmd; [ -x $SQLCMD ] || SQLCMD=sqlcmd; $SQLCMD -S localhost -U sa -P '" + pass + "' -Q 'SELECT 1' -C -l 2 >/dev/null 2>&1"}
	case domain.EngineOracle:
		cmd = []string{"sh", "-c", "echo \"SELECT 1 FROM dual;\" | sqlplus -s system/'" + pass + "'@//localhost:1521/" + strings.ToUpper(db.DBName) + " >/dev/null 2>&1"}
	default:
		return true
	}
	res, err := c.Driver.Exec(ctx, db.ContainerID, runtime.ExecRequest{Command: cmd})
	if err != nil {
		return false
	}
	if db.Engine == domain.EngineRedis {
		return strings.Contains(string(res.Stdout), "PONG")
	}
	if db.Engine == domain.EngineMongoDB {
		return strings.Contains(string(res.Stdout), "1")
	}
	return res.ExitCode == 0
}

func (c *Core) DatabaseConnectionString(db *domain.Database) (string, error) {
	pass, err := c.DatabasePassword(db)
	if err != nil {
		return "", err
	}
	host := "aether-db-" + db.Name
	switch db.Engine {
	case domain.EnginePostgres:
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", db.User, pass, host, db.Port, db.DBName), nil
	case domain.EngineMysql, domain.EngineMariaDB:
		return fmt.Sprintf("mysql://%s:%s@%s:%d/%s", db.User, pass, host, db.Port, db.DBName), nil
	case domain.EngineRedis:
		return fmt.Sprintf("redis://:%s@%s:%d/0", pass, host, db.Port), nil
	case domain.EngineMongoDB:
		return fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?authSource=admin", db.User, pass, host, db.Port, db.DBName), nil
	case domain.EngineMSSQL:
		return fmt.Sprintf("sqlserver://sa:%s@%s:%d?database=%s&trustServerCertificate=true", pass, host, db.Port, db.DBName), nil
	case domain.EngineOracle:
		return fmt.Sprintf("oracle://system:%s@%s:%d/%s", pass, host, db.Port, strings.ToUpper(db.DBName)), nil
	}
	return "", errors.New("engine desconhecida")
}

func (c *Core) BackupDatabase(dbID string) (*domain.Backup, error) {
	var result *domain.Backup
	var opErr error
	lockErr := c.withLock("lock:backup:"+dbID, lockBackupTTL, func() error {
		b, err := c.backupDatabase(dbID)
		if b != nil {
			result = b
		}
		opErr = err
		return nil
	})
	if lockErr != nil {
		return nil, lockErr
	}
	return result, opErr
}

func (c *Core) backupDatabase(dbID string) (*domain.Backup, error) {
	db, err := c.Store.GetDatabase(dbID)
	if err != nil {
		return nil, err
	}
	if db.Status != "ready" || db.ContainerID == "" {
		return nil, errors.New("banco não está pronto para backup")
	}
	c.publishBackupEvent(db, "backup.started", nil)
	pass, err := c.DatabasePassword(db)
	if err != nil {
		c.publishBackupEvent(db, "backup.failed", map[string]any{"error": err.Error()})
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	cmd := c.dumpCommand(db, pass)
	res, err := c.Driver.Exec(ctx, db.ContainerID, runtime.ExecRequest{Command: cmd})
	if err != nil {
		c.publishBackupEvent(db, "backup.failed", map[string]any{"error": err.Error()})
		return nil, err
	}
	if res.ExitCode != 0 && len(res.Stdout) == 0 {
		c.publishBackupEvent(db, "backup.failed", map[string]any{"error": fmt.Sprintf("dump exit %d", res.ExitCode)})
		return nil, fmt.Errorf("dump falhou (exit %d): %s", res.ExitCode, string(res.Stdout))
	}
	dir := filepath.Join(c.Cfg.StateDir, "backups")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	name := "db-" + db.Name + "-" + time.Now().UTC().Format("20060102T150405") + ".dump.gz"
	path := filepath.Join(dir, name)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gw := gzipWriter(f)
	if _, err := gw.Write(res.Stdout); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	b := &domain.Backup{
		ID:        domain.NewID(),
		Path:      path,
		Size:      info.Size(),
		CreatedAt: time.Now().UTC(),
	}
	if err := c.createBackupRecord(b, "database", "local", db.AppIDRef()); err != nil {
		return nil, err
	}
	c.FireWebhookEvent(context.Background(), db.OrgID, EvBackupDone, map[string]any{
		"database": db.Name,
		"engine":   string(db.Engine),
		"size":     info.Size(),
		"path":     path,
	})
	c.publishBackupEvent(db, "backup.finished", map[string]any{"size": info.Size()})
	return b, nil
}

func (c *Core) publishBackupEvent(db *domain.Database, eventType string, extra map[string]any) {
	payload := map[string]any{
		"org_id":    db.OrgID,
		"database":  db.Name,
		"engine":    string(db.Engine),
		"backup_id": db.ID,
	}
	for k, v := range extra {
		payload[k] = v
	}
	_ = c.Bus.Publish(context.Background(), "database", db.ID, eventType, payload, nil)
}

func (c *Core) dumpCommand(db *domain.Database, pass string) []string {
	switch db.Engine {
	case domain.EnginePostgres:
		return []string{"pg_dump", "-U", db.User, "-d", db.DBName, "-Fc"}
	case domain.EngineMysql, domain.EngineMariaDB:
		return []string{"mysqldump", "-u" + db.User, "-p" + pass, db.DBName}
	case domain.EngineRedis:
		return []string{"sh", "-c", "redis-cli -a " + pass + " save >/dev/null 2>&1; cat /data/dump.rdb"}
	case domain.EngineMongoDB:
		return []string{"mongodump", "--archive", "--gzip", "-u", db.User, "-p", pass, "--authenticationDatabase", "admin", "--db", db.DBName}
	case domain.EngineMSSQL:
		return []string{"sh", "-c", "SQLCMD=/opt/mssql-tools18/bin/sqlcmd; [ -x $SQLCMD ] || SQLCMD=sqlcmd; $SQLCMD -S localhost -U sa -P '" + pass + "' -C -Q \"BACKUP DATABASE [" + db.DBName + "] TO DISK='/tmp/aether.bak' WITH INIT, FORMAT\" >/dev/null 2>&1 && cat /tmp/aether.bak"}
	case domain.EngineOracle:
		return []string{"sh", "-c", "echo \"CREATE OR REPLACE DIRECTORY AETHER_DIR AS '/tmp';\" | sqlplus -s system/'" + pass + "'@//localhost:1521/" + strings.ToUpper(db.DBName) + " >/dev/null 2>&1; expdp system/'" + pass + "'@//localhost:1521/" + strings.ToUpper(db.DBName) + " DIRECTORY=AETHER_DIR DUMPFILE=aether.dmp FULL=Y logfile=expdp.log >/dev/null 2>&1; cat /tmp/aether.dmp"}
	}
	return nil
}

func (c *Core) RestoreDatabase(dbID string, backupID string) error {
	return c.withLock("lock:restore:"+dbID, lockBackupTTL, func() error {
		return c.restoreDatabase(dbID, backupID)
	})
}

func (c *Core) restoreDatabase(dbID string, backupID string) error {
	db, err := c.Store.GetDatabase(dbID)
	if err != nil {
		return err
	}
	b, err := c.Store.GetBackup(backupID)
	if err != nil {
		return err
	}
	if b.AppID != db.AppIDRef() {
		return errors.New("backup não pertence ao banco")
	}
	data, err := os.ReadFile(b.Path)
	if err != nil {
		return err
	}
	raw, err := gunzipBytes(data)
	if err != nil {
		return err
	}
	pass, err := c.DatabasePassword(db)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	cmd := c.restoreCommand(db, pass)
	req := runtime.ExecRequest{Command: cmd}
	stream, err := c.Driver.ExecStream(ctx, db.ContainerID, req)
	if err != nil {
		return err
	}
	defer stream.Close()
	if _, err := stream.Write(raw); err != nil {
		return err
	}
	code, err := stream.Wait()
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("restore falhou (exit %d)", code)
	}
	return nil
}

func (c *Core) restoreCommand(db *domain.Database, pass string) []string {
	switch db.Engine {
	case domain.EnginePostgres:
		return []string{"pg_restore", "-U", db.User, "-d", db.DBName, "--no-owner", "--clean", "--if-exists"}
	case domain.EngineMysql, domain.EngineMariaDB:
		return []string{"mysql", "-u" + db.User, "-p" + pass, db.DBName}
	case domain.EngineMongoDB:
		return []string{"mongorestore", "--archive", "--gzip", "-u", db.User, "-p", pass, "--authenticationDatabase", "admin"}
	case domain.EngineMSSQL:
		return []string{"sh", "-c", "cat > /tmp/aether.bak && SQLCMD=/opt/mssql-tools18/bin/sqlcmd; [ -x $SQLCMD ] || SQLCMD=sqlcmd; $SQLCMD -S localhost -U sa -P '" + pass + "' -C -Q \"RESTORE DATABASE [" + db.DBName + "] FROM DISK='/tmp/aether.bak' WITH REPLACE\""}
	case domain.EngineOracle:
		return []string{"sh", "-c", "cat > /tmp/aether.dmp && echo \"CREATE OR REPLACE DIRECTORY AETHER_DIR AS '/tmp';\" | sqlplus -s system/'" + pass + "'@//localhost:1521/" + strings.ToUpper(db.DBName) + " >/dev/null 2>&1; impdp system/'" + pass + "'@//localhost:1521/" + strings.ToUpper(db.DBName) + " DIRECTORY=AETHER_DIR DUMPFILE=aether.dmp FULL=Y TABLE_EXISTS_ACTION=REPLACE logfile=impdp.log"}
	default:
		return []string{"cat"}
	}
}

func (c *Core) DeleteDatabase(dbID string) error {
	db, err := c.Store.GetDatabase(dbID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if db.ContainerID != "" {
		c.Driver.Remove(ctx, db.ContainerID, true)
	}
	return c.Store.DeleteDatabase(dbID)
}

var _ = io.Discard
