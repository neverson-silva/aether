package mssql

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"aether/internal/modules/backups/adapters/container"
	"aether/internal/modules/backups/application"
)

func init() {
	application.RegisterBackupAdapter(New(container.NewPodman()))
}

const backupDir = "/var/opt/mssql/backups"

type Adapter struct {
	exec container.Executor
}

func New(exec container.Executor) *Adapter {
	return &Adapter{exec: exec}
}

func (a *Adapter) Engine() application.BackupEngine { return application.EngineMSSQL }
func (a *Adapter) Format() string                   { return "bak" }
func (a *Adapter) ContentType() string              { return "application/octet-stream" }

func sqlEnv(pass string) []string {
	return []string{"SQLCMDPASSWORD=" + pass}
}

func sqlcmd(db application.DBDescriptor) []string {
	return []string{"sqlcmd", "-C", "-S", "localhost", "-U", db.User}
}

func quoteIdent(name string) string {
	return strings.ReplaceAll(name, "]", "]]")
}

func failureMessage(stdout, stderr string) string {
	msg := strings.TrimSpace(stdout + " " + stderr)
	if len(msg) > 400 {
		msg = msg[len(msg)-400:]
	}
	return msg
}

func (a *Adapter) Validate(ctx context.Context, db application.DBDescriptor) error {
	stdout, stderr, err := a.exec.Exec(ctx, db.ContainerID, sqlEnv(db.Password), append(sqlcmd(db), "-Q", "SELECT @@VERSION")...)
	if err != nil {
		return fmt.Errorf("sqlcmd unavailable: %s", failureMessage(stdout, stderr))
	}
	return nil
}

func (a *Adapter) CreateBackup(ctx context.Context, db application.DBDescriptor, dest io.Writer) error {
	if err := a.ensureBackupDir(ctx, db); err != nil {
		return err
	}
	path := fmt.Sprintf("%s/%s-%s.bak", backupDir, application.SafeToken(db.DBName), time.Now().UTC().Format("20060102T150405Z"))
	query := fmt.Sprintf("BACKUP DATABASE [%s] TO DISK = N'%s' WITH COMPRESSION, CHECKSUM, INIT", quoteIdent(db.DBName), path)
	stdout, stderr, err := a.exec.Exec(ctx, db.ContainerID, sqlEnv(db.Password), append(sqlcmd(db), "-Q", query)...)
	if err != nil {
		return fmt.Errorf("BACKUP DATABASE failed: %s", failureMessage(stdout, stderr))
	}
	if err := a.exec.ExecStream(ctx, db.ContainerID, nil, dest, io.Discard, "cat", path); err != nil {
		return fmt.Errorf("read backup file from container: %w", err)
	}
	_, _, _ = a.exec.Exec(ctx, db.ContainerID, nil, "rm", "-f", path)
	return nil
}

func (a *Adapter) Restore(ctx context.Context, db application.DBDescriptor, src io.Reader) error {
	if err := a.ensureBackupDir(ctx, db); err != nil {
		return err
	}
	path := fmt.Sprintf("%s/%s-restore-%s.bak", backupDir, application.SafeToken(db.DBName), time.Now().UTC().Format("20060102T150405Z"))
	if err := a.exec.ExecIn(ctx, db.ContainerID, nil, src, io.Discard, io.Discard, "tee", path); err != nil {
		return fmt.Errorf("write backup file into container: %w", err)
	}
	query := fmt.Sprintf("RESTORE DATABASE [%s] FROM DISK = N'%s' WITH REPLACE, RECOVERY", quoteIdent(db.DBName), path)
	stdout, stderr, err := a.exec.Exec(ctx, db.ContainerID, sqlEnv(db.Password), append(sqlcmd(db), "-Q", query)...)
	if err != nil {
		_, _, _ = a.exec.Exec(ctx, db.ContainerID, nil, "rm", "-f", path)
		return fmt.Errorf("RESTORE DATABASE failed: %s", failureMessage(stdout, stderr))
	}
	_, _, _ = a.exec.Exec(ctx, db.ContainerID, nil, "rm", "-f", path)
	return nil
}

func (a *Adapter) ensureBackupDir(ctx context.Context, db application.DBDescriptor) error {
	stdout, stderr, err := a.exec.Exec(ctx, db.ContainerID, sqlEnv(db.Password), append(sqlcmd(db), "-Q", fmt.Sprintf("EXEC master.dbo.xp_create_subdir N'%s'", backupDir))...)
	if err != nil {
		return fmt.Errorf("prepare backup dir: %s", failureMessage(stdout, stderr))
	}
	return nil
}
