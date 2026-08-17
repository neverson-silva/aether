package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/auth/http"
	"aether/internal/stats/application"
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
	lines, err := h.stats.DatabaseLogs(c.Request.Context(), dbID, orgID(c), 100)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": joinLines(lines)})
}

func joinLines(lines []string) string {
	out := ""
	for _, line := range lines {
		out += line + "\n"
	}
	return out
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
