package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiterAllowsBurst(t *testing.T) {
	rl := NewRateLimiter(1, 3)
	for i := 0; i < 3; i++ {
		if !rl.Allow("ip-a") {
			t.Fatalf("burst de 3 deveria ser permitido na tentativa %d", i+1)
		}
	}
	if rl.Allow("ip-a") {
		t.Fatalf("4ª requisição além do burst deveria ser bloqueada")
	}
}

func TestRateLimiterRefills(t *testing.T) {
	now := time.Unix(0, 0)
	rl := NewRateLimiter(1, 1)
	rl.now = func() time.Time { return now }
	if !rl.Allow("ip-b") {
		t.Fatalf("primeira requisição deveria passar")
	}
	if rl.Allow("ip-b") {
		t.Fatalf("sem tokens deveria bloquear")
	}
	now = now.Add(time.Second)
	if !rl.Allow("ip-b") {
		t.Fatalf("após 1s (rate=1/s) deveria refillar 1 token")
	}
}

func TestRateLimiterIsolatedByKey(t *testing.T) {
	rl := NewRateLimiter(1, 1)
	if !rl.Allow("ip-a") {
		t.Fatalf("ip-a deveria passar")
	}
	if !rl.Allow("ip-b") {
		t.Fatalf("ip-b é independente e deveria ter burst cheio")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(1, 1)
	engine := gin.New()
	engine.POST("/login", RateLimit(rl), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := func() int {
		rec := httptest.NewRecorder()
		httpreq := httptest.NewRequest(http.MethodPost, "/login", nil)
		httpreq.RemoteAddr = "10.0.0.1:1234"
		engine.ServeHTTP(rec, httpreq)
		return rec.Code
	}
	if code := req(); code != http.StatusOK {
		t.Fatalf("primeira requisição deveria ser 200, got %d", code)
	}
	if code := req(); code != http.StatusTooManyRequests {
		t.Fatalf("segunda requisição deveria ser 429, got %d", code)
	}
}
