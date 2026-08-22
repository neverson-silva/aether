package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	appsInfra "aether/internal/modules/apps/infra"
	"aether/internal/modules/databases/domain"
	"aether/internal/modules/databases/infra"
	"aether/internal/platform/hostinfo"
)

type env struct {
	ctx       context.Context
	svc       *Databases
	orgID     uuid.UUID
	projectID uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	appsStore := appsInfra.NewStore(pool)
	cipher, err := infra.NewPasswordCipher([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	e := &env{ctx: context.Background(), svc: &Databases{Store: store, Apps: appsStore, Passwords: cipher}, orgID: uuid.New()}
	t.Cleanup(func() {
		_ = store.Close()
		_ = appsStore.Close()
	})
	if _, err := pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, e.orgID, "DB Org", "db-org"); err != nil {
		t.Fatalf("criar org: %v", err)
	}
	project, err := appsStore.CreateProject(e.ctx, e.orgID, "DBProj", "db-proj", "", "")
	if err != nil {
		t.Fatalf("criar project: %v", err)
	}
	e.projectID = project.ID
	return e
}

func TestDatabaseLifecycle(t *testing.T) {
	e := newEnv(t)
	db, err := e.svc.Create(e.ctx, e.orgID, e.projectID, "maindb", domain.EnginePostgres, "", "", "", 256, 1024)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if db.Status != "creating" || db.Port != 5432 || db.Version != "16" || db.User != "aether" {
		t.Fatalf("db inesperado: %+v", db)
	}
	if db.PassEnc == "" {
		t.Fatalf("senha deveria estar encriptada")
	}

	if _, err := e.svc.Create(e.ctx, e.orgID, e.projectID, "maindb", domain.EnginePostgres, "", "", "", 0, 0); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("nome duplicado deveria falhar: %v", err)
	}

	list, err := e.svc.List(e.ctx, e.orgID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}

	got, err := e.svc.Get(e.ctx, db.ID, e.orgID)
	if err != nil || got.Name != "maindb" {
		t.Fatalf("get: %v", err)
	}

	dsn, err := e.svc.ConnectionString(e.ctx, db.ID, e.orgID)
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}
	if !contains(dsn, "postgres://aether:") || !contains(dsn, "@"+hostinfo.PublicIP()+":5432/maindb") {
		t.Fatalf("dsn inesperado: %s", dsn)
	}

	if err := e.svc.Delete(e.ctx, db.ID, e.orgID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := e.svc.Get(e.ctx, db.ID, e.orgID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("db deveria ter sido removido: %v", err)
	}
}

func TestDatabaseValidation(t *testing.T) {
	e := newEnv(t)
	if _, err := e.svc.Create(e.ctx, e.orgID, e.projectID, "", domain.EnginePostgres, "", "", "", 0, 0); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("nome vazio deveria falhar: %v", err)
	}
	if _, err := e.svc.Create(e.ctx, e.orgID, e.projectID, "db", "oracle-fake", "", "", "", 0, 0); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("engine inválida deveria falhar: %v", err)
	}
}

func TestDatabaseIsolation(t *testing.T) {
	e := newEnv(t)
	db, _ := e.svc.Create(e.ctx, e.orgID, e.projectID, "db", domain.EngineRedis, "", "", "", 0, 0)
	if _, err := e.svc.Get(e.ctx, db.ID, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("outra org deveria falhar: %v", err)
	}
}

func TestDatabaseEngines(t *testing.T) {
	e := newEnv(t)
	cases := []struct {
		engine domain.Engine
		port   int
	}{
		{domain.EngineMysql, 3306},
		{domain.EngineMariaDB, 3306},
		{domain.EngineMongoDB, 27017},
		{domain.EngineMSSQL, 1433},
		{domain.EngineOracle, 1521},
	}
	for i, c := range cases {
		db, err := e.svc.Create(e.ctx, e.orgID, e.projectID, "db"+uuid.NewString()[:4], c.engine, "", "", "", 0, 0)
		if err != nil {
			t.Fatalf("engine %s: %v", c.engine, err)
		}
		if db.Port != c.port {
			t.Fatalf("engine %s deveria usar porta %d, got %d", c.engine, c.port, db.Port)
		}
		_ = i
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
