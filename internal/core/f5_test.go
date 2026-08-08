package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aether/internal/domain"
)

func TestBrandingRoundtrip(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	_, org, err := c.CreateUserAndOrg("br@aether.local", "br", "senha-br")
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.GetBranding(org.ID)
	if err != nil || b.PrimaryColor != "" {
		t.Fatalf("default: %+v %v", b, err)
	}
	b.Name = "Acme Cloud"
	b.PrimaryColor = "#ff8800"
	if err := c.SaveBranding(&b); err != nil {
		t.Fatal(err)
	}
	got, _ := c.GetBranding(org.ID)
	if got.Name != "Acme Cloud" || got.PrimaryColor != "#ff8800" {
		t.Fatalf("roundtrip: %+v", got)
	}
}

func TestPipelineLifecycle(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	_, org, err := c.CreateUserAndOrg("pl@aether.local", "pl", "senha-pl")
	if err != nil {
		t.Fatal(err)
	}
	proj, _ := c.CreateProject(org.ID, "plproj")
	app := &domain.App{ID: domain.NewID(), OrgID: org.ID, ProjectID: proj.ID, Name: "plapp"}
	c.CreateApp(org.ID, app)
	stages := []domain.PipelineStage{{Name: "lint", Image: "busybox", Commands: []string{"sh", "-c", "echo lint-ok"}}}
	p, err := c.CreatePipeline(org.ID, app.ID, "ci", "manual", stages)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Stages) != 1 || p.Trigger != "manual" {
		t.Fatalf("pipeline: %+v", p)
	}
	run, err := c.RunPipeline(context.Background(), p.ID, "manual")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "success" {
		t.Fatalf("run: %s log=%s", run.Status, run.Log)
	}
	runs, err := c.Store.ListPipelineRuns(p.ID, 5)
	if err != nil || len(runs) != 1 {
		t.Fatalf("runs: %d %v", len(runs), err)
	}
}

func TestOIDCFlowWithMockProvider(t *testing.T) {
	// mock do provider OIDC
	srvURL := ""
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 srvURL,
			"authorization_endpoint": srvURL + "/auth",
			"token_endpoint":         srvURL + "/token",
			"userinfo_endpoint":      srvURL + "/userinfo",
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"access_token": "at-123", "id_token": "it-123"})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer at-123" {
			http.Error(w, "no token", http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"sub": "u1", "email": "oidc@aether.local", "name": "OIDC User"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL

	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	disc, err := c.DiscoverOIDC(context.Background(), srv.URL)
	if err != nil || disc.TokenEndpoint == "" {
		t.Fatalf("discovery: %+v %v", disc, err)
	}
	_, org, err := c.CreateUserAndOrg("owner@aether.local", "owner", "senha-owner")
	if err != nil {
		t.Fatal(err)
	}
	p, err := c.CreateOIDCProvider(org.ID, "mock", srv.URL, "client-1", "secret-1", "openid email")
	if err != nil {
		t.Fatal(err)
	}
	authURL, err := c.OIDCAuthURL(p)
	if err != nil || authURL == "" || !containsString([]string{authURL[:1]}, "h") {
		t.Fatalf("auth url: %s %v", authURL, err)
	}
	user, err := c.OIDCExchange(context.Background(), p, "code-xyz")
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "oidc@aether.local" {
		t.Fatalf("user: %+v", user)
	}
	u, token, err := c.OIDCLogin(user)
	if err != nil || token == "" || u.Email != "oidc@aether.local" {
		t.Fatalf("login: %+v %v %v", u, token, err)
	}
}

func TestClusterAffinity(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	_, org, err := c.CreateUserAndOrg("cl@aether.local", "cl", "senha-cl")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	srvA := &domain.Server{ID: "srv-a", Name: "a", Status: "healthy", Load: 0.5, LastHeartbeat: now, CreatedAt: now, ClusterID: "cl-1"}
	srvB := &domain.Server{ID: "srv-b", Name: "b", Status: "healthy", Load: 0.1, LastHeartbeat: now, CreatedAt: now}
	c.Store.CreateServer(srvA)
	c.Store.CreateServer(srvB)
	proj, _ := c.CreateProject(org.ID, "clproj")
	app := &domain.App{ID: domain.NewID(), OrgID: org.ID, ProjectID: proj.ID, Name: "clapp", ClusterID: "cl-1"}
	got, err := c.placeOnAgent(app)
	if err != nil || got != "srv-a" {
		t.Fatalf("afinidade de cluster: %s %v", got, err)
	}
	app.ClusterID = "cl-inexistente"
	got, _ = c.placeOnAgent(app)
	if got != "" {
		t.Fatalf("cluster vazio deveria cair pra local: %s", got)
	}
}
