package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appsdomain "aether/internal/modules/apps/domain"
	authhttp "aether/internal/modules/auth/http"
	"aether/internal/modules/deployments/application"
	deploydomain "aether/internal/modules/deployments/domain"
	templatesdomain "aether/internal/modules/templates/domain"
)

type Handler struct {
	deployments *application.Deployments
	apps        AppReader
	appOps      *application.AppOps
	logsDir     string
	runtime     LogFollower
	compose     interface {
		Get(context.Context, uuid.UUID, uuid.UUID) (*templatesdomain.ComposeApp, error)
	}
}

type AppReader interface {
	GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error)
}

type LogFollower interface {
	FollowLogs(ctx context.Context, containerID string, writer io.Writer) error
}

func New(deployments *application.Deployments, apps AppReader, appOps *application.AppOps, logsDir string, runtime LogFollower) *Handler {
	return &Handler{deployments: deployments, apps: apps, appOps: appOps, logsDir: logsDir, runtime: runtime}
}

func (h *Handler) WithCompose(reader interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (*templatesdomain.ComposeApp, error)
}) *Handler {
	h.compose = reader
	return h
}

func (h *Handler) Deploy(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, deploydomain.ErrValidation)
		return
	}
	dep, err := h.deployments.Deploy(c.Request.Context(), appID, orgID(c), application.DeployOpts{
		Trigger: "api", TriggeredBy: triggeredBy(c),
	})
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusAccepted, deploymentDTO(dep))
}

func (h *Handler) Rollback(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, deploydomain.ErrValidation)
		return
	}
	dep, err := h.deployments.Rollback(c.Request.Context(), appID, orgID(c), triggeredBy(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusAccepted, deploymentDTO(dep))
}

func (h *Handler) Cancel(c *gin.Context) {
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
	dep, err := h.deployments.Cancel(c.Request.Context(), appID, orgID(c), depID)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusAccepted, deploymentDTO(dep))
}

func (h *Handler) List(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, deploydomain.ErrValidation)
		return
	}
	limit := 25
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if h.compose != nil {
		if compose, composeErr := h.compose.Get(c.Request.Context(), appID, orgID(c)); composeErr == nil {
			status := compose.Status
			if status == "running" {
				c.JSON(http.StatusOK, []gin.H{{
					"id": compose.ID, "number": 1, "status": "ready", "image_ref": "compose",
					"commit": "", "created_at": compose.CreatedAt, "started_at": compose.CreatedAt,
					"finished_at": compose.CreatedAt, "error": "",
				}})
				return
			}
		}
	}
	deps, err := h.deployments.List(c.Request.Context(), appID, orgID(c), limit)
	if err != nil {
		if h.compose != nil {
			if _, composeErr := h.compose.Get(c.Request.Context(), appID, orgID(c)); composeErr == nil {
				c.JSON(http.StatusOK, []gin.H{})
				return
			}
		}
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(deps))
	for i := range deps {
		out = append(out, deploymentDTO(&deps[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) Get(c *gin.Context) {
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
	c.JSON(http.StatusOK, deploymentDTO(dep))
}

func deploymentDTO(d *deploydomain.Deployment) gin.H {
	var started, finished, deploySpec any
	if d.StartedAt != nil {
		started = d.StartedAt
	}
	if d.FinishedAt != nil {
		finished = d.FinishedAt
	}
	if len(d.DeploySpec) > 0 {
		deploySpec = string(d.DeploySpec)
	}
	return gin.H{
		"id": d.ID, "app_id": d.AppID, "service_id": d.ServiceID, "number": d.Number, "status": d.Status,
		"trigger": d.Trigger, "triggered_by": d.TriggeredBy,
		"env_snapshot": string(d.EnvSnapshot),
		"commit":       d.CommitSHA, "image_ref": d.ImageRef, "container_id": d.ContainerID,
		"server_id": d.ServerID, "error": d.Error, "compose_yaml": d.ComposeYAML,
		"deploy_spec": deploySpec, "compose_hash": d.ComposeHash,
		"created_at": d.CreatedAt, "started_at": started, "finished_at": finished,
	}
}

func orgID(c *gin.Context) uuid.UUID {
	return c.MustGet(authhttp.ContextOrgID).(uuid.UUID)
}

func triggeredBy(c *gin.Context) string {
	if v, ok := c.Get(authhttp.ContextUserID); ok {
		return v.(uuid.UUID).String()
	}
	return ""
}

func abort(c *gin.Context, err error) {
	switch {
	case errors.Is(err, deploydomain.ErrNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, deploydomain.ErrConflict):
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "conflict"})
	case errors.Is(err, deploydomain.ErrValidation):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
	case errors.Is(err, deploydomain.ErrForbidden):
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
	case errors.Is(err, deploydomain.ErrInvalidTransition):
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "invalid status transition"})
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
