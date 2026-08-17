package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"aether/internal/mirrors/domain"
	"aether/internal/mirrors/infra"
)

type env struct {
	ctx context.Context
	svc *Mirrors
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	e := &env{ctx: context.Background(), svc: &Mirrors{Store: store}}
	t.Cleanup(func() { _ = store.Close() })
	return e
}

func TestMirrorLifecycle(t *testing.T) {
	e := newEnv(t)
	m, err := e.svc.Create(e.ctx, "dockerhub", "docker.io/library/nginx", "127.0.0.1:5000/nginx", true, "latest", "0 3 * * *")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if m.Status != "idle" || !m.DestTLSVerify {
		t.Fatalf("mirror inesperado: %+v", m)
	}
	if _, err := e.svc.Create(e.ctx, "", "s", "d", true, "", ""); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("name vazio deveria falhar: %v", err)
	}
	list, err := e.svc.List(e.ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}
	if err := e.svc.Delete(e.ctx, m.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestMirrorRunNotFound(t *testing.T) {
	e := newEnv(t)
	if err := e.svc.Run(e.ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("mirror inexistente deveria falhar: %v", err)
	}
}
