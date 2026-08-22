package http

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/modules/auth/http"
	"aether/internal/modules/stats/application"
)

type Handler struct {
	stats *application.Stats
}

func New(stats *application.Stats) *Handler {
	return &Handler{stats: stats}
}

func (h *Handler) AppStats(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, application.ErrValidation)
		return
	}
	info, err := h.stats.AppStats(c.Request.Context(), appID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"state": info.State, "stats": info.Stats})
}

func (h *Handler) DatabaseStats(c *gin.Context) {
	dbID, err := uuid.Parse(c.Param("dbID"))
	if err != nil {
		abort(c, application.ErrValidation)
		return
	}
	info, err := h.stats.DatabaseStats(c.Request.Context(), dbID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"state": info.State, "stats": info.Stats})
}

func (h *Handler) DatabaseLogs(c *gin.Context) {
	dbID, err := uuid.Parse(c.Param("dbID"))
	if err != nil {
		abort(c, application.ErrValidation)
		return
	}
	db, err := h.stats.Database(c.Request.Context(), dbID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	stream := newLogSSEWriter(c)
	defer stream.Close()

	if db.ContainerID == "" {
		stream.Ping()
		<-c.Request.Context().Done()
		return
	}
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	reader, writer := io.Pipe()
	go func() {
		_ = h.stats.FollowLogs(ctx, db.ContainerID, writer)
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
				stream.LogLine(scanner.Text())
				continue
			}
			if err := scanner.Err(); err != nil {
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

type logSSEWriter struct {
	c   *gin.Context
	buf *bufio.Writer
}

func newLogSSEWriter(c *gin.Context) *logSSEWriter {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()
	return &logSSEWriter{c: c, buf: bufio.NewWriter(c.Writer)}
}

func (w *logSSEWriter) LogLine(line string) {
	_, _ = fmt.Fprintf(w.buf, "event: log\ndata: %s\n\n", line)
	_ = w.buf.Flush()
	w.c.Writer.Flush()
}

func (w *logSSEWriter) Ping() {
	_, _ = fmt.Fprintf(w.buf, ": ping\n\n")
	_ = w.buf.Flush()
	w.c.Writer.Flush()
}

func (w *logSSEWriter) Close() {
	_, _ = io.WriteString(w.c.Writer, "event: close\ndata: done\n\n")
	w.c.Writer.Flush()
}

func orgID(c *gin.Context) uuid.UUID {
	return c.MustGet(authhttp.ContextOrgID).(uuid.UUID)
}

func abort(c *gin.Context, err error) {
	switch {
	case errors.Is(err, application.ErrNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "no active container"})
	case errors.Is(err, application.ErrValidation):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
