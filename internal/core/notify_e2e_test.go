package core_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"aether/internal/api"
	"aether/internal/config"
	"aether/internal/core"
	"aether/internal/db"
	"aether/internal/domain"
)

func cleanupTestContainers() {
	out, err := exec.Command("podman", "ps", "-a", "--filter", "label=aether.test=1", "--format", "{{.ID}}").Output()
	if err == nil {
		for _, id := range strings.Fields(string(out)) {
			_ = exec.Command("podman", "rm", "-f", id).Run()
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func bootstrapTestCore(t *testing.T) (*core.Core, *domain.Org, string) {
	t.Helper()
	t.Setenv("AETHER_RUNTIME", "podman")
	state := t.TempDir()
	cfg := db.TestConfig(t)
	cfg.DatabaseSchema = "t_e2e" + db.SchemaNameFor(t.Name())
	cfg.StateDir = state
	cfg.DataDir = state + "/data"
	cfg.CertsDir = state + "/certs"
	cfg.LogsDir = state + "/logs"
	cfg.BuildsDir = state + "/builds"
	cfg.CacheDir = state + "/cache"
	cfg.KeysDir = state + "/keys"
	cfg.APIAddr = "127.0.0.1:0"
	cfg.AgentAddr = "127.0.0.1:0"
	cfg.ChallengeAddr = "127.0.0.1:0"
	c, err := core.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	c.SetTestMode()
	t.Cleanup(func() {
		cleanupTestContainers()
		db.CleanupTestSchema(cfg)
		c.Stop(context.Background())
	})
	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, org, err := c.CreateUserAndOrg("e2e@aether.local", "e2e", "senha-e2e")
	if err != nil {
		t.Fatal(err)
	}
	_, token, err := c.Login("e2e@aether.local", "senha-e2e", "")
	if err != nil {
		t.Fatal(err)
	}
	return c, org, token
}

func TestNotificationsE2ETwoSessionsAndOffline(t *testing.T) {
	c, org, token := bootstrapTestCore(t)

	proj, _ := c.CreateProject(org.ID, "e2eproj")
	envs, _ := c.ListEnvironments(proj.ID)
	app := &domain.App{ID: domain.NewID(), OrgID: org.ID, ProjectID: proj.ID, EnvironmentID: envs[0].ID, Name: "api", SourceType: domain.SourceImage, Image: "nginx:alpine", Port: 18080}
	if err := c.CreateApp(org.ID, app); err != nil {
		t.Fatal(err)
	}

	srv := api.New(c, "")
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqA, _ := http.NewRequest("GET", ts.URL+"/api/v1/events/stream?token="+token, nil)
	respA, err := (&http.Client{Timeout: 90 * time.Second}).Do(reqA)
	if err != nil {
		t.Fatal(err)
	}
	defer respA.Body.Close()
	streamA := make(chan string, 20)
	go func() {
		buf := make([]byte, 8192)
		acc := ""
		for {
			n, err := respA.Body.Read(buf)
			if n > 0 {
				acc += string(buf[:n])
				for {
					idx := strings.Index(acc, "\n\n")
					if idx < 0 {
						break
					}
					streamA <- acc[:idx]
					acc = acc[idx+2:]
				}
			}
			if err != nil {
				return
			}
		}
	}()

	if _, err := c.Deploy(app.ID, core.DeployOpts{Trigger: "api", TriggeredBy: "e2e@aether.local"}); err != nil {
		t.Fatal(err)
	}

	gotTypes := map[string]bool{}
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) && !gotTypes["deployment.ready"] {
		select {
		case frame := <-streamA:
			if i := strings.Index(frame, "data: "); i >= 0 {
				var payload map[string]any
				if json.Unmarshal([]byte(frame[i+6:]), &payload) == nil {
					if typ, ok := payload["type"].(string); ok {
						gotTypes[typ] = true
						t.Logf("sessão A SSE: %s", payload["message"])
					}
				}
			}
		case <-time.After(2 * time.Second):
		}
	}
	for _, required := range []string{"deployment.queued", "deployment.building", "deployment.ready"} {
		if !gotTypes[required] {
			t.Fatalf("sessão A não recebeu %s em tempo real (recebeu: %v)", required, gotTypes)
		}
	}

	notifs := apiList(t, ts.URL+"/api/v1/notifications", token)
	if len(notifs) == 0 {
		t.Fatal("sessão B (offline) não encontrou notificações ao conectar")
	}
	count := apiUnread(t, ts.URL+"/api/v1/notifications/unread-count", token)
	if count == 0 {
		t.Fatal("unread count deveria ser > 0 após eventos")
	}
	t.Logf("sessão B: %d notificações, %d não lidas", len(notifs), count)

	reqRA, _ := http.NewRequest("POST", ts.URL+"/api/v1/notifications/read-all", nil)
	reqRA.Header.Set("Authorization", "Bearer "+token)
	if resp, err := http.DefaultClient.Do(reqRA); err == nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	if after := apiUnread(t, ts.URL+"/api/v1/notifications/unread-count", token); after != 0 {
		t.Fatalf("após read-all, unread deveria ser 0, foi %d", after)
	}
}

func apiList(t *testing.T, url, token string) []map[string]any {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func apiUnread(t *testing.T, url, token string) int {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out.Count
}

var _ = config.DefaultStateDir
