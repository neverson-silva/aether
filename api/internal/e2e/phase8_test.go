package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"nhooyr.io/websocket"

	"aether/internal/platform/database"
)

func TestPhase8HTTPDeploymentPipeline(t *testing.T) {
	runPhase8AppLifecycle(t, "phase8-web", "phase8-app-terminal")
}

func TestPhase8CanonicalAPILifecycle(t *testing.T) {
	runPhase8AppLifecycle(t, "phase8-api", "phase8-api-terminal")
}

func runPhase8AppLifecycle(t *testing.T, serviceName, terminalMarker string) {
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
	port := freePort(t)
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/projects/"+project.ID+"/apps", me.Org.ID, map[string]any{
		"name": serviceName, "source_type": "image", "image": "nginx:alpine", "port": port,
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
		Error  string `json:"error"`
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
			t.Fatalf("deployment ended in %s: %s", state.Status, state.Error)
		}
		time.Sleep(2 * time.Second)
	}
	if state.Status != "ready" {
		t.Fatalf("deployment did not become ready: %s", state.Status)
	}
	var services []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Kind string `json:"kind"`
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services?project_id="+project.ID, me.Org.ID, nil, &services); err != nil {
		t.Fatal(err)
	}
	serviceID := ""
	for _, service := range services {
		if service.Name == serviceName && service.Kind == "app" {
			serviceID = service.ID
		}
	}
	if serviceID == "" {
		t.Fatal("canonical app service identity was not listed")
	}
	var canonical struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID, me.Org.ID, nil, &canonical); err != nil {
		t.Fatal(err)
	}
	if canonical.ID != serviceID || canonical.Status != "running" {
		t.Fatalf("unexpected canonical app details: %+v", canonical)
	}
	var canonicalContainers []map[string]any
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID+"/containers", me.Org.ID, nil, &canonicalContainers); err != nil {
		t.Fatal(err)
	}
	if len(canonicalContainers) != 1 {
		t.Fatalf("expected one canonical app container, got %d", len(canonicalContainers))
	}
	if err := client.assertTerminal(ctx, baseURL, serviceID, "", terminalMarker); err != nil {
		t.Fatal(err)
	}
	var canonicalLogs map[string]any
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID+"/logs", me.Org.ID, nil, &canonicalLogs); err != nil {
		t.Fatal(err)
	}
	var canonicalStats map[string]any
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID+"/stats", me.Org.ID, nil, &canonicalStats); err != nil {
		t.Fatal(err)
	}
	var canonicalDeployments []struct {
		ID        string `json:"id"`
		ServiceID string `json:"service_id"`
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID+"/deployments", me.Org.ID, nil, &canonicalDeployments); err != nil {
		t.Fatal(err)
	}
	if len(canonicalDeployments) == 0 {
		t.Fatal("canonical app deployments were empty")
	}
	if canonicalDeployments[0].ServiceID != serviceID {
		t.Fatalf("canonical deployment was not scoped to the service: %+v", canonicalDeployments[0])
	}
	var cronJob struct {
		ID string `json:"id"`
	}
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/services/"+serviceID+"/cron-jobs", me.Org.ID, map[string]string{"name": "phase8-cron", "schedule": "0 2 * * *", "command": "true"}, &cronJob); err != nil {
		t.Fatal(err)
	}
	if cronJob.ID == "" {
		t.Fatal("canonical cron job did not include an id")
	}
	var cronJobs []struct {
		ID string `json:"id"`
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID+"/cron-jobs", me.Org.ID, nil, &cronJobs); err != nil {
		t.Fatal(err)
	}
	if len(cronJobs) != 1 || cronJobs[0].ID != cronJob.ID {
		t.Fatalf("unexpected canonical cron jobs: %+v", cronJobs)
	}
	if err := client.request(ctx, http.MethodDelete, baseURL+"/api/v1/services/"+serviceID+"/cron-jobs/"+cronJob.ID, me.Org.ID, nil, nil); err != nil {
		t.Fatal(err)
	}
	var canonicalDeploymentLog map[string]any
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID+"/deployments/"+canonicalDeployments[0].ID+"/log", me.Org.ID, nil, &canonicalDeploymentLog); err != nil {
		t.Fatal(err)
	}
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/services/"+serviceID+"/stop", me.Org.ID, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID, me.Org.ID, nil, &canonical); err != nil {
		t.Fatal(err)
	}
	if canonical.Status != "stopped" {
		t.Fatalf("canonical app did not stop: %s", canonical.Status)
	}
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/services/"+serviceID+"/start", me.Org.ID, nil, nil); err != nil {
		t.Fatal(err)
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
	if err := assertServiceDeleted(ctx, client, baseURL, serviceID, me.Org.ID); err != nil {
		t.Fatal(err)
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

func TestPhase8CanonicalComposeLifecycle(t *testing.T) {
	if os.Getenv("AETHER_E2E") != "1" || os.Getenv("AETHER_E2E_COMPOSE") != "1" {
		t.Skip("AETHER_E2E=1 and AETHER_E2E_COMPOSE=1 are required")
	}
	baseURL := envOr("AETHER_E2E_API_URL", "http://127.0.0.1:8080")
	client, err := newE2EClient()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	email := fmt.Sprintf("phase8-compose-%d@example.test", time.Now().UnixNano())
	var registered struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/auth/register", "", map[string]string{"email": email, "name": "Phase 8 Compose", "password": "phase8-compose-password"}, &registered); err != nil {
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
	var project struct {
		ID string `json:"id"`
	}
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/projects", me.Org.ID, map[string]string{"name": "Phase 8 Compose", "description": "E2E", "color": ""}, &project); err != nil {
		t.Fatal(err)
	}
	var compose struct {
		ID        string `json:"id"`
		ServiceID string `json:"service_id"`
	}
	content := "services:\n  web:\n    image: nginx:alpine\n  worker:\n    image: busybox:1.36\n    command: [\"sh\", \"-c\", \"sleep 300\"]\n"
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/compose", me.Org.ID, map[string]string{"project_id": project.ID, "name": "phase8-compose", "compose": content}, &compose); err != nil {
		t.Fatal(err)
	}
	if compose.ID == "" || compose.ServiceID == "" {
		t.Fatalf("compose response did not expose canonical service identity: %+v", compose)
	}
	var composeDetails struct {
		ServiceID string `json:"service_id"`
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/compose/"+compose.ID, me.Org.ID, nil, &composeDetails); err != nil {
		t.Fatal(err)
	}
	if composeDetails.ServiceID != compose.ServiceID {
		t.Fatalf("compose detail returned inconsistent service identity: create=%s detail=%s", compose.ServiceID, composeDetails.ServiceID)
	}
	var serviceList []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Kind string `json:"kind"`
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services?project_id="+project.ID, me.Org.ID, nil, &serviceList); err != nil {
		t.Fatal(err)
	}
	var serviceID string
	for _, service := range serviceList {
		if service.Name == "phase8-compose" && service.Kind == "compose" {
			serviceID = service.ID
		}
	}
	if serviceID == "" {
		t.Fatal("compose service identity was not created")
	}
	cleanup := func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cleanupCancel()
		_ = client.request(cleanupCtx, http.MethodPost, baseURL+"/api/v1/services/"+serviceID+"/stop", me.Org.ID, nil, nil)
		_ = client.request(cleanupCtx, http.MethodPost, baseURL+"/api/v1/services/"+serviceID+"/delete", me.Org.ID, nil, nil)
		_ = client.request(cleanupCtx, http.MethodDelete, baseURL+"/api/v1/projects/"+project.ID, me.Org.ID, nil, nil)
	}
	defer cleanup()
	var serviceDetails struct {
		ID string `json:"id"`
	}
	var service struct {
		Status string `json:"status"`
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID, me.Org.ID, nil, &serviceDetails); err != nil {
		t.Fatal(err)
	}
	if serviceDetails.ID != serviceID {
		t.Fatalf("service details returned unexpected id: %s", serviceDetails.ID)
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID, me.Org.ID, nil, &service); err != nil {
		t.Fatal(err)
	}
	if service.Status != "pending" {
		t.Fatalf("expected pending compose service before deployment, got %s", service.Status)
	}
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/services/"+serviceID+"/start", me.Org.ID, nil, nil); err != nil {
		t.Fatal(err)
	}
	var containers []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID+"/containers", me.Org.ID, nil, &containers); err != nil {
			t.Fatal(err)
		}
		if len(containers) >= 2 {
			break
		}
		time.Sleep(time.Second)
	}
	if len(containers) < 2 {
		t.Fatalf("expected at least two compose containers, got %d", len(containers))
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID, me.Org.ID, nil, &service); err != nil {
		t.Fatal(err)
	}
	if service.Status != "running" {
		t.Fatalf("expected running compose service, got %s", service.Status)
	}
	var composeDeployments []struct {
		ID string `json:"id"`
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID+"/deployments", me.Org.ID, nil, &composeDeployments); err != nil {
		t.Fatal(err)
	}
	cancelled := false
	for _, deployment := range composeDeployments {
		if deployment.ID != serviceID {
			continue
		}
		if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/services/"+serviceID+"/deployments/"+deployment.ID+"/cancel", me.Org.ID, nil, nil); err != nil {
			t.Fatal(err)
		}
		if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID, me.Org.ID, nil, &service); err != nil {
			t.Fatal(err)
		}
		if service.Status != "stopped" {
			t.Fatalf("expected stopped compose service after cancellation, got %s", service.Status)
		}
		if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/services/"+serviceID+"/start", me.Org.ID, nil, nil); err != nil {
			t.Fatal(err)
		}
		cancelled = true
		break
	}
	if !cancelled {
		t.Fatal("canonical compose deployment was not exposed for cancellation")
	}
	if cancelled {
		deadline = time.Now().Add(60 * time.Second)
		for time.Now().Before(deadline) {
			if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID, me.Org.ID, nil, &service); err != nil {
				t.Fatal(err)
			}
			if service.Status == "running" {
				break
			}
			time.Sleep(time.Second)
		}
		if service.Status != "running" {
			t.Fatalf("compose service did not restart after cancellation: %s", service.Status)
		}
	}
	var allLogs struct {
		Logs string `json:"logs"`
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID+"/logs", me.Org.ID, nil, &allLogs); err != nil {
		t.Fatal(err)
	}
	var composeStats struct {
		State string `json:"state"`
		Stats struct {
			Memory int64 `json:"mem_bytes"`
		} `json:"stats"`
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID+"/stats", me.Org.ID, nil, &composeStats); err != nil {
		t.Fatal(err)
	}
	worker := ""
	for _, container := range containers {
		if strings.Contains(container.Name, "worker") {
			worker = container.Name
			break
		}
	}
	if worker == "" {
		t.Fatal("worker container was not found")
	}
	var selectedLogs struct {
		Logs string `json:"logs"`
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID+"/logs?container="+url.QueryEscape(worker), me.Org.ID, nil, &selectedLogs); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.CommandContext(ctx, "podman", "stop", worker).CombinedOutput(); err != nil {
		t.Fatalf("stop compose worker: %v: %s", err, output)
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID, me.Org.ID, nil, &service); err != nil {
		t.Fatal(err)
	}
	if service.Status != "degraded" {
		t.Fatalf("expected degraded compose service, got %s", service.Status)
	}
	if output, err := exec.CommandContext(ctx, "podman", "start", worker).CombinedOutput(); err != nil {
		t.Fatalf("start compose worker: %v: %s", err, output)
	}
	deadline = time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID, me.Org.ID, nil, &service); err != nil {
			t.Fatal(err)
		}
		if service.Status == "running" {
			break
		}
		time.Sleep(time.Second)
	}
	if service.Status != "running" {
		t.Fatalf("compose service did not recover: %s", service.Status)
	}
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/services/"+serviceID+"/stop", me.Org.ID, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID, me.Org.ID, nil, &service); err != nil {
		t.Fatal(err)
	}
	if service.Status != "stopped" {
		t.Fatalf("expected stopped compose service after stop, got %s", service.Status)
	}
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/services/"+serviceID+"/start", me.Org.ID, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := client.assertTerminal(ctx, baseURL, serviceID, worker, "phase8-compose-terminal"); err != nil {
		t.Fatal(err)
	}
	if err := assertServiceDeleted(ctx, client, baseURL, serviceID, me.Org.ID); err != nil {
		t.Fatal(err)
	}
}

func TestPhase8CanonicalDatabaseLifecycle(t *testing.T) {
	if os.Getenv("AETHER_E2E") != "1" || os.Getenv("AETHER_E2E_DATABASE") != "1" {
		t.Skip("AETHER_E2E=1 and AETHER_E2E_DATABASE=1 are required")
	}
	baseURL := envOr("AETHER_E2E_API_URL", "http://127.0.0.1:8080")
	client, err := newE2EClient()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	email := fmt.Sprintf("phase8-database-%d@example.test", time.Now().UnixNano())
	var registered struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/auth/register", "", map[string]string{"email": email, "name": "Phase 8 Database", "password": "phase8-database-password"}, &registered); err != nil {
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
	var project struct {
		ID string `json:"id"`
	}
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/projects", me.Org.ID, map[string]string{"name": "Phase 8 Database", "description": "E2E", "color": ""}, &project); err != nil {
		t.Fatal(err)
	}
	var database struct {
		ID      string `json:"id"`
		Engine  string `json:"engine"`
		Version string `json:"version"`
	}
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/databases", me.Org.ID, map[string]any{
		"project_id": project.ID, "name": "phase8-db", "engine": "postgres", "version": "16", "user": "phase8", "password": "phase8-db-password", "mem_mb": 256, "storage_mb": 1024,
	}, &database); err != nil {
		t.Fatal(err)
	}
	if database.ID == "" || database.Engine != "postgres" || database.Version != "16" {
		t.Fatalf("unexpected database specification: %+v", database)
	}
	var services []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Kind string `json:"kind"`
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services?project_id="+project.ID, me.Org.ID, nil, &services); err != nil {
		t.Fatal(err)
	}
	serviceID := ""
	for _, service := range services {
		if service.Name == "phase8-db" && service.Kind == "database" {
			serviceID = service.ID
		}
	}
	if serviceID == "" {
		t.Fatal("database service identity was not created")
	}
	var initialDetails struct {
		Status string `json:"status"`
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID, me.Org.ID, nil, &initialDetails); err != nil {
		t.Fatal(err)
	}
	if initialDetails.Status != "pending" {
		t.Fatalf("expected pending database service before deployment, got %s", initialDetails.Status)
	}
	var backupConfigurations []any
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID+"/backup/configurations", me.Org.ID, nil, &backupConfigurations); err != nil {
		t.Fatal(err)
	}
	var backupHistory []any
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID+"/backups", me.Org.ID, nil, &backupHistory); err != nil {
		t.Fatal(err)
	}
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 45*time.Second)
	t.Cleanup(func() {
		defer cleanupCancel()
		_ = client.request(cleanupCtx, http.MethodPost, baseURL+"/api/v1/services/"+serviceID+"/delete", me.Org.ID, nil, nil)
		_ = client.request(cleanupCtx, http.MethodDelete, baseURL+"/api/v1/projects/"+project.ID, me.Org.ID, nil, nil)
	})
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/services/"+serviceID+"/deploy", me.Org.ID, nil, nil); err != nil {
		t.Fatal(err)
	}
	var details struct {
		ID            string         `json:"id"`
		EnvironmentID string         `json:"environment_id"`
		Status        string         `json:"status"`
		Spec          map[string]any `json:"spec"`
		Runtime       map[string]any `json:"runtime"`
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID, me.Org.ID, nil, &details); err != nil {
		t.Fatal(err)
	}
	if details.ID != serviceID || details.EnvironmentID == "" || details.Status != "running" {
		t.Fatalf("unexpected canonical database details: %+v", details)
	}
	if details.Spec["engine"] != "postgres" || details.Spec["version"] != "16" {
		t.Fatalf("database spec was not exposed canonically: %+v", details.Spec)
	}
	if _, ok := details.Spec["pass_enc"]; ok {
		t.Fatal("database credentials leaked through canonical service details")
	}
	if _, ok := details.Spec["dsn"]; ok {
		t.Fatal("database connection string leaked through canonical service details")
	}
	var connection struct {
		DSN string `json:"dsn"`
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID+"/connection", me.Org.ID, nil, &connection); err != nil {
		t.Fatal(err)
	}
	if connection.DSN == "" {
		t.Fatal("canonical database connection endpoint returned no DSN")
	}
	if _, ok := details.Runtime["containers"]; !ok {
		t.Fatal("database runtime containers were not exposed canonically")
	}
	var containers []map[string]any
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID+"/containers", me.Org.ID, nil, &containers); err != nil {
		t.Fatal(err)
	}
	if len(containers) != 1 {
		t.Fatalf("expected one database container, got %d", len(containers))
	}
	if err := client.assertTerminal(ctx, baseURL, serviceID, "", "phase8-database-terminal"); err != nil {
		t.Fatal(err)
	}
	for _, endpoint := range []string{"logs", "stats", "deployments", "environment"} {
		var response json.RawMessage
		if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID+"/"+endpoint, me.Org.ID, nil, &response); err != nil {
			t.Fatal(err)
		}
	}
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/services/"+serviceID+"/stop", me.Org.ID, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID, me.Org.ID, nil, &details); err != nil {
		t.Fatal(err)
	}
	if details.Status != "stopped" {
		t.Fatalf("canonical database did not stop: %s", details.Status)
	}
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/services/"+serviceID+"/start", me.Org.ID, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := assertServiceDeleted(ctx, client, baseURL, serviceID, me.Org.ID); err != nil {
		t.Fatal(err)
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

func (c *e2eClient) assertTerminal(ctx context.Context, baseURL, serviceID, container, marker string) error {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return err
	}
	cookies := c.http.Jar.Cookies(parsed)
	header := http.Header{}
	if len(cookies) > 0 {
		values := make([]string, 0, len(cookies))
		for _, cookie := range cookies {
			values = append(values, cookie.Name+"="+cookie.Value)
		}
		header.Set("Cookie", strings.Join(values, "; "))
	}
	parsed.Scheme = "ws"
	parsed.Path = "/api/v1/ws/terminal/" + serviceID
	query := parsed.Query()
	if container != "" {
		query.Set("container", container)
	}
	parsed.RawQuery = query.Encode()
	connection, _, err := websocket.Dial(ctx, parsed.String(), &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		return fmt.Errorf("open terminal: %w", err)
	}
	defer connection.Close(websocket.StatusNormalClosure, "test complete")
	if err := connection.Write(ctx, websocket.MessageBinary, []byte("printf '%s\\n' "+marker+"\n")); err != nil {
		return fmt.Errorf("write terminal command: %w", err)
	}
	readCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	for {
		messageType, data, err := connection.Read(readCtx)
		if err != nil {
			return fmt.Errorf("read terminal output: %w", err)
		}
		if messageType == websocket.MessageBinary && strings.Contains(string(data), marker) {
			return nil
		}
	}
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

func assertServiceDeleted(ctx context.Context, client *e2eClient, baseURL, serviceID, orgID string) error {
	if err := client.request(ctx, http.MethodPost, baseURL+"/api/v1/services/"+serviceID+"/delete", orgID, nil, nil); err != nil {
		return err
	}
	containers, err := exec.CommandContext(ctx, "podman", "ps", "-aq", "--filter", "label=aether.service-id="+serviceID).Output()
	if err != nil {
		return fmt.Errorf("inspect deleted service containers: %w", err)
	}
	if strings.TrimSpace(string(containers)) != "" {
		return fmt.Errorf("deleted service %s left runtime containers", serviceID)
	}
	var details map[string]any
	if err := client.request(ctx, http.MethodGet, baseURL+"/api/v1/services/"+serviceID, orgID, nil, &details); err == nil {
		return fmt.Errorf("deleted service %s remained accessible", serviceID)
	}
	return nil
}

func freePort(t *testing.T) int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
