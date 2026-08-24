package health

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aether/internal/platform/observability"
)

func TestStatusReadiness(t *testing.T) {
	status := &Status{}
	if status.Ready() {
		t.Fatal("new status should not be ready")
	}
	status.SetReady(true)
	if !status.Ready() {
		t.Fatal("status should be ready")
	}
	status.SetReady(false)
	if status.Ready() {
		t.Fatal("status should not be ready after reset")
	}
}

func TestServerHealthReadinessAndMetrics(t *testing.T) {
	status := &Status{}
	server := NewServer("127.0.0.1:0", status)
	server.SetMetrics(observability.NewMetrics())
	httpServer := httptest.NewServer(server.http.Handler)
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("health status: %d", response.StatusCode)
	}
	response.Body.Close()

	response, err = http.Get(httpServer.URL + "/ready")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("not-ready status: %d", response.StatusCode)
	}
	response.Body.Close()

	status.SetReady(true)
	response, err = http.Get(httpServer.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), "aether_jobs_active") {
		t.Fatalf("metrics response: status=%d body=%q", response.StatusCode, body)
	}
}
