package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"aether/internal/modules/mirrors/domain"
	"aether/internal/platform/worker"
)

type Mirrors struct {
	Store   domain.Store
	Runtime worker.ImageRegistryRuntime
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
	if err := m.copyImage(ctx, mirror.Source, mirror.Dest, mirror.DestTLSVerify); err != nil {
		_ = m.Store.SetStatus(ctx, id, "error")
		return err
	}
	return m.Store.SetStatus(ctx, id, "synced")
}

func (m *Mirrors) copyImage(ctx context.Context, source, dest string, tlsVerify bool) error {
	if m.Runtime == nil {
		return domain.ErrValidation
	}
	if !tlsVerify {
		return fmt.Errorf("per-mirror TLS verification is not supported by the configured Docker client")
	}
	if _, err := m.Runtime.Pull(ctx, source); err != nil {
		return err
	}
	if err := m.Runtime.Tag(ctx, source, dest); err != nil {
		return err
	}
	if _, err := m.Runtime.Push(ctx, dest); err != nil {
		return err
	}
	return nil
}
