package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/auth/http"
	"aether/internal/gitops/application"
	"aether/internal/gitops/domain"
)

type Handler struct {
	gitops *application.GitOps
}

func New(gitops *application.GitOps) *Handler {
	return &Handler{gitops: gitops}
}

type createReq struct {
	Name      string `json:"name"`
	RepoURL   string `json:"repo_url"`
	Branch    string `json:"branch"`
	Path      string `json:"path"`
	ApplyMode string `json:"apply_mode"`
}

type syncReq struct {
	SHA     string `json:"commit_sha"`
	Added   int    `json:"added"`
	Changed int    `json:"changed"`
	Removed int    `json:"removed"`
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.gitops.List(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(list))
	for i := range list {
		out = append(out, gitopsDTO(&list[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) Create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	g, err := h.gitops.Create(c.Request.Context(), orgID(c), req.Name, req.RepoURL, req.Branch, req.Path, req.ApplyMode)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, gitopsDTO(g))
}

func (h *Handler) Sync(c *gin.Context) {
	id, err := uuid.Parse(c.Param("gitopsID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req syncReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	g, err := h.gitops.Sync(c.Request.Context(), id, orgID(c), req.SHA, req.Added, req.Changed, req.Removed)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gitopsDTO(g))
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("gitopsID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.gitops.Delete(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func gitopsDTO(g *domain.GitOps) gin.H {
	var lastSync string
	if g.LastSync != nil {
		lastSync = g.LastSync.Format(time.RFC3339)
	}
	return gin.H{
		"id": g.ID, "org_id": g.OrgID, "name": g.Name, "repo_url": g.RepoURL,
		"branch": g.Branch, "path": g.Path, "target_org_id": g.TargetOrgID,
		"apply_mode": g.ApplyMode, "last_sha": g.LastSHA, "last_status": g.LastStatus,
		"drift_added": g.DriftAdded, "drift_changed": g.DriftChanged,
		"drift_removed": g.DriftRemoved, "last_sync": lastSync, "created_at": g.CreatedAt,
	}
}

func orgID(c *gin.Context) uuid.UUID {
	return c.MustGet(authhttp.ContextOrgID).(uuid.UUID)
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
