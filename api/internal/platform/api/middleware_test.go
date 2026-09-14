package api

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func testEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RequestID())
	engine.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"request_id": c.GetString(requestIDKey)})
	})
	return engine
}

func TestRequestIDGenerated(t *testing.T) {
	engine := testEngine()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	engine.ServeHTTP(rec, req)
	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatalf("request id ausente no header")
	}
	if !strings.Contains(rec.Body.String(), "request_id") {
		t.Fatalf("request id ausente no corpo")
	}
}

func TestRequestIDPropagated(t *testing.T) {
	engine := testEngine()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Request-ID", "req-abc-123")
	engine.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-ID"); got != "req-abc-123" {
		t.Fatalf("esperava req-abc-123, got %s", got)
	}
}

func TestRequestIDReplacesInvalidID(t *testing.T) {
	engine := testEngine()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ping", nil)
	request.Header.Set("X-Request-ID", "unsafe\r\nX-Injected: true")
	engine.ServeHTTP(recorder, request)

	value := recorder.Header().Get("X-Request-ID")
	if value == "" || !validRequestID(value) || strings.Contains(value, "unsafe") {
		t.Fatalf("unexpected sanitized request id: %q", value)
	}
}

func TestCORSPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(CORS([]string{"https://app.example.com"}))
	engine.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/x", nil)
	req.Header.Set("Origin", "https://app.example.com")
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight deveria retornar 204, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatalf("allow-origin incorreto")
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	engine.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("origin não permitida não deveria ter header CORS")
	}
}

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(SecurityHeaders())
	engine.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("security headers missing")
	}
}

func TestCSRFProtectionRejectsForeignCookieOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(CSRFProtection([]string{"https://app.example.com"}))
	engine.POST("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Host = "api.example.com"
	req.Header.Set("Origin", "https://evil.example.com")
	req.AddCookie(&http.Cookie{Name: "aether_token", Value: "session"})
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected CSRF rejection, got %d", rec.Code)
	}
}

func TestCSRFProtectionRejectsMissingCookieOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(CSRFProtection([]string{"https://app.example.com"}))
	engine.POST("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.AddCookie(&http.Cookie{Name: "aether_token", Value: "session"})
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected missing origin rejection, got %d", rec.Code)
	}
}

func TestCSRFProtectionRejectsMissingOriginForRefreshCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(CSRFProtection([]string{"https://app.example.com"}))
	engine.POST("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.AddCookie(&http.Cookie{Name: "aether_refresh", Value: "refresh"})
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected refresh CSRF rejection, got %d", rec.Code)
	}
}

func TestRequestBodyLimitRejectsLargeBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RequestBodyLimit())
	engine.POST("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(strings.Repeat("x", int(maxRequestBodyBytes)+1)))
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected body limit rejection, got %d", rec.Code)
	}
}

func TestRequestLoggerOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RequestID(), RequestLogger(logger))
	engine.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	engine.ServeHTTP(rec, req)
	if !strings.Contains(buf.String(), `"status":200`) || !strings.Contains(buf.String(), `"path":"/ping"`) {
		t.Fatalf("log estruturado incompleto: %s", buf.String())
	}
	if !strings.Contains(buf.String(), `"request_id"`) {
		t.Fatalf("log sem request_id: %s", buf.String())
	}
}

func TestTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(Timeout(10 * time.Millisecond))
	engine.GET("/slow", func(c *gin.Context) {
		select {
		case <-c.Request.Context().Done():
			c.Status(http.StatusGatewayTimeout)
		case <-time.After(time.Second):
			c.Status(http.StatusOK)
		}
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("esperava timeout 504, got %d", rec.Code)
	}
}

func TestTimeoutSkipsEventStreams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(Timeout(10 * time.Millisecond))
	engine.GET("/stream", func(c *gin.Context) {
		select {
		case <-c.Request.Context().Done():
			c.Status(http.StatusGatewayTimeout)
		default:
			c.Status(http.StatusOK)
		}
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	req.Header.Set("Accept", "text/event-stream")
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected event stream to skip timeout, got %d", rec.Code)
	}
}

func TestReadiness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := &Router{ready: func(ctx context.Context) error {
		return nil
	}}
	engine := gin.New()
	engine.GET("/api/v1/ready", router.handleReady)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil)
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ready ok deveria ser 200, got %d", rec.Code)
	}

	router.ready = func(ctx context.Context) error {
		return context.DeadlineExceeded
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil)
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready falho deveria ser 503, got %d", rec.Code)
	}
}
