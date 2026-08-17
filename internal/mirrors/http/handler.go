package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"aether/internal/mirrors/application"
	"aether/internal/mirrors/domain"
)

type Handler struct {
	mirrors *application.Mirrors
}

func New(mirrors *application.Mirrors) *Handler {
	return &Handler{mirrors: mirrors}
}

type createReq struct {
	Name          string `json:"name"`
	Source        string `json:"source"`
	Dest          string `json:"dest"`
	DestTLSVerify bool   `json:"dest_tls_verify"`
	TagsFilter    string `json:"tags_filter"`
	Schedule      string `json:"schedule"`
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.mirrors.List(c.Request.Context())
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(list))
	for i := range list {
		out = append(out, mirrorDTO(&list[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) Create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	mirror, err := h.mirrors.Create(c.Request.Context(), req.Name, req.Source, req.Dest, req.DestTLSVerify, req.TagsFilter, req.Schedule)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, mirrorDTO(mirror))
}

func (h *Handler) Run(c *gin.Context) {
	id, err := uuid.Parse(c.Param("mirrorID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.mirrors.Run(c.Request.Context(), id); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "started"})
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("mirrorID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.mirrors.Delete(c.Request.Context(), id); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func mirrorDTO(m *domain.Mirror) gin.H {
	return gin.H{
		"id": m.ID, "name": m.Name, "source": m.Source, "dest": m.Dest,
		"dest_tls_verify": m.DestTLSVerify, "tags_filter": m.TagsFilter, "schedule": m.Schedule,
		"last_run": m.LastRun, "status": m.Status, "created_at": m.CreatedAt,
	}
}

func abort(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, domain.ErrConflict):
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "conflict"})
	case errors.Is(err, domain.ErrValidation):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
	case errors.Is(err, domain.ErrForbidden):
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
