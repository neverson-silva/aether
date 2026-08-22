package application

import (
	"context"
	"os/exec"
	"strings"

	"github.com/google/uuid"

	"aether/internal/modules/mirrors/domain"
)

type Mirrors struct {
	Store domain.Store
}

func (m *Mirrors) Create(ctx context.Context, name, source, dest string, destTLSVerify bool, tagsFilter, schedule string) (*domain.Mirror, error) {
	name = strings.TrimSpace(name)
	source = strings.TrimSpace(source)
	dest = strings.TrimSpace(dest)
	if name == "" || source == "" || dest == "" {
		return nil, domain.ErrValidation
	}
	return m.Store.CreateMirror(ctx, &domain.Mirror{
		Name: name, Source: source, Dest: dest, DestTLSVerify: destTLSVerify,
		TagsFilter: strings.TrimSpace(tagsFilter), Schedule: strings.TrimSpace(schedule), Status: "idle",
	})
}

func (m *Mirrors) List(ctx context.Context) ([]domain.Mirror, error) {
	return m.Store.ListMirrors(ctx)
}

func (m *Mirrors) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Store.DeleteMirror(ctx, id)
}

func (m *Mirrors) Run(ctx context.Context, id uuid.UUID) error {
	mirror, err := m.Store.GetMirror(ctx, id)
	if err != nil {
		return err
	}
	if err := m.Store.SetStatus(ctx, id, "syncing"); err != nil {
		return err
	}
	if err := copyImage(ctx, mirror.Source, mirror.Dest, mirror.DestTLSVerify); err != nil {
		_ = m.Store.SetStatus(ctx, id, "error")
		return err
	}
	return m.Store.SetStatus(ctx, id, "synced")
}

func copyImage(ctx context.Context, source, dest string, tlsVerify bool) error {
	pull := exec.CommandContext(ctx, "podman", "pull", source)
	if out, err := pull.CombinedOutput(); err != nil {
		return errWith(out, err)
	}
	args := []string{"push", source, dest}
	if !tlsVerify {
		args = append(args, "--tls-verify=false")
	}
	push := exec.CommandContext(ctx, "podman", args...)
	if out, err := push.CombinedOutput(); err != nil {
		return errWith(out, err)
	}
	return nil
}

func errWith(out []byte, err error) error {
	if trimmed := strings.TrimSpace(string(out)); trimmed != "" {
		return &exec.ExitError{Stderr: []byte(trimmed)}
	}
	return err
}
