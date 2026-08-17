package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	appsInfra "aether/internal/apps/infra"
	"aether/internal/backups/domain"
	"aether/internal/backups/infra"
	dbdomain "aether/internal/databases/domain"
	databasesInfra "aether/internal/databases/infra"
)

type env struct {
	ctx       context.Context
	svc       *Backups
	databases *databasesInfra.Store
	orgID     uuid.UUID
	dbID      uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	databasesStore := databasesInfra.NewStore(pool)
	appsStore := appsInfra.NewStore(pool)
	e := &env{ctx: context.Background(), svc: &Backups{Store: store, Databases: databasesStore}, databases: databasesStore, orgID: uuid.New()}
	t.Cleanup(func() {
		_ = store.Close()
		_ = databasesStore.Close()
		_ = appsStore.Close()
	})
	if _, err := pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, e.orgID, "Bkp Org", "bkp-org"); err != nil {
		t.Fatalf("criar org: %v", err)
	}
	project, err := appsStore.CreateProject(e.ctx, e.orgID, "BkpProj", "bkp-proj", "", "")
	if err != nil {
		t.Fatalf("criar project: %v", err)
	}
	db, err := databasesStore.CreateDatabase(e.ctx, &dbdomain.Database{
		OrgID: e.orgID, ProjectID: project.ID, Name: "db", Engine: dbdomain.EnginePostgres,
		Version: "16", Port: 5432, DBName: "db", User: "aether", PassEnc: "x", Status: "creating",
	})
	if err != nil {
		t.Fatalf("criar database: %v", err)
	}
	e.dbID = db.ID
	return e
}

func TestDatabaseBackupLifecycle(t *testing.T) {
	e := newEnv(t)
	b, err := e.svc.CreateDatabase(e.ctx, e.dbID, e.orgID)
	if err != nil {
		t.Fatalf("backup db: %v", err)
	}
	if b.Kind != "db" || b.DatabaseID == nil || *b.DatabaseID != e.dbID {
		t.Fatalf("backup inesperado: %+v", b)
	}
	if b.Path == "" {
		t.Fatalf("path deveria estar presente")
	}

	if err := e.svc.RestoreDatabase(e.ctx, e.dbID, e.orgID, b.ID); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if err := e.svc.RestoreDatabase(e.ctx, e.dbID, e.orgID, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("backup inexistente deveria falhar: %v", err)
	}
}

func TestStateBackupLifecycle(t *testing.T) {
	e := newEnv(t)
	b, err := e.svc.CreateState(e.ctx, e.orgID)
	if err != nil {
		t.Fatalf("state backup: %v", err)
	}
	if b.Kind != "state" || b.DatabaseID != nil {
		t.Fatalf("state backup inesperado: %+v", b)
	}
	if err := e.svc.RestoreState(e.ctx, b.ID, e.orgID); err != nil {
		t.Fatalf("restore state: %v", err)
	}
}

func TestBackupIsolation(t *testing.T) {
	e := newEnv(t)
	b, _ := e.svc.CreateDatabase(e.ctx, e.dbID, e.orgID)
	if err := e.svc.RestoreDatabase(e.ctx, e.dbID, uuid.New(), b.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("outra org deveria falhar: %v", err)
	}
}

func TestListBackups(t *testing.T) {
	e := newEnv(t)
	_, _ = e.svc.CreateDatabase(e.ctx, e.dbID, e.orgID)
	_, _ = e.svc.CreateState(e.ctx, e.orgID)
	list, err := e.svc.List(e.ctx, e.orgID, 50)
	if err != nil || len(list) != 2 {
		t.Fatalf("list: %v %d", err, len(list))
	}
	if list[0].CreatedAt.Before(list[1].CreatedAt) {
		t.Fatalf("list deveria ser DESC")
	}
}
