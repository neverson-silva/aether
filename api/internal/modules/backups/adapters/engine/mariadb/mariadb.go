package mariadb

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

func (a *Adapter) Engine() application.BackupEngine { return application.EngineMariaDB }
func (a *Adapter) Format() string                   { return "sql" }
func (a *Adapter) ContentType() string              { return "text/plain" }

func (a *Adapter) Validate(ctx context.Context, db application.DBDescriptor) error {
	_, stderr, err := a.exec.Exec(ctx, db.ContainerID, mysqlEnv(db.Password), "mariadb-dump", "--version")
	if err != nil {
		return fmt.Errorf("mariadb-dump unavailable: %s", strings.TrimSpace(stderr))
	}
	return nil
}

func (a *Adapter) CreateBackup(ctx context.Context, db application.DBDescriptor, dest io.Writer) error {
	args := []string{
		"mariadb-dump", "-u", db.User, "--single-transaction", "--routines",
		"--triggers", "--hex-blob", db.DBName,
	}
	return a.exec.ExecStream(ctx, db.ContainerID, mysqlEnv(db.Password), dest, io.Discard, args...)
}

func (a *Adapter) Restore(ctx context.Context, db application.DBDescriptor, src io.Reader) error {
	args := []string{"mariadb", "-u", db.User, db.DBName}
	var stderr strings.Builder
	if err := a.exec.ExecIn(ctx, db.ContainerID, mysqlEnv(db.Password), src, io.Discard, &stderr, args...); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("mariadb restore failed: %s", message)
	}
	return nil
}

func mysqlEnv(pass string) []string {
	return []string{"MYSQL_PWD=" + pass}
}
