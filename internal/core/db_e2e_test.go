package core

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"aether/internal/domain"
)

func TestDatabaseLifecycleWithPodman(t *testing.T) {
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("podman indisponível")
	}
	if err := exec.Command("podman", "info").Run(); err != nil {
		t.Skip("podman daemon indisponível")
	}
	cfg := testConfig(t)
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := c.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer c.Stop(context.Background())

	_, org, err := c.CreateUserAndOrg("dbe2e@aether.local", "dbe2e", "senha-db")
	if err != nil {
		t.Fatal(err)
	}
	project, err := c.CreateProject(org.ID, "dbs")
	if err != nil {
		t.Fatal(err)
	}
	db, err := c.CreateDatabase(org.ID, project.ID, "pg1", domain.EnginePostgres, "16", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Minute)
	var ready *domain.Database
	for time.Now().Before(deadline) {
		d, err := c.Store.GetDatabase(db.ID)
		if err != nil {
			t.Fatal(err)
		}
		if d.Status == "ready" {
			ready = d
			break
		}
		if d.Status == "failed" {
			t.Fatalf("banco falhou")
		}
		time.Sleep(2 * time.Second)
	}
	if ready == nil {
		t.Fatal("timeout aguardando banco pronto")
	}
	dsn, err := c.DatabaseConnectionString(ready)
	if err != nil || !strings.Contains(dsn, "aether-db-pg1") {
		t.Fatalf("dsn errado: %s %v", dsn, err)
	}
	pass, err := c.DatabasePassword(ready)
	if err != nil || pass == "" {
		t.Fatalf("senha: %v", err)
	}

	backup, err := c.BackupDatabase(ready.ID)
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if backup.Size == 0 {
		t.Fatal("backup vazio")
	}
	if backup.Kind != "database" || backup.AppID != ready.AppIDRef() {
		t.Fatalf("metadados do backup: %+v", backup)
	}
	data, err := os.ReadFile(backup.Path)
	if err != nil || len(data) == 0 {
		t.Fatalf("arquivo de backup: %v", err)
	}

	if err := c.DeleteDatabase(ready.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Store.GetDatabase(ready.ID); err == nil {
		t.Fatal("banco deveria ter sido removido")
	}
}
