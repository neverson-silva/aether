package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"aether/internal/modules/monitoring/application"
)

type Handler struct {
	mon application.Reader
}

func New(mon application.Reader) *Handler {
	return &Handler{mon: mon}
}

func (h *Handler) Overview(c *gin.Context) {
	c.JSON(http.StatusOK, h.mon.Latest())
}

func (h *Handler) Resources(c *gin.Context) {
	snap := h.mon.Latest()
	if snap == nil {
		c.JSON(http.StatusOK, gin.H{"resources": []any{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"resources": snap.Resources})
}

func (h *Handler) History(c *gin.Context) {
	window := c.DefaultQuery("window", "1h")
	c.JSON(http.StatusOK, gin.H{"window": window, "points": h.mon.History(window)})
}

func (h *Handler) ResourceHistory(c *gin.Context) {
	id := c.Param("id")
	window := c.DefaultQuery("window", "1h")
	c.JSON(http.StatusOK, gin.H{"resource_id": id, "window": window, "points": h.mon.ResourceHistory(id, window)})
}

func (h *Handler) Collector(c *gin.Context) {
	c.JSON(http.StatusOK, h.mon.CollectorStats())
}

func (h *Handler) Stream(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	writeSnapshot(c, h.mon.Latest())
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			writeSnapshot(c, h.mon.Latest())
		}
	}
}

func writeSnapshot(c *gin.Context, snap interface{}) {
	data, _ := json.Marshal(snap)
	_, _ = fmt.Fprintf(c.Writer, "event: monitoring\ndata: %s\n\n", data)
	c.Writer.Flush()
}
