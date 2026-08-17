package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/auth/http"
	"aether/internal/pipelines/application"
	"aether/internal/pipelines/domain"
)

type Handler struct {
	pipelines *application.Pipelines
}

func New(pipelines *application.Pipelines) *Handler {
	return &Handler{pipelines: pipelines}
}

type createReq struct {
	AppID   string         `json:"app_id"`
	Name    string         `json:"name"`
	Trigger string         `json:"trigger"`
	Stages  []domain.Stage `json:"stages"`
}

func (h *Handler) List(c *gin.Context) {
	pipelines, err := h.pipelines.List(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(pipelines))
	for i := range pipelines {
		out = append(out, pipelineDTO(&pipelines[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) Create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var appID *uuid.UUID
	if req.AppID != "" {
		id, err := uuid.Parse(req.AppID)
		if err != nil {
			abort(c, domain.ErrValidation)
			return
		}
		appID = &id
	}
	pipeline, err := h.pipelines.Create(c.Request.Context(), orgID(c), appID, req.Name, req.Trigger, req.Stages)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, pipelineDTO(pipeline))
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("pipelineID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.pipelines.Delete(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) Run(c *gin.Context) {
	id, err := uuid.Parse(c.Param("pipelineID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	run, err := h.pipelines.Run(c.Request.Context(), id, orgID(c), "manual")
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, runDTO(run))
}

func (h *Handler) ListRuns(c *gin.Context) {
	id, err := uuid.Parse(c.Param("pipelineID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	runs, err := h.pipelines.ListRuns(c.Request.Context(), id, orgID(c), 30)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(runs))
	for i := range runs {
		out = append(out, runDTO(&runs[i]))
	}
	c.JSON(http.StatusOK, out)
}

func pipelineDTO(p *domain.Pipeline) gin.H {
	return gin.H{
		"id": p.ID, "org_id": p.OrgID, "app_id": p.AppID, "name": p.Name,
		"trigger": p.Trigger, "stages": p.Stages, "enabled": p.Enabled, "created_at": p.CreatedAt,
	}
}

func runDTO(r *domain.Run) gin.H {
	return gin.H{
		"id": r.ID, "pipeline_id": r.PipelineID, "status": r.Status, "trigger": r.Trigger,
		"log": r.Log, "started_at": r.StartedAt, "finished_at": r.FinishedAt,
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
