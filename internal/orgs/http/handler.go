package http

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/auth/http"
	"aether/internal/orgs/application"
	"aether/internal/orgs/domain"
)

type Handler struct {
	orgs *application.Organizations
}

func New(orgs *application.Organizations) *Handler {
	return &Handler{orgs: orgs}
}

type createReq struct {
	Name string `json:"name"`
}

type updateReq struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Avatar      *string `json:"avatar"`
	Color       *string `json:"color"`
}

type memberReq struct {
	Role string `json:"role"`
}

type assignmentReq struct {
	UserID    string `json:"user_id"`
	ProjectID string `json:"project_id"`
}

func (h *Handler) List(c *gin.Context) {
	orgs, err := h.orgs.List(c.Request.Context(), userID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(orgs))
	for i := range orgs {
		out = append(out, orgDTO(&orgs[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) Create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	org, err := h.orgs.Create(c.Request.Context(), userID(c), req.Name)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, orgDTO(org))
}

func (h *Handler) Get(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("orgID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	org, err := h.orgs.Get(c.Request.Context(), orgID, userID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, orgDTO(org))
}

func (h *Handler) Update(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("orgID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req updateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	org, err := h.orgs.Update(c.Request.Context(), orgID, userID(c), req.Name, req.Description, req.Avatar, req.Color)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, orgDTO(org))
}

func (h *Handler) Delete(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("orgID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.orgs.Delete(c.Request.Context(), orgID, userID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) Members(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("orgID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	members, err := h.orgs.Members(c.Request.Context(), orgID, userID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(members))
	for i := range members {
		out = append(out, gin.H{
			"user_id": members[i].UserID, "email": members[i].Email,
			"name": members[i].Name, "role": members[i].Role,
		})
	}
	c.JSON(http.StatusOK, gin.H{"members": out})
}

func (h *Handler) UpdateMember(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("orgID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	targetID, err := uuid.Parse(c.Param("userID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req memberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.orgs.UpdateMember(c.Request.Context(), orgID, userID(c), targetID, domain.Role(req.Role)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) RemoveMember(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("orgID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	targetID, err := uuid.Parse(c.Param("userID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.orgs.RemoveMember(c.Request.Context(), orgID, userID(c), targetID); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AssignProject(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("orgID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	targetID, err := uuid.Parse(c.Param("userID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.orgs.AssignProject(c.Request.Context(), orgID, userID(c), targetID, projectID); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "assigned"})
}

func (h *Handler) RemoveAssignment(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("orgID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	targetID, err := uuid.Parse(c.Param("userID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.orgs.RemoveAssignment(c.Request.Context(), orgID, userID(c), targetID, projectID); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "removed"})
}

func (h *Handler) Audit(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("orgID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	events, err := h.orgs.Audit(c.Request.Context(), orgID, userID(c), 50)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(events))
	for i := range events {
		out = append(out, gin.H{
			"id": events[i].ID, "action": events[i].Action, "user_id": events[i].UserID,
			"resource": events[i].Resource, "details": events[i].Details, "created_at": events[i].CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"events": out})
}

func (h *Handler) Export(c *gin.Context) {
	data, err := h.orgs.Export(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.Header("Content-Type", "application/yaml")
	c.Header("Content-Disposition", "attachment; filename=aether.yml")
	c.Writer.Write(data)
}

func (h *Handler) Import(c *gin.Context) {
	data, err := io.ReadAll(io.LimitReader(c.Request.Body, 10*1024*1024))
	if err != nil {
		abort(c, err)
		return
	}
	if err := h.orgs.Import(c.Request.Context(), orgID(c), data); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "imported"})
}

func orgDTO(o *domain.Org) gin.H {
	return gin.H{
		"id": o.ID, "name": o.Name, "slug": o.Slug, "avatar": o.Avatar, "color": o.Color,
		"description": o.Description, "owner_user_id": o.OwnerID, "role": o.Role,
		"created_at": o.CreatedAt,
	}
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
