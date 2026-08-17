package http

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	deploydomain "aether/internal/deployments/domain"
)

func (h *Handler) DeploymentLog(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, deploydomain.ErrValidation)
		return
	}
	depID, err := uuid.Parse(c.Param("depID"))
	if err != nil {
		abort(c, deploydomain.ErrValidation)
		return
	}
	dep, err := h.deployments.Get(c.Request.Context(), depID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	if dep.AppID != appID {
		abort(c, deploydomain.ErrNotFound)
		return
	}
	content, err := os.ReadFile(h.logPath(dep.ID))
	if err != nil {
		content = nil
	}
	c.JSON(http.StatusOK, gin.H{
		"number": dep.Number, "status": dep.Status, "error": dep.Error, "content": string(content),
	})
}

func (h *Handler) LogHistory(c *gin.Context) {
	depID, err := uuid.Parse(c.Param("depID"))
	if err != nil {
		abort(c, deploydomain.ErrValidation)
		return
	}
	if _, err := h.deployments.Get(c.Request.Context(), depID, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	limit := 100
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	lines, err := h.tailLog(depID, limit)
	if err != nil {
		lines = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"lines": lines, "has_more": len(lines) == limit})
}

func (h *Handler) AppLogHistory(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, deploydomain.ErrValidation)
		return
	}
	if _, err := h.apps.GetApp(c.Request.Context(), appID, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	deps, err := h.deployments.List(c.Request.Context(), appID, orgID(c), 1)
	if err != nil || len(deps) == 0 {
		abort(c, deploydomain.ErrNotFound)
		return
	}
	limit := 200
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	lines, err := h.tailLog(deps[0].ID, limit)
	if err != nil {
		lines = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"lines": lines, "has_more": len(lines) == limit})
}

func (h *Handler) Logs(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, deploydomain.ErrValidation)
		return
	}
	if _, err := h.apps.GetApp(c.Request.Context(), appID, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	stream := newSSEWriter(c)
	defer stream.Close()

	deps, err := h.deployments.List(c.Request.Context(), appID, orgID(c), 1)
	if err != nil || len(deps) == 0 || deps[0].ContainerID == "" {
		stream.Ping()
		<-c.Request.Context().Done()
		return
	}
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	reader, writer := io.Pipe()
	go func() {
		_ = h.runtime.FollowLogs(ctx, deps[0].ContainerID, writer)
		_ = writer.Close()
	}()
	scanner := bufio.NewScanner(reader)
	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-heartbeat.C:
			stream.Ping()
		default:
			if scanner.Scan() {
				stream.Line(scanner.Text())
				continue
			}
			if err := scanner.Err(); err != nil {
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

type sseWriter struct {
	c   *gin.Context
	buf *bufio.Writer
}

func newSSEWriter(c *gin.Context) *sseWriter {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()
	return &sseWriter{c: c, buf: bufio.NewWriter(c.Writer)}
}

func (w *sseWriter) Line(line string) {
	_, _ = fmt.Fprintf(w.buf, "data: %s\n\n", line)
	_ = w.buf.Flush()
	w.c.Writer.Flush()
}

func (w *sseWriter) Lines(lines []string) {
	for _, line := range lines {
		if line != "" {
			w.Line(line)
		}
	}
}

func (w *sseWriter) Ping() {
	_, _ = fmt.Fprintf(w.buf, ": ping\n\n")
	_ = w.buf.Flush()
	w.c.Writer.Flush()
}

func (w *sseWriter) Close() {
	_, _ = io.WriteString(w.c.Writer, "event: close\ndata: done\n\n")
	w.c.Writer.Flush()
}

func (h *Handler) logPath(depID uuid.UUID) string {
	return filepath.Join(h.logsDir, "deployments", depID.String()+".log")
}

func (h *Handler) tailLog(depID uuid.UUID, limit int) ([]string, error) {
	file, err := os.Open(h.logPath(depID))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var all []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		all = append(all, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(all) > limit {
		all = all[len(all)-limit:]
	}
	return all, nil
}
