package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"aether/internal/host/application"
	"aether/internal/host/domain"
)

type Handler struct {
	host *application.Host
}

func New(host *application.Host) *Handler {
	return &Handler{host: host}
}

func (h *Handler) Stats(c *gin.Context) {
	c.JSON(http.StatusOK, h.host.Stats(c.Request.Context()))
}

func (h *Handler) StatsStream(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	writeStats(c, h.host.Stats(c.Request.Context()))
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			writeStats(c, h.host.Stats(c.Request.Context()))
		}
	}
}

func (h *Handler) Events(c *gin.Context) {
	events, err := h.host.Events(30)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(events))
	for _, event := range events {
		out = append(out, gin.H{"ts": event.TS, "type": event.Type, "detail": event.Detail})
	}
	c.JSON(http.StatusOK, gin.H{"events": out})
}

func (h *Handler) Logs(c *gin.Context) {
	lines, err := h.host.Logs(100)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"lines": lines})
}

func writeStats(c *gin.Context, stats domain.Stats) {
	data, _ := json.Marshal(stats)
	_, _ = fmt.Fprintf(c.Writer, "event: stats\ndata: %s\n\n", data)
	c.Writer.Flush()
}

func abort(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}
