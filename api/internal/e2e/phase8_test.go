package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"aether/internal/platform/database"
)

func TestPhase8HTTPDeploymentPipeline(t *testing.T) {
	if os.Getenv("AETHER_E2E") != "1" {
		t.Skip("AETHER_E2E=1 is required")
	}
	baseURL := envOr("AETHER_E2E_API_URL", "http://127.0.0.1:8080")
	client, err := newE2EClient()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	email := fmt.Sprintf("phase8-%d@example.test", time.Now().UnixNano())
	register := map[string]string{"email": email, "name": "Phase 8", "password": "phase8-test-password"}
	var registered struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/auth/register", "", register, &registered); err != nil {
		t.Fatal(err)
	}
	var me struct {
		Org struct {
			ID string `json:"id"`
		} `json:"org"`
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/auth/me", "", nil, &me); err != nil {
		t.Fatal(err)
	}
	if me.Org.ID == "" {
		t.Fatal("registration did not create an organization")
	}

	var project struct {
		ID string `json:"id"`
	}
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/projects", me.Org.ID, map[string]string{"name": "Phase 8", "description": "E2E", "color": ""}, &project); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_ = client.request(cleanupCtx, http.MethodDelete, baseURL+"/api/v1/projects/"+project.ID, me.Org.ID, nil, nil)
	})
	var app struct {
		ID string `json:"id"`
	}
	port := 38000 + time.Now().UnixNano()%1000
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/projects/"+project.ID+"/apps", me.Org.ID, map[string]any{
		"name": "phase8-web", "source_type": "image", "image": "nginx:alpine", "port": port,
		"health_check": map[string]any{"enabled": true, "path": "/", "timeout_ms": 30000, "retries": 10},
	}, &app); err != nil {
		t.Fatal(err)
	}
	var deployment struct {
		ID string `json:"id"`
	}
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/apps/"+app.ID+"/deploy", me.Org.ID, nil, &deployment); err != nil {
		t.Fatal(err)
	}
	if deployment.ID == "" {
		t.Fatal("deployment response did not include an id")
	}

	var state struct {
		Status string `json:"status"`
	}
	deadline := time.Now().Add(8 * time.Minute)
	for time.Now().Before(deadline) {
		if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/apps/"+app.ID+"/deployments/"+deployment.ID, me.Org.ID, nil, &state); err != nil {
			t.Fatal(err)
		}
		if state.Status == "ready" {
			break
		}
		if state.Status == "failed" || state.Status == "cancelled" {
			t.Fatalf("deployment ended in %s", state.Status)
		}
		time.Sleep(2 * time.Second)
	}
	if state.Status != "ready" {
		t.Fatalf("deployment did not become ready: %s", state.Status)
	}

	var runtimeMetrics map[string]any
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/runtime/metrics", me.Org.ID, nil, &runtimeMetrics); err != nil {
		t.Fatal(err)
	}
	if _, ok := runtimeMetrics["queues"]; !ok {
		t.Fatal("runtime metrics did not include queue metrics")
	}
	var events []map[string]any
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/events", me.Org.ID, nil, &events); err != nil {
		t.Fatal(err)
	}
	if !containsEvent(events, deployment.ID) {
		t.Fatal("deployment event was not available through realtime history")
	}

	pool, err := openE2EDatabase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM deployments WHERE id=$1", deployment.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("deployment was not persisted: %d", count)
	}
}

func TestPhase8ProcessHealth(t *testing.T) {
	if os.Getenv("AETHER_E2E") != "1" {
		t.Skip("AETHER_E2E=1 is required")
	}
	client, err := newE2EClient()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"worker", "monitoring"} {
		url := os.Getenv("AETHER_E2E_" + strings.ToUpper(name) + "_HEALTH_URL")
		if url == "" {
			t.Fatalf("AETHER_E2E_%s_HEALTH_URL is required", strings.ToUpper(name))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		var response *http.Response
		request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, url+"/ready", nil)
		if requestErr == nil {
			response, requestErr = client.http.Do(request)
		}
		if requestErr != nil {
			cancel()
			t.Fatal(requestErr)
		}
		response.Body.Close()
		cancel()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s is not ready: %d", name, response.StatusCode)
		}
	}
}

func TestPhase8RestartRecovery(t *testing.T) {
	if os.Getenv("AETHER_E2E") != "1" || os.Getenv("AETHER_E2E_RESTART") != "1" {
		t.Skip("AETHER_E2E=1 and AETHER_E2E_RESTART=1 are required")
	}
	for _, item := range []struct {
		name string
		url  string
		unit string
		path string
	}{
		{name: "worker", url: os.Getenv("AETHER_E2E_WORKER_HEALTH_URL"), unit: envOr("AETHER_E2E_WORKER_CONTAINER", "aether-worker"), path: "/ready"},
		{name: "nats", url: os.Getenv("AETHER_E2E_NATS_HEALTH_URL"), unit: envOr("AETHER_E2E_NATS_CONTAINER", "aether-nats"), path: "/healthz"},
	} {
		if item.url == "" {
			t.Fatalf("AETHER_E2E_%s_HEALTH_URL is required", strings.ToUpper(item.name))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		command := exec.CommandContext(ctx, "podman", "restart", item.unit)
		if output, err := command.CombinedOutput(); err != nil {
			cancel()
			t.Fatalf("restart %s: %v: %s", item.name, err, output)
		}
		deadline := time.Now().Add(60 * time.Second)
		ready := false
		for time.Now().Before(deadline) {
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, item.url+item.path, nil)
			if err == nil {
				response, requestErr := http.DefaultClient.Do(request)
				if requestErr == nil {
					response.Body.Close()
					if response.StatusCode == http.StatusOK {
						ready = true
						break
					}
				}
			}
			time.Sleep(time.Second)
		}
		cancel()
		if !ready {
			t.Fatalf("%s did not become healthy after restart", item.name)
		}
	}
}

type e2eClient struct {
	http *http.Client
}

func newE2EClient() (*e2eClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &e2eClient{http: &http.Client{Jar: jar, Timeout: 30 * time.Second}}, nil
}

func (c *e2eClient) request(ctx context.Context, method, url, orgID string, body any, result any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if orgID != "" {
		req.Header.Set("X-Aether-Org", orgID)
	}
	response, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		raw, _ := io.ReadAll(response.Body)
		return fmt.Errorf("%s %s: status %d: %s", method, url, response.StatusCode, raw)
	}
	if result == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(result)
}

func openE2EDatabase(ctx context.Context) (*pgxpool.Pool, error) {
	port := 5432
	if value := os.Getenv("AETHER_E2E_DATABASE_PORT"); value != "" {
		_, _ = fmt.Sscan(value, &port)
	}
	cfg := database.Config{Host: envOr("AETHER_E2E_DATABASE_HOST", "127.0.0.1"), Port: port, Name: envOr("AETHER_E2E_DATABASE_NAME", "aether"), User: os.Getenv("AETHER_E2E_DATABASE_USER"), Password: os.Getenv("AETHER_E2E_DATABASE_PASSWORD"), SSLMode: "disable", PoolMax: 2, ConnectTimeout: 5}
	if cfg.User == "" {
		return nil, fmt.Errorf("AETHER_E2E_DATABASE_USER is required")
	}
	return database.Open(ctx, cfg)
}

func containsEvent(events []map[string]any, deploymentID string) bool {
	for _, event := range events {
		if event["resource_id"] == deploymentID || event["deployment_id"] == deploymentID {
			return true
		}
	}
	return false
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
