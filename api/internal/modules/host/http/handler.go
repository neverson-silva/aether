package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"aether/internal/modules/host/application"
	"aether/internal/modules/host/domain"
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

func (h *Handler) Info(c *gin.Context) {
	c.JSON(http.StatusOK, h.host.Info())
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
	if c.Query("follow") == "1" {
		h.followLogs(c)
		return
	}
	lines, err := h.host.Logs(100)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"lines": lines})
}

func (h *Handler) followLogs(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	previous := []string{}
	if lines, err := h.host.Logs(100); err == nil {
		writeLogLines(c, lines)
		previous = lines
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			lines, err := h.host.Logs(100)
			if err != nil {
				continue
			}
			writeLogLines(c, appendedLogLines(previous, lines))
			previous = lines
		}
	}
}

func writeLogLines(c *gin.Context, lines []string) {
	for _, line := range lines {
		data, _ := json.Marshal(gin.H{"line": line})
		_, _ = fmt.Fprintf(c.Writer, "event: log\ndata: %s\n\n", data)
	}
	if len(lines) > 0 {
		c.Writer.Flush()
	}
}

func appendedLogLines(previous, current []string) []string {
	if len(previous) == 0 {
		return current
	}
	limit := len(previous)
	if len(current) < limit {
		limit = len(current)
	}
	for overlap := limit; overlap > 0; overlap-- {
		matches := true
		for index := 0; index < overlap; index++ {
			if previous[len(previous)-overlap+index] != current[index] {
				matches = false
				break
			}
		}
		if matches {
			return current[overlap:]
		}
	}
	return current
}

func writeStats(c *gin.Context, stats domain.Stats) {
	data, _ := json.Marshal(stats)
	_, _ = fmt.Fprintf(c.Writer, "event: stats\ndata: %s\n\n", data)
	c.Writer.Flush()
}

func abort(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}
