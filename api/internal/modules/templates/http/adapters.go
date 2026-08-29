package http

import (
	"bufio"
	"bytes"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"aether/internal/modules/templates/domain"
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
	c.JSON(http.StatusOK, gin.H{"env": []gin.H{}})
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
	containerID, err := h.compose.ContainerID(c.Request.Context(), id, orgID(c))
	if err != nil {
		writeComposeSSE(c, nil)
		return
	}
	output, err := exec.CommandContext(c.Request.Context(), "podman", "logs", "--tail", "200", containerID).Output()
	if err != nil {
		writeComposeSSE(c, nil)
		return
	}
	lines := make([]string, 0)
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	writeComposeSSE(c, lines)
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
	output, err := exec.CommandContext(c.Request.Context(), "podman", "stats", "--no-stream", "--format", "{{.CPUPerc}}|{{.MemUsage}}|{{.MemPerc}}", containerID).Output()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"state": "unknown", "stats": nil})
		return
	}
	values := strings.Split(strings.TrimSpace(string(output)), "|")
	cpu := parsePercent(valueAt(values, 0))
	memPercent := parsePercent(valueAt(values, 2))
	memBytes, memLimit := parseMemoryPair(valueAt(values, 1))
	c.JSON(http.StatusOK, gin.H{"state": "running", "stats": gin.H{"cpu_percent": cpu, "mem_bytes": memBytes, "mem_limit": memLimit, "mem_percent": memPercent}})
}

func valueAt(values []string, index int) string {
	if index >= len(values) {
		return ""
	}
	return values[index]
}

func parsePercent(value string) float64 {
	parsed, _ := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "%")), 64)
	return parsed
}

func parseMemoryPair(value string) (int64, int64) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return 0, 0
	}
	return parseMemory(parts[0]), parseMemory(parts[1])
}

func parseMemory(value string) int64 {
	value = strings.TrimSpace(value)
	units := []struct {
		suffix     string
		multiplier float64
	}{{"GiB", 1024 * 1024 * 1024}, {"MiB", 1024 * 1024}, {"KiB", 1024}, {"B", 1}}
	for _, unit := range units {
		if strings.HasSuffix(value, unit.suffix) {
			n, _ := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, unit.suffix)), 64)
			return int64(n * unit.multiplier)
		}
	}
	return 0
}
