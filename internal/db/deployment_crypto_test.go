package db

import (
	"strings"
	"testing"
	"time"

	"aether/internal/domain"
	"aether/internal/security"
)

func TestDeploymentRepoEncryptsAtRest(t *testing.T) {
	dir := t.TempDir()
	secrets, err := security.LoadSecrets(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := testStore(t)
	s.Secrets = secrets

	app := &domain.App{ID: domain.NewID(), OrgID: "org1", ProjectID: "p1", Name: "demo"}
	if err := s.CreateApp(app); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	dep := &domain.Deployment{
		ID:          domain.NewID(),
		AppID:       app.ID,
		Number:      1,
		Status:      domain.DeploymentReady,
		ComposeYAML: "services:\n  web:\n    image: nginx\n    environment:\n      DB_PASSWORD: supersecret\n",
		DeploySpec:  `{"env":[{"name":"SECRET","value":"x"}]}`,
		EnvSnapshot: "DB_PASSWORD=supersecret",
		CreatedAt:   now,
		StartedAt:   now,
	}
	if err := s.CreateDeployment(dep); err != nil {
		t.Fatal(err)
	}

	// Verifica que no banco está criptografado (não contém o segredo em claro).
	var storedCompose, storedSpec, storedEnv string
	err = s.db.QueryRow(`SELECT compose_yaml, deploy_spec, env_snapshot FROM deployments WHERE id=?`, dep.ID).
		Scan(&storedCompose, &storedSpec, &storedEnv)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{storedCompose, storedSpec, storedEnv} {
		if strings.Contains(v, "supersecret") || strings.Contains(v, "nginx") {
			t.Fatalf("dado em claro no banco: %.40q", v)
		}
	}

	// Leitura transparente: consumidor recebe o texto plano.
	got, err := s.GetDeployment(dep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.ComposeYAML, "supersecret") || !strings.Contains(got.ComposeYAML, "nginx") {
		t.Fatalf("compose não descriptografado")
	}
	if !strings.Contains(got.DeploySpec, "SECRET") {
		t.Fatalf("spec não descriptografado")
	}
	if !strings.Contains(got.EnvSnapshot, "DB_PASSWORD=supersecret") {
		t.Fatalf("env_snapshot não descriptografado")
	}
}

func TestDeploymentRepoUpdateEncrypts(t *testing.T) {
	secrets, err := security.LoadSecrets(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := testStore(t)
	s.Secrets = secrets

	app := &domain.App{ID: domain.NewID(), OrgID: "org1", ProjectID: "p1", Name: "demo"}
	if err := s.CreateApp(app); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	dep := &domain.Deployment{ID: domain.NewID(), AppID: app.ID, Number: 1, Status: domain.DeploymentBuilding, CreatedAt: now, StartedAt: now}
	if err := s.CreateDeployment(dep); err != nil {
		t.Fatal(err)
	}
	dep.Status = domain.DeploymentReady
	dep.ComposeYAML = "services:\n  api:\n    env:\n      TOKEN: toksecret\n"
	dep.DeploySpec = `{"build":{"cmd":"x"}}`
	dep.EnvSnapshot = "TOKEN=toksecret"
	if err := s.UpdateDeployment(dep); err != nil {
		t.Fatal(err)
	}

	var stored string
	if err := s.db.QueryRow(`SELECT compose_yaml FROM deployments WHERE id=?`, dep.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, "toksecret") {
		t.Fatal("update não criptografou")
	}
	got, err := s.GetDeployment(dep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.ComposeYAML, "toksecret") {
		t.Fatal("update não descriptografou na leitura")
	}
}
