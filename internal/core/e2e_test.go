package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"aether/internal/config"
	"aether/internal/db"
	"aether/internal/domain"
)

func podmanAvailable() bool {
	if _, err := exec.LookPath("podman"); err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := exec.CommandContext(ctx, "podman", "info").Run(); err != nil {
		return false
	}
	return true
}

func testConfig(t *testing.T) *config.Config {
	t.Setenv("AETHER_RUNTIME", "podman")
	state := t.TempDir()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	challengePort := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	ln.Close()
	cfg := db.ConfigFromEnvPublic(t)
	cfg.DatabaseSchema = "t_" + schemaForTest(t)
	cfg.StateDir = state
	cfg.DataDir = filepath.Join(state, "data")
	cfg.CertsDir = filepath.Join(state, "certs")
	cfg.LogsDir = filepath.Join(state, "logs")
	cfg.BuildsDir = filepath.Join(state, "builds")
	cfg.CacheDir = filepath.Join(state, "cache")
	cfg.KeysDir = filepath.Join(state, "keys")
	cfg.APIAddr = "127.0.0.1:0"
	cfg.AgentAddr = "127.0.0.1:0"
	cfg.ChallengeAddr = "127.0.0.1:" + challengePort
	cfg.CertEmail = ""
	cfg.ACMEDirectory = ""
	t.Cleanup(func() {
		db.CleanupTestSchema(cfg)
		os.RemoveAll(state)
	})
	return cfg
}

func schemaForTest(t *testing.T) string {
	sum := sha256.Sum256([]byte(t.Name()))
	return hex.EncodeToString(sum[:])[:10]
}

func TestFullLifecycleWithPodman(t *testing.T) {
	if !podmanAvailable() {
		t.Skip("podman indisponível — teste E2E ignorado")
	}
	step := func(s string) { t.Logf("[e2e] %s", s) }
	step("config")
	cfg := testConfig(t)
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	step("start")
	if err := c.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer c.Stop(context.Background())

	user, org, err := c.CreateUserAndOrg("e2e@aether.local", "e2e", "senha-e2e")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.Login("e2e@aether.local", "senha-e2e", ""); err != nil {
		t.Fatalf("login: %v", err)
	}
	project, err := c.CreateProject(org.ID, "demo")
	if err != nil {
		t.Fatal(err)
	}
	app := &domain.App{
		ID:         domain.NewID(),
		ProjectID:  project.ID,
		Name:       "web",
		SourceType: domain.SourceImage,
		Image:      "nginx:alpine",
		Port:       18080,
		HealthCheck: domain.HealthCheck{
			Enabled:    true,
			Path:       "/",
			IntervalMS: 1000,
			TimeoutMS:  2000,
			Retries:    30,
		},
	}
	step("create-app")
	if err := c.CreateApp(org.ID, app); err != nil {
		t.Fatal(err)
	}
	if err := c.Store.SaveDeploymentPlan(&domain.DeploymentPlan{
		ID:        domain.NewID(),
		AppID:     app.ID,
		WebServer: "nginx",
	}); err != nil {
		t.Fatal(err)
	}
	if err := c.SetAppEnv(app.ID, "MODE", "test", false); err != nil {
		t.Fatal(err)
	}
	if err := c.SetAppEnv(app.ID, "TOKEN", "segredo-123", true); err != nil {
		t.Fatal(err)
	}

	first, err := c.Deploy(app.ID, DeployOpts{Trigger: "e2e"})
	if err != nil {
		t.Fatal(err)
	}
	waitDeployment(t, c, first.ID, domain.DeploymentReady)

	dep1, err := c.Store.GetDeployment(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dep1.ContainerID == "" {
		t.Fatal("container não registrado no deployment")
	}
	info, err := c.Driver.Inspect(ctx, dep1.ContainerID)
	if err != nil {
		t.Fatal(err)
	}
	if info.State != "running" {
		t.Fatalf("container deveria estar running: %s", info.State)
	}
	st, err := c.Driver.Stats(ctx, dep1.ContainerID)
	if err != nil {
		t.Fatal(err)
	}
	if st.MemBytes == 0 {
		t.Fatal("stats sem memória")
	}

	ll := c.LiveLog(first.ID)
	if ll == nil {
		t.Fatal("live log não criado")
	}
	tail, err := ll.Tail(64 * 1024)
	if err != nil || len(tail) == 0 {
		t.Fatalf("log vazio: %v", err)
	}

	env, err := c.EnsureAppEnv(app.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, e := range env {
		found[e] = true
	}
	if !found["MODE=test"] || !found["TOKEN=segredo-123"] {
		t.Fatalf("env não resolvido: %v", env)
	}

	second, err := c.Deploy(app.ID, DeployOpts{Trigger: "e2e"})
	if err != nil {
		t.Fatal(err)
	}
	waitDeployment(t, c, second.ID, domain.DeploymentReady)
	if _, err := c.Driver.Inspect(ctx, dep1.ContainerID); err == nil {
		t.Fatal("container do deployment anterior deveria ter sido removido")
	}

	rolled, err := c.Rollback(app.ID)
	if err != nil {
		t.Fatal(err)
	}
	waitDeployment(t, c, rolled.ID, domain.DeploymentReady)
	rb, err := c.Store.GetDeployment(rolled.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rb.ImageRef != dep1.ImageRef {
		t.Fatalf("rollback deveria usar imagem do deploy 1: %q != %q", rb.ImageRef, dep1.ImageRef)
	}

	step("cleanup")
	if err := c.StopActiveContainers(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Driver.Inspect(ctx, rb.ContainerID); err == nil {
		t.Fatal("container deveria ter sido parado no cleanup")
	}
	_ = user
}

func waitDeployment(t *testing.T, c *Core, id string, want domain.DeploymentStatus) {
	t.Helper()
	deadline := time.Now().Add(120 * time.Second)
	for time.Now().Before(deadline) {
		dep, err := c.Store.GetDeployment(id)
		if err != nil {
			t.Fatal(err)
		}
		if dep.Status == want {
			return
		}
		if dep.Status == domain.DeploymentFailed {
			t.Fatalf("deployment falhou: %s", dep.Error)
		}
		if dep.Status == domain.DeploymentCancelled {
			t.Fatalf("deployment cancelado")
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal(fmt.Sprintf("timeout esperando %s", want))
}
