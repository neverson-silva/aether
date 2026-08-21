package mysql

import (
	"context"
	"fmt"
	"io"
	"strings"

	"aether/internal/backups/adapters/container"
	"aether/internal/backups/application"
)

func init() {
	application.RegisterBackupAdapter(New(container.NewPodman()))
}

type Adapter struct {
	exec container.Executor
}

func New(exec container.Executor) *Adapter {
	return &Adapter{exec: exec}
}

func (a *Adapter) Engine() application.BackupEngine { return application.EngineMySQL }
func (a *Adapter) Format() string                   { return "sql" }
func (a *Adapter) ContentType() string              { return "text/plain" }

func (a *Adapter) Validate(ctx context.Context, db application.DBDescriptor) error {
	_, stderr, err := a.exec.Exec(ctx, db.ContainerID, mysqlEnv(db.Password), "mysqldump", "--version")
	if err != nil {
		return fmt.Errorf("mysqldump unavailable: %s", strings.TrimSpace(stderr))
	}
	return nil
}

func (a *Adapter) CreateBackup(ctx context.Context, db application.DBDescriptor, dest io.Writer) error {
	args := []string{
		"mysqldump", "-u", db.User, "--single-transaction", "--routines",
		"--triggers", "--hex-blob", "--no-tablespaces", db.DBName,
	}
	return a.exec.ExecStream(ctx, db.ContainerID, mysqlEnv(db.Password), dest, io.Discard, args...)
}

func (a *Adapter) Restore(ctx context.Context, db application.DBDescriptor, src io.Reader) error {
	args := []string{"mysql", "-u", db.User, db.DBName}
	return a.exec.ExecIn(ctx, db.ContainerID, mysqlEnv(db.Password), src, io.Discard, io.Discard, args...)
}

func mysqlEnv(pass string) []string {
	return []string{"MYSQL_PWD=" + pass}
}
