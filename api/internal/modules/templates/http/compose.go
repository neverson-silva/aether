package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"aether/internal/modules/templates/domain"
)

type composeReq struct {
	ProjectID     string `json:"project_id"`
	EnvironmentID string `json:"environment_id"`
	Name          string `json:"name"`
	Compose       string `json:"compose"`
}

type validateReq struct {
	Content string `json:"content"`
}

func (h *Handler) Create(c *gin.Context) {
	var req composeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Error("compose creation request binding failed", "error", err, "request_id", c.GetString("request_id"))
		abort(c, domain.ErrValidation)
		return
	}
	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		slog.Error("compose creation project id is invalid", "error", err, "project_id", req.ProjectID, "request_id", c.GetString("request_id"))
		abort(c, domain.ErrValidation)
		return
	}
	var environmentID *uuid.UUID
	if req.EnvironmentID != "" {
		id, err := uuid.Parse(req.EnvironmentID)
		if err != nil {
			slog.Error("compose creation environment id is invalid", "error", err, "environment_id", req.EnvironmentID, "request_id", c.GetString("request_id"))
			abort(c, domain.ErrValidation)
			return
		}
		environmentID = &id
	}
	app, err := h.compose.Create(c.Request.Context(), orgID(c), projectID, req.Name, req.Compose, environmentID)
	if err != nil {
		slog.Error("compose creation failed", "error", err, "project_id", projectID, "environment_id", req.EnvironmentID, "name", req.Name, "compose_bytes", len(req.Compose), "request_id", c.GetString("request_id"))
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, composeDTO(app))
}

func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("composeID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	app, err := h.compose.Get(c.Request.Context(), id, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, composeDTO(app))
}

func (h *Handler) Up(c *gin.Context) {
	id, err := uuid.Parse(c.Param("composeID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if h.deploymentEnqueuer == nil {
		abort(c, errors.New("deployment queue is not configured"))
		return
	}
	app, err := h.compose.Get(c.Request.Context(), id, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	deploymentID, err := h.deploymentEnqueuer.EnqueueServiceDeployment(c.Request.Context(), app.ServiceID, app.ID, orgID(c), "compose", "deploy")
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "queued", "deployment_id": deploymentID})
}

func (h *Handler) Down(c *gin.Context) {
	id, err := uuid.Parse(c.Param("composeID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.compose.Down(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

func (h *Handler) Validate(c *gin.Context) {
	var req validateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	c.JSON(http.StatusOK, h.compose.Validate(req.Content))
}

func (h *Handler) AppCompose(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	compose, err := h.compose.AppCompose(c.Request.Context(), appID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"compose": compose})
}

func (h *Handler) DeploymentCompose(c *gin.Context) {
	depID, err := uuid.Parse(c.Param("depID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	compose, err := h.compose.DeploymentCompose(c.Request.Context(), depID)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"compose": compose})
}

func composeDTO(app *domain.ComposeApp) gin.H {
	return gin.H{
		"id": app.ID, "service_id": app.ServiceID, "org_id": app.OrgID, "project_id": app.ProjectID, "environment_id": app.EnvironmentID,
		"name": app.Name, "status": app.Status, "compose": app.Compose, "created_at": app.CreatedAt,
	}
}

func (h *Handler) ImportCompose(c *gin.Context) {
	var req composeReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Compose == "" {
		abort(c, domain.ErrValidation)
		return
	}
	result := h.compose.Validate(req.Compose)
	if !result.Valid {
		abort(c, domain.ErrValidation)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
