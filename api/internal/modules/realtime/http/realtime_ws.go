package http

import (
	"net/http"

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
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{OriginPatterns: h.originPatterns})
	if err != nil {
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
