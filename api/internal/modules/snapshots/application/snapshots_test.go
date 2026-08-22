package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"aether/internal/modules/snapshots/domain"
	"aether/internal/modules/snapshots/infra"
)

type env struct {
	ctx   context.Context
	svc   *Snapshots
	orgID uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	e := &env{ctx: context.Background(), svc: &Snapshots{Store: store}, orgID: uuid.New()}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, e.orgID, "Snap Org", "snap-org"); err != nil {
		t.Fatalf("criar org: %v", err)
	}
	return e
}

func TestSnapshotLifecycle(t *testing.T) {
	e := newEnv(t)
	sn, err := e.svc.Create(e.ctx, e.orgID, nil, "data", "snap-1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if sn.Volume != "data" || sn.Name != "snap-1" {
		t.Fatalf("snapshot inesperado: %+v", sn)
	}
	if _, err := e.svc.Create(e.ctx, e.orgID, nil, "", "x"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("volume vazio deveria falhar: %v", err)
	}

	if err := e.svc.Restore(e.ctx, sn.ID, e.orgID); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if err := e.svc.Restore(e.ctx, sn.ID, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("outra org deveria falhar: %v", err)
	}

	list, err := e.svc.List(e.ctx, e.orgID, 50)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}

	if err := e.svc.Delete(e.ctx, sn.ID, e.orgID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestScheduleLifecycle(t *testing.T) {
	e := newEnv(t)
	sched, err := e.svc.CreateSchedule(e.ctx, e.orgID, nil, "data", "hourly", "0 * * * *", 0, true)
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	if sched.Retention != 7 || !sched.Enabled {
		t.Fatalf("defaults inesperados: %+v", sched)
	}
	if _, err := e.svc.CreateSchedule(e.ctx, e.orgID, nil, "data", "bad", "not-cron", 5, true); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("cron inválido deveria falhar: %v", err)
	}

	list, err := e.svc.ListSchedules(e.ctx, e.orgID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list schedules: %v %d", err, len(list))
	}

	if err := e.svc.DeleteSchedule(e.ctx, sched.ID, e.orgID); err != nil {
		t.Fatalf("delete schedule: %v", err)
	}
}
