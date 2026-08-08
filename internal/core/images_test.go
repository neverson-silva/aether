package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"aether/internal/domain"
)

func TestImageRetentionWithFakeRegistry(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	user, org, err := c.CreateUserAndOrg("img@aether.local", "img", "senha-img")
	if err != nil {
		t.Fatal(err)
	}
	_ = user
	proj, _ := c.CreateProject(org.ID, "img")
	envs, _ := c.ListEnvironments(proj.ID)
	app := &domain.App{ID: domain.NewID(), OrgID: org.ID, ProjectID: proj.ID, EnvironmentID: envs[0].ID, Name: "imgapp", SourceType: domain.SourceGit, GitURL: "https://x/y.git", Port: 80, ImageRetention: 2}
	if err := c.CreateApp(org.ID, app); err != nil {
		t.Fatal(err)
	}

	// registry fake: tags 1..6 de aether.local/imgapp, DELETE registrado
	var mu sync.Mutex
	deleted := []string{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method == "GET" && r.URL.Path == "/v2/aether.local/imgapp/tags/list" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"name": "aether.local/imgapp", "tags": []string{"1", "2", "3", "4", "5", "6"}})
			return
		}
		if r.Method == "HEAD" {
			w.Header().Set("Docker-Content-Digest", "sha256:"+r.URL.Path)
			w.WriteHeader(200)
			return
		}
		if r.Method == "DELETE" {
			deleted = append(deleted, r.URL.Path)
			w.WriteHeader(202)
			return
		}
		w.WriteHeader(404)
	}))
	defer ts.Close()
	host, port := hostPortFromTestServer(ts)

	if err := c.Store.SaveRegistrySettings(&domain.RegistrySettings{Enabled: true, Host: host, Port: port, Status: "running"}); err != nil {
		t.Fatal(err)
	}

	if err := c.RunImageRetention(timeoutCtxT()); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	got := append([]string{}, deleted...)
	mu.Unlock()
	// retenção 2 -> mantém 5 e 6; deleta 1..4
	if len(got) != 4 {
		t.Fatalf("esperado 4 deletes, obtido %d: %v", len(got), got)
	}
	allowed := map[string]bool{"1": true, "2": true, "3": true, "4": true}
	for _, p := range got {
		tag := p[strings.LastIndex(p, "/")+1:]
		if !allowed[tag] {
			t.Fatalf("tag inesperada deletada: %s", p)
		}
	}
}

func hostPortFromTestServer(ts *httptest.Server) (string, int) {
	// extrai host:port de ts.URL
	url := ts.URL
	host := url[7:stringsLastColon(url)]
	port := 0
	fmt.Sscanf(url[stringsLastColon(url)+1:], "%d", &port)
	return host, port
}

func stringsLastColon(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return i
		}
	}
	return -1
}

func containsSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
