package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/auth/http"
	"aether/internal/webhooks/application"
	"aether/internal/webhooks/domain"
)

type Handler struct {
	webhooks  *application.Webhooks
	providers *application.ProviderHooks
}

func New(webhooks *application.Webhooks, providers *application.ProviderHooks) *Handler {
	return &Handler{webhooks: webhooks, providers: providers}
}

type createReq struct {
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Secret string   `json:"secret"`
	Events []string `json:"events"`
}

func (h *Handler) List(c *gin.Context) {
	hooks, err := h.webhooks.List(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(hooks))
	for i := range hooks {
		out = append(out, webhookDTO(&hooks[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) Create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	hook, err := h.webhooks.Create(c.Request.Context(), orgID(c), req.Name, req.URL, req.Secret, req.Events)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, webhookDTO(hook))
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("webhookID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.webhooks.Delete(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) GitHub(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	status, payload := h.providers.GitHub(c.Request.Context(), appID, c.Request.Body, c.GetHeader("X-Hub-Signature-256"))
	c.JSON(status, payload)
}

func (h *Handler) GitLab(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	status, payload := h.providers.GitLab(c.Request.Context(), appID, c.Request.Body, c.GetHeader("X-Gitlab-Token"))
	c.JSON(status, payload)
}

func (h *Handler) Bitbucket(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	status, payload := h.providers.Bitbucket(c.Request.Context(), appID, c.Request.Body, c.GetHeader("X-Hub-Signature-256"))
	c.JSON(status, payload)
}

func webhookDTO(h *domain.OutWebhook) gin.H {
	return gin.H{
		"id": h.ID, "org_id": h.OrgID, "name": h.Name, "url": h.URL,
		"events": h.Events, "enabled": h.Enabled, "created_at": h.CreatedAt,
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
