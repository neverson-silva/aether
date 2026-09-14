package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"aether/internal/modules/host/application"
)

func TestAppendedLogLines(t *testing.T) {
	previous := []string{"one", "two", "three"}
	current := []string{"two", "three", "four", "five"}

	if got := appendedLogLines(previous, current); !reflect.DeepEqual(got, []string{"four", "five"}) {
		t.Fatalf("unexpected log delta: %v", got)
	}
}

func TestAppendedLogLinesAfterRotation(t *testing.T) {
	previous := []string{"old-one", "old-two"}
	current := []string{"new-one"}

	if got := appendedLogLines(previous, current); !reflect.DeepEqual(got, current) {
		t.Fatalf("unexpected rotated log delta: %v", got)
	}
}

func TestLogsFollowUsesEventStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(logsDir, "aether.log"), []byte("2026-09-13T12:00:00Z info ready\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/host/logs?follow=1", nil)
	ctx, cancel := context.WithCancel(request.Context())
	cancel()
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = request

	New(&application.Host{LogsDir: logsDir}).Logs(ginContext)

	if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("expected SSE content type, got %q", got)
	}
	if !strings.Contains(recorder.Body.String(), "event: log") {
		t.Fatalf("expected log event, got %q", recorder.Body.String())
	}
}
