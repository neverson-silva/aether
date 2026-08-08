package core

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"aether/internal/domain"
	"aether/internal/runtime"
)

func TestOutWebhookHMAC(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	user, org, err := c.CreateUserAndOrg("wh@aether.local", "wh", "senha-wh")
	if err != nil {
		t.Fatal(err)
	}
	_ = user
	secret := "segredo-test"
	delivered := make(chan map[string]any, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		sig := r.Header.Get("X-Aether-Signature")
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if sig != expected {
			t.Errorf("assinatura inválida: %s != %s", sig, expected)
		}
		if r.Header.Get("X-Aether-Event") != EvBackupDone {
			t.Errorf("evento errado: %s", r.Header.Get("X-Aether-Event"))
		}
		var payload map[string]any
		json.Unmarshal(body, &payload)
		delivered <- payload
		w.WriteHeader(200)
	}))
	defer server.Close()

	if _, err := c.CreateOutWebhook(org.ID, "backup-hook", server.URL, secret, []string{EvBackupDone}); err != nil {
		t.Fatal(err)
	}
	c.FireWebhookEvent(context.Background(), org.ID, EvBackupDone, map[string]any{"database": "pg1", "size": 42})
	select {
	case <-delivered:
	case <-time.After(5 * time.Second):
		t.Fatal("webhook não entregue")
	}
	select {
	case payload := <-delivered:
		if payload["payload"].(map[string]any)["database"] != "pg1" {
			t.Fatalf("payload errado: %v", payload)
		}
	default:
	}
}

func TestOutWebhookFilterEvents(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	_, org, err := c.CreateUserAndOrg("wh2@aether.local", "wh2", "senha-wh2")
	if err != nil {
		t.Fatal(err)
	}
	called := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called <- struct{}{}
	}))
	defer server.Close()
	if _, err := c.CreateOutWebhook(org.ID, "only-ready", server.URL, "", []string{EvDeployReady}); err != nil {
		t.Fatal(err)
	}
	c.FireWebhookEvent(context.Background(), org.ID, EvDeployFailed, map[string]any{})
	select {
	case <-called:
		t.Fatal("webhook não deveria ser chamado para evento não assinado")
	case <-time.After(300 * time.Millisecond):
	}
}

func TestRegistrySettingsRoundtrip(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	cfg, err := c.RegistrySettings()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 5000 || cfg.Status != "stopped" {
		t.Fatalf("default errado: %+v", cfg)
	}
	cfg.Enabled = true
	cfg.Status = "running"
	cfg.ContainerID = "abc123"
	if err := c.Store.SaveRegistrySettings(&cfg); err != nil {
		t.Fatal(err)
	}
	got, err := c.RegistrySettings()
	if err != nil || !got.Enabled || got.ContainerID != "abc123" {
		t.Fatalf("roundtrip falhou: %+v %v", got, err)
	}
}

func TestBuildSourceUnknownAndMissingTools(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	user, org, err := c.CreateUserAndOrg("nix@aether.local", "nix", "senha-nix")
	if err != nil {
		t.Fatal(err)
	}
	_ = user
	proj, err := c.CreateProject(org.ID, "nixproj")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp("", "nixsrc")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	os.WriteFile(dir+"/package.json", []byte(`{"name":"x"}`), 0o644)
	app := &domain.App{ID: domain.NewID(), OrgID: org.ID, ProjectID: proj.ID, Name: "nixapp", Dockerfile: "Dockerfile"}

	app.BuildType = "banana"
	if err := c.buildSource(context.Background(), app, dir, "aether.local/nixapp:1", nil); err == nil {
		t.Fatal("build_type desconhecido deveria falhar")
	}

	empty, _ := os.MkdirTemp("", "nobin")
	defer os.RemoveAll(empty)
	t.Setenv("PATH", empty)
	app.BuildType = "nixpacks"
	if err := c.buildSource(context.Background(), app, dir, "aether.local/nixapp:1", nil); err == nil || !strings.Contains(err.Error(), "nixpacks") {
		t.Fatalf("nixpacks ausente deveria falhar: %v", err)
	}
	app.BuildType = "buildpacks"
	if err := c.buildSource(context.Background(), app, dir, "aether.local/nixapp:1", nil); err == nil || !strings.Contains(err.Error(), "pack") {
		t.Fatalf("pack ausente deveria falhar: %v", err)
	}
	_ = runtime.Stats{}
}
