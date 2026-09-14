package http

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"aether/internal/modules/monitoring/domain"
)

type streamReader struct{}

func (streamReader) Latest() *domain.Snapshot {
	return &domain.Snapshot{}
}

func (streamReader) History(string) []domain.HistoryPoint {
	return nil
}

func (streamReader) ResourceHistory(string, string) []domain.ResourcePoint {
	return nil
}

func (streamReader) CollectorStats() domain.CollectorStats {
	return domain.CollectorStats{}
}

func TestStreamUsesEventStreamContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/api/v1/monitoring/stream", nil)
	ctx, cancel := context.WithCancel(request.Context())
	cancel()
	request = request.WithContext(ctx)
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = request

	New(streamReader{}).Stream(ginContext)

	if !strings.HasPrefix(recorder.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("expected event-stream content type, got %q", recorder.Header().Get("Content-Type"))
	}
	if !strings.Contains(recorder.Body.String(), "event: monitoring") {
		t.Fatalf("expected monitoring event, got %q", recorder.Body.String())
	}
}
