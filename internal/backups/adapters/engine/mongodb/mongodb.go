package mongodb

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

func (a *Adapter) Engine() application.BackupEngine { return application.EngineMongoDB }
func (a *Adapter) Format() string                   { return "archive" }
func (a *Adapter) ContentType() string              { return "application/gzip" }

func (a *Adapter) uri(db application.DBDescriptor) string {
	return fmt.Sprintf("mongodb://%s:%s@localhost:27017/%s?authSource=admin", db.User, db.Password, db.DBName)
}

func (a *Adapter) Validate(ctx context.Context, db application.DBDescriptor) error {
	_, stderr, err := a.exec.Exec(ctx, db.ContainerID, nil, "mongodump", "--version")
	if err != nil {
		return fmt.Errorf("mongodump unavailable: %s", strings.TrimSpace(stderr))
	}
	return nil
}

func (a *Adapter) CreateBackup(ctx context.Context, db application.DBDescriptor, dest io.Writer) error {
	args := []string{"mongodump", "--uri", a.uri(db), "--archive", "--gzip"}
	return a.exec.ExecStream(ctx, db.ContainerID, nil, dest, io.Discard, args...)
}

func (a *Adapter) Restore(ctx context.Context, db application.DBDescriptor, src io.Reader) error {
	args := []string{"mongorestore", "--uri", a.uri(db), "--archive", "--gzip"}
	return a.exec.ExecIn(ctx, db.ContainerID, nil, src, io.Discard, io.Discard, args...)
}
