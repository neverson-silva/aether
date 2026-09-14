package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireGlobalAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for name, role := range map[string]string{"tenant": "", "org-admin": "admin"} {
		t.Run(name, func(t *testing.T) {
			engine := gin.New()
			engine.Use(func(c *gin.Context) {
				c.Set(ContextGlobal, role)
				c.Next()
			})
			engine.GET("/admin", RequireGlobalAdmin(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, req)
			if role == "admin" && response.Code != http.StatusNoContent {
				t.Fatalf("admin status = %d", response.Code)
			}
			if role != "admin" && response.Code != http.StatusForbidden {
				t.Fatalf("tenant status = %d", response.Code)
			}
		})
	}
}
