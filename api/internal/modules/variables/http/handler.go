package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/modules/auth/http"
	"aether/internal/modules/variables/application"
	"aether/internal/modules/variables/domain"
)

type Handler struct {
	variables *application.Variables
}

func New(variables *application.Variables) *Handler {
	return &Handler{variables: variables}
}

type variableReq struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Secret   bool   `json:"is_secret"`
	SecretIm bool   `json:"secret"`
}

type bulkReq struct {
	Values map[string]string `json:"values"`
}

type replaceEntry struct {
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

func (h *Handler) ReplaceVariables(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req map[string]replaceEntry
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	entries := make(map[string]domain.VariableInput, len(req))
	for key, entry := range req {
		entries[key] = domain.VariableInput{Value: entry.Value, Secret: entry.Secret}
	}
	saved, err := h.variables.Replace(c.Request.Context(), projectID, orgID(c), nil, userID(c), entries)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "saved": saved})
}

func (h *Handler) SetVariable(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req variableReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	variable, err := h.variables.Set(c.Request.Context(), projectID, orgID(c), nil, userID(c), req.Key, req.Value, req.Secret || req.SecretIm)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"key": variable.Key, "is_secret": variable.IsSecret})
}

func (h *Handler) ListVariables(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	variables, err := h.variables.List(c.Request.Context(), projectID, orgID(c), nil)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(variables))
	for _, variable := range variables {
		out = append(out, gin.H{
			"key": variable.Key, "value": variable.Value, "is_secret": variable.IsSecret,
		})
	}
	c.JSON(http.StatusOK, gin.H{"variables": out})
}

func (h *Handler) DeleteVariable(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.variables.Delete(c.Request.Context(), projectID, orgID(c), nil, userID(c), c.Param("key")); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) Audit(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	events, err := h.variables.Audit(c.Request.Context(), projectID, orgID(c), 50)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(events))
	for _, event := range events {
		out = append(out, gin.H{
			"action": event.Action, "key": event.Key, "user_id": event.UserID,
			"created_at": event.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"events": out})
}

func (h *Handler) Export(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	variables, err := h.variables.List(c.Request.Context(), projectID, orgID(c), nil)
	if err != nil {
		abort(c, err)
		return
	}
	out := make(map[string]string, len(variables))
	for _, variable := range variables {
		if !variable.IsSecret {
			out[variable.Key] = variable.Value
		}
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) Import(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req bulkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.variables.Import(c.Request.Context(), projectID, orgID(c), nil, userID(c), req.Values); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) SetEnvironmentVariable(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	envID, err := uuid.Parse(c.Param("envID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req variableReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if _, err := h.variables.Set(c.Request.Context(), projectID, orgID(c), &envID, userID(c), req.Key, req.Value, req.Secret || req.SecretIm); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ListEnvironmentVariables(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	envID, err := uuid.Parse(c.Param("envID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	variables, err := h.variables.List(c.Request.Context(), projectID, orgID(c), &envID)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(variables))
	for _, variable := range variables {
		out = append(out, gin.H{
			"key": variable.Key, "value": variable.Value, "is_secret": variable.IsSecret,
		})
	}
	c.JSON(http.StatusOK, gin.H{"variables": out})
}

func (h *Handler) ReplaceEnvironmentVariables(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	envID, err := uuid.Parse(c.Param("envID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req map[string]replaceEntry
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	entries := make(map[string]domain.VariableInput, len(req))
	for key, entry := range req {
		entries[key] = domain.VariableInput{Value: entry.Value, Secret: entry.Secret}
	}
	saved, err := h.variables.Replace(c.Request.Context(), projectID, orgID(c), &envID, userID(c), entries)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "saved": saved})
}

func (h *Handler) DeleteEnvironmentVariable(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	envID, err := uuid.Parse(c.Param("envID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.variables.Delete(c.Request.Context(), projectID, orgID(c), &envID, userID(c), c.Param("key")); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) SetDefaultEnvironment(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	envID, err := uuid.Parse(c.Param("envID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.variables.SetDefaultEnvironment(c.Request.Context(), projectID, orgID(c), envID); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ExportEnvironmentVariables(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	envID, err := uuid.Parse(c.Param("envID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	variables, err := h.variables.List(c.Request.Context(), projectID, orgID(c), &envID)
	if err != nil {
		abort(c, err)
		return
	}
	out := make(map[string]string, len(variables))
	for _, variable := range variables {
		if !variable.IsSecret {
			out[variable.Key] = variable.Value
		}
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) ImportEnvironmentVariables(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	envID, err := uuid.Parse(c.Param("envID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req bulkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.variables.Import(c.Request.Context(), projectID, orgID(c), &envID, userID(c), req.Values); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) EnvironmentAudit(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	envID, err := uuid.Parse(c.Param("envID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	events, err := h.variables.AuditByEnvironment(c.Request.Context(), projectID, orgID(c), envID, 50)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(events))
	for _, event := range events {
		out = append(out, gin.H{
			"action": event.Action, "key": event.Key, "user_id": event.UserID,
			"created_at": event.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"events": out})
}

func userID(c *gin.Context) uuid.UUID {
	return c.MustGet(authhttp.ContextUserID).(uuid.UUID)
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
