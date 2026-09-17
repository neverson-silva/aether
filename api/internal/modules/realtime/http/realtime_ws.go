package http

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"nhooyr.io/websocket"

	"aether/internal/modules/realtime/domain"
)

func (h *Handler) RealtimeWS(c *gin.Context) {
	orgID := orgID(c)
	release, err := h.streams.Acquire(orgID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "stream limit reached"})
		return
	}
	defer release()
	userID, err := uuid.Parse(member(c))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	patterns := append([]string(nil), h.originPatterns...)
	requestHost := c.GetHeader("X-Forwarded-Host")
	if requestHost == "" {
		requestHost = c.Request.Host
	}
	if host := requestHostname(requestHost); host != "" && !containsString(patterns, host) {
		patterns = append(patterns, host)
	}
	if requestHost != "" && !containsString(patterns, requestHost) {
		patterns = append(patterns, requestHost)
	}
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{OriginPatterns: patterns})
	if err != nil {
		slog.Warn("realtime websocket handshake failed", "err", err, "origin", c.GetHeader("Origin"), "host", c.Request.Host, "upgrade", c.GetHeader("Upgrade"), "connection", c.GetHeader("Connection"), "patterns", patterns)
		return
	}
	conn.SetReadLimit(64 << 10)
	client := h.hub.Add(conn, orgID, userID)
	if client == nil {
		_ = conn.Close(websocket.StatusPolicyViolation, "connection limit reached")
		return
	}
	h.hub.Run(client, c.Request.Context())
}

func requestHostname(host string) string {
	parsed, err := url.Parse("http://" + host)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
