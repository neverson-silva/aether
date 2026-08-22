package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"aether/internal/modules/clusters/domain"
	"aether/internal/modules/clusters/infra"
)

type env struct {
	ctx   context.Context
	pool  *pgxpool.Pool
	svc   *Clusters
	orgID uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	e := &env{ctx: context.Background(), pool: pool, svc: &Clusters{Store: store}, orgID: uuid.New()}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, e.orgID, "Clu Org", "clu-org"); err != nil {
		t.Fatalf("criar org: %v", err)
	}
	return e
}

func TestClusterLifecycle(t *testing.T) {
	e := newEnv(t)
	cluster, err := e.svc.CreateCluster(e.ctx, e.orgID, "prod", []string{"us-east"})
	if err != nil {
		t.Fatalf("create cluster: %v", err)
	}
	if _, err := e.svc.CreateCluster(e.ctx, e.orgID, "", nil); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("name vazio deveria falhar: %v", err)
	}

	list, err := e.svc.ListClusters(e.ctx, e.orgID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}

	if err := e.svc.DeleteCluster(e.ctx, cluster.ID, e.orgID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := e.svc.DeleteCluster(e.ctx, cluster.ID, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("outra org deveria falhar: %v", err)
	}
}

func TestServerAssignment(t *testing.T) {
	e := newEnv(t)
	cluster, _ := e.svc.CreateCluster(e.ctx, e.orgID, "prod", nil)
	var serverID uuid.UUID
	if err := e.pool.QueryRow(e.ctx, `INSERT INTO servers (name) VALUES ('node-1') RETURNING id`).Scan(&serverID); err != nil {
		t.Fatalf("criar server: %v", err)
	}

	if err := e.svc.AddServer(e.ctx, cluster.ID, e.orgID, serverID); err != nil {
		t.Fatalf("add server: %v", err)
	}
	if err := e.svc.RemoveServer(e.ctx, cluster.ID, e.orgID, serverID); err != nil {
		t.Fatalf("remove server: %v", err)
	}

	if err := e.svc.AddServer(e.ctx, uuid.New(), e.orgID, serverID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cluster inexistente deveria falhar: %v", err)
	}
}

func TestAgentToken(t *testing.T) {
	e := newEnv(t)
	token, err := e.svc.AgentToken(e.ctx)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if !strings.HasPrefix(token, "aether-agent_") {
		t.Fatalf("token inesperado")
	}
}

func TestRegistry(t *testing.T) {
	e := newEnv(t)
	registry, err := e.svc.GetRegistry(e.ctx)
	if err != nil {
		t.Fatalf("get registry: %v", err)
	}
	if registry.Enabled || registry.Port != 5000 {
		t.Fatalf("registry default inesperado: %+v", registry)
	}
	enabled, err := e.svc.SetRegistryEnabled(e.ctx, true)
	if err != nil || !enabled.Enabled {
		t.Fatalf("enable registry: %v %+v", err, enabled)
	}
	got, _ := e.svc.GetRegistry(e.ctx)
	if !got.Enabled {
		t.Fatalf("registry deveria persistir enabled")
	}
}
