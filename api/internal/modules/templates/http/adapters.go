package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"aether/internal/modules/templates/domain"
	"aether/internal/platform/worker"
)

func (h *Handler) ComposeEnv(c *gin.Context) {
	id, err := uuid.Parse(c.Param("composeID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if _, err := h.compose.Get(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	variables, err := h.compose.Environment(c.Request.Context(), id, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	result := make([]gin.H, 0, len(variables))
	for _, variable := range variables {
		result = append(result, gin.H{"key": variable.Key, "value": variable.Value, "is_secret": variable.IsSecret, "environment_id": variable.EnvironmentID})
	}
	c.JSON(http.StatusOK, gin.H{"env": result})
}

func (h *Handler) ComposeTimeline(c *gin.Context) {
	id, err := uuid.Parse(c.Param("composeID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if _, err := h.compose.Get(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	events, err := h.compose.Timeline(c.Request.Context(), id, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, events)
}

func (h *Handler) ComposeLogs(c *gin.Context) {
	id, err := uuid.Parse(c.Param("composeID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	follow := c.Query("follow") == "1" || c.Query("follow") == "true"
	containers, err := h.compose.ContainerIDs(c.Request.Context(), id, orgID(c))
	if err != nil {
		writeComposeSSE(c, []string{})
		return
	}
	if follow {
		writeComposeFollowSSE(c, h.runtime, containers)
		return
	}
	lines := make([]string, 0)
	for _, item := range containers {
		itemLines, logErr := h.runtime.LogTail(c.Request.Context(), item.ID, 200)
		if logErr != nil {
			continue
		}
		lines = append(lines, itemLines...)
	}
	writeComposeSSE(c, lines)
}

func writeComposeFollowSSE(c *gin.Context, runtime worker.LogsRuntime, containers []worker.ContainerInfo) {
	if runtime == nil {
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, item := range containers {
		if item.State != "running" && item.State != "restarting" {
			continue
		}
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()
			writer := &composeSSEWriter{writer: c.Writer, flush: c.Writer.Flush, mu: &mu}
			err := runtime.FollowLogs(c.Request.Context(), containerID, writer)
			mu.Lock()
			defer mu.Unlock()
			if err != nil && !errors.Is(err, context.Canceled) {
				_, _ = c.Writer.WriteString("event: error\ndata: " + fmt.Sprint(err) + "\n\n")
				c.Writer.Flush()
			}
		}(item.ID)
	}
	wg.Wait()
}

type composeSSEWriter struct {
	writer gin.ResponseWriter
	flush  func()
	mu     *sync.Mutex
}

func (w *composeSSEWriter) Write(data []byte) (int, error) {
	lines := strings.Split(string(data), "\n")
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, line := range lines {
		if line == "" {
			continue
		}
		if _, err := w.writer.WriteString("data: " + line + "\n\n"); err != nil {
			return 0, err
		}
	}
	w.flush()
	return len(data), nil
}

func writeComposeSSE(c *gin.Context, lines []string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	for _, line := range lines {
		_, _ = c.Writer.WriteString("data: " + line + "\n\n")
	}
	c.Writer.Flush()
}

func (h *Handler) ComposeStats(c *gin.Context) {
	id, err := uuid.Parse(c.Param("composeID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	containerID, err := h.compose.ContainerID(c.Request.Context(), id, orgID(c))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"state": "stopped", "stats": nil})
		return
	}
	stats, err := h.runtime.Stats(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"state": "unknown", "stats": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"state": "running", "stats": gin.H{"cpu_percent": stats.CPUPercent, "mem_bytes": stats.MemBytes, "mem_limit": stats.MemLimit, "mem_percent": stats.MemPercent}})
}
