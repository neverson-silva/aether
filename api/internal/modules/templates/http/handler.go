package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/modules/auth/http"
	"aether/internal/modules/templates/application"
	"aether/internal/modules/templates/domain"
	"aether/internal/platform/worker"
)

type Handler struct {
	templates          *application.Templates
	compose            *application.Compose
	runtime            worker.Runtime
	deploymentEnqueuer interface {
		EnqueueServiceDeployment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, string) (uuid.UUID, error)
	}
}

func New(templates *application.Templates, compose *application.Compose) *Handler {
	return &Handler{templates: templates, compose: compose}
}

func (h *Handler) WithRuntime(runtime worker.Runtime) *Handler {
	h.runtime = runtime
	return h
}

func (h *Handler) WithDeploymentEnqueuer(enqueuer interface {
	EnqueueServiceDeployment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, string) (uuid.UUID, error)
}) *Handler {
	h.deploymentEnqueuer = enqueuer
	return h
}

type installReq struct {
	ProjectID string            `json:"project_id"`
	Name      string            `json:"name"`
	Overrides map[string]string `json:"overrides"`
}

func (h *Handler) List(c *gin.Context) {
	if c.Query("categories") == "true" {
		filter := domain.Filter{}
		templates, err := h.templates.List(c.Request.Context(), filter)
		if err != nil {
			abort(c, err)
			return
		}
		seen := make(map[string]struct{})
		out := make([]string, 0, len(templates))
		for i := range templates {
			cat := templates[i].Category
			if cat == "" {
				continue
			}
			if _, ok := seen[cat]; ok {
				continue
			}
			seen[cat] = struct{}{}
			out = append(out, cat)
		}
		c.JSON(http.StatusOK, out)
		return
	}
	filter := domain.Filter{
		Category: c.Query("category"),
		Search:   c.Query("q"),
	}
	if c.Query("featured") == "true" {
		filter.Featured = true
	}
	if c.Query("verified") == "true" {
		filter.Verified = true
	}
	if c.Query("editors_choice") == "true" {
		filter.EditorsChoice = true
	}
	templates, err := h.templates.List(c.Request.Context(), filter)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(templates))
	for i := range templates {
		out = append(out, templateDTO(&templates[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) Install(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req installReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	tpl, err := h.templates.Install(c.Request.Context(), templateID, orgID(c), projectID, req.Name, req.Overrides)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, templateDTO(tpl))
}

func (h *Handler) ListCompose(c *gin.Context) {
	apps, err := h.templates.ListCompose(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(apps))
	for i := range apps {
		out = append(out, gin.H{
			"id": apps[i].ID, "service_id": apps[i].ServiceID, "project_id": apps[i].ProjectID, "name": apps[i].Name,
			"status": apps[i].Status, "created_at": apps[i].CreatedAt,
		})
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) DeleteCompose(c *gin.Context) {
	id, err := uuid.Parse(c.Param("composeID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	_ = h.compose.Down(c.Request.Context(), id, orgID(c))
	if err := h.templates.DeleteCompose(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func templateDTO(t *domain.Template) gin.H {
	composeYAML := any(nil)
	if t.ComposeYAML != "" {
		composeYAML = t.ComposeYAML
	}
	return gin.H{
		"id": t.ID, "name": t.Name, "description": t.Description, "category": t.Category,
		"tags": t.Tags, "icon": t.Icon, "version": t.Version, "definition": t.Definition,
		"compose_yaml": composeYAML, "readme": t.Readme, "homepage": t.Homepage,
		"github": t.GitHub, "license": t.License, "installs": t.Installs,
		"featured": t.Featured, "editors_choice": t.EditorsChoice, "verified": t.Verified,
		"updated_at": t.UpdatedAt,
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
