package postgres

import (
	"context"
	"fmt"
	"io"
	"strings"

	"aether/internal/modules/backups/adapters/container"
	"aether/internal/modules/backups/application"
)

type Adapter struct {
	exec container.Executor
}

func New(exec container.Executor) *Adapter {
	return &Adapter{exec: exec}
}

func (a *Adapter) Engine() application.BackupEngine { return application.EnginePostgres }
func (a *Adapter) Format() string                   { return "dump" }
func (a *Adapter) ContentType() string              { return "application/octet-stream" }

func (a *Adapter) Validate(ctx context.Context, db application.DBDescriptor) error {
	_, stderr, err := a.exec.Exec(ctx, db.ContainerID, pgEnv(db.Password), "pg_dump", "--version")
	if err != nil {
		return fmt.Errorf("pg_dump unavailable: %s", strings.TrimSpace(stderr))
	}
	return nil
}

func (a *Adapter) CreateBackup(ctx context.Context, db application.DBDescriptor, dest io.Writer) error {
	args := []string{"pg_dump", "-U", db.User, "-d", db.DBName, "-Fc", "-Z", "9", "--no-owner", "--no-privileges"}
	return a.exec.ExecStream(ctx, db.ContainerID, pgEnv(db.Password), dest, io.Discard, args...)
}

func (a *Adapter) Restore(ctx context.Context, db application.DBDescriptor, src io.Reader) error {
	args := []string{"pg_restore", "-U", db.User, "-d", db.DBName, "--clean", "--if-exists", "--no-owner", "--no-privileges", "--exit-on-error"}
	if db.Format == "sql" || db.Format == "sql.gz" || db.Format == "gzip" {
		args = []string{"psql", "-U", db.User, "-d", db.DBName, "--set", "ON_ERROR_STOP=1", "--single-transaction"}
	}
	var stderr strings.Builder
	if err := a.exec.ExecIn(ctx, db.ContainerID, pgEnv(db.Password), src, io.Discard, &stderr, args...); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("pg_restore failed: %s", message)
	}
	return nil
}

func pgEnv(pass string) []string {
	return []string{"PGPASSWORD=" + pass}
}
