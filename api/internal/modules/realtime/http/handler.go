package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/modules/auth/http"
	"aether/internal/modules/realtime/application"
	"aether/internal/modules/realtime/domain"
	"aether/internal/modules/realtime/infra"
)

type Handler struct {
	realtime *application.Realtime
	hub      *infra.Hub
}

func New(realtime *application.Realtime, hub *infra.Hub) *Handler {
	return &Handler{realtime: realtime, hub: hub}
}

type presenceReq struct {
	Scope string `json:"scope"`
}

func (h *Handler) Join(c *gin.Context) {
	var req presenceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.realtime.Join(c.Request.Context(), strings.TrimSpace(req.Scope), member(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "joined", "scope": req.Scope})
}

func (h *Handler) Heartbeat(c *gin.Context) {
	var req presenceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.realtime.Heartbeat(c.Request.Context(), strings.TrimSpace(req.Scope), member(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) Leave(c *gin.Context) {
	var req presenceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.realtime.Leave(c.Request.Context(), req.Scope, member(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "left"})
}

func (h *Handler) Count(c *gin.Context) {
	scope := c.Query("scope")
	count, members, err := h.realtime.Count(c.Request.Context(), scope)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"scope": scope, "count": count, "members": members})
}

func (h *Handler) RuntimeMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, h.realtime.Metrics(c.Request.Context()))
}

func (h *Handler) Events(c *gin.Context) {
	events, err := h.realtime.RecentEvents(c.Request.Context(), orgID(c), 100)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, events)
}

func (h *Handler) EventsStream(c *gin.Context) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		abort(c, domain.ErrValidation)
		return
	}
	org := orgID(c)
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	events, _ := h.realtime.RecentEvents(c.Request.Context(), org, 20)
	for _, event := range events {
		if data, err := json.Marshal(event); err == nil {
			c.Writer.WriteString("event: history\ndata: " + string(data) + "\n\n")
		}
	}
	flusher.Flush()

	ctx := c.Request.Context()
	sub, err := h.realtime.SubscribeEvents(ctx, org, func(event domain.Event) {
		if data, err := json.Marshal(event); err == nil {
			c.Writer.WriteString("event: notification\ndata: " + string(data) + "\n\n")
			flusher.Flush()
		}
	})
	if err != nil {
		return
	}
	defer sub.Unsubscribe()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			c.Writer.WriteString(": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func (h *Handler) NetworkQuality(c *gin.Context) {
	c.JSON(http.StatusOK, h.realtime.Probe(c.Request.Context(), orgID(c)))
}

func member(c *gin.Context) string {
	return c.MustGet(authhttp.ContextUserID).(uuid.UUID).String()
}

func orgID(c *gin.Context) uuid.UUID {
	return c.MustGet(authhttp.ContextOrgID).(uuid.UUID)
}

func abort(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
	case errors.Is(err, domain.ErrForbidden):
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
