package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"aether/internal/auth/domain"
)

func (h *Handler) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""
		if raw, err := c.Cookie("aether_token"); err == nil {
			token = raw
		}
		if token == "" {
			if header := c.GetHeader("Authorization"); strings.HasPrefix(header, "Bearer ") {
				token = strings.TrimPrefix(header, "Bearer ")
			} else {
				token = c.Query("token")
			}
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		auth, err := h.auth.Tokens.Verify(c.Request.Context(), token)
		if err != nil {
			if errors.Is(err, domain.ErrUnauthorized) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.Set(ContextUserID, auth.Subject)
		c.Set(ContextOrgID, auth.OrgID)
		c.Set(ContextRole, string(auth.Role))
		c.Set(ContextGlobal, auth.Global)
		c.Next()
	}
}
