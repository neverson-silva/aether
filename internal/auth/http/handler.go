package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"aether/internal/auth/application"
	"aether/internal/auth/domain"
)

type Handler struct {
	auth *application.Auth
}

func New(auth *application.Auth) *Handler {
	return &Handler{auth: auth}
}

const (
	ContextUserID = "auth.user_id"
	ContextOrgID  = "auth.org_id"
	ContextRole   = "auth.role"
	ContextGlobal = "auth.global"
)

type registerReq struct {
	Email    string `json:"email" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginReq struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type memberReq struct {
	Email    string `json:"email" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

type createKeyReq struct {
	Name string `json:"name" binding:"required"`
}

func (h *Handler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	user, token, err := h.auth.Register(c.Request.Context(), req.Email, req.Name, req.Password)
	if err != nil {
		abort(c, err)
		return
	}
	setAuthCookie(c, token)
	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"user":  gin.H{"id": user.ID, "email": user.Email, "name": user.Name, "created_at": user.CreatedAt},
	})
}

func (h *Handler) Login(c *gin.Context) {
	println("DEBUG_LOGIN_ENTER")
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		println("DEBUG_BIND_LOGIN:", err.Error())
		abort(c, domain.ErrValidation)
		return
	}
	user, token, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		abort(c, err)
		return
	}
	setAuthCookie(c, token)
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  gin.H{"id": user.ID, "email": user.Email, "name": user.Name, "created_at": user.CreatedAt},
	})
}

func (h *Handler) Me(c *gin.Context) {
	ctx := c.Request.Context()
	user, orgs, err := h.auth.Me(ctx, userID(c))
	if err != nil {
		abort(c, err)
		return
	}
	type orgView struct {
		ID    uuid.UUID `json:"id"`
		Name  string    `json:"name"`
		Slug  string    `json:"slug"`
		Role  string    `json:"role"`
		Color string    `json:"color"`
	}
	out := make([]orgView, 0, len(orgs))
	for _, o := range orgs {
		color := ""
		if o.Color != nil {
			color = *o.Color
		}
		out = append(out, orgView{ID: o.ID, Name: o.Name, Slug: o.Slug, Role: string(o.Role), Color: color})
	}
	var current *orgView
	if len(out) > 0 {
		current = &out[0]
	}
	c.JSON(http.StatusOK, gin.H{
		"id": user.ID, "email": user.Email, "name": user.Name,
		"global_role":   user.GlobalRole,
		"org":           current,
		"organizations": out,
	})
}

func (h *Handler) ListMembers(c *gin.Context) {
	members, err := h.auth.ListMembers(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(members))
	for _, m := range members {
		out = append(out, gin.H{"user_id": m.UserID, "email": m.Email, "name": m.Name, "role": string(m.Role)})
	}
	c.JSON(http.StatusOK, gin.H{"members": out})
}

func (h *Handler) AddMember(c *gin.Context) {
	var req memberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.auth.AddMember(c.Request.Context(), orgID(c), userID(c), req.Email, req.Name, req.Password, domain.Role(req.Role)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

func (h *Handler) ListKeys(c *gin.Context) {
	keys, err := h.auth.ListAPIKeys(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(keys))
	for _, k := range keys {
		out = append(out, gin.H{
			"id": k.ID, "name": k.Name, "created_at": k.CreatedAt.Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, gin.H{"keys": out})
}

func (h *Handler) CreateKey(c *gin.Context) {
	var req createKeyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	key, raw, err := h.auth.CreateAPIKey(c.Request.Context(), orgID(c), userID(c), req.Name)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": key.ID, "name": key.Name, "key": raw})
}

func (h *Handler) DeleteKey(c *gin.Context) {
	keyID, err := uuid.Parse(c.Param("keyID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.auth.DeleteAPIKey(c.Request.Context(), orgID(c), userID(c), keyID); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ListKeysV1(c *gin.Context) {
	keys, err := h.auth.ListAPIKeys(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(keys))
	for _, k := range keys {
		out = append(out, gin.H{
			"id": k.ID, "name": k.Name, "scopes": []string{}, "created_at": k.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) CreateKeyV1(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if req.Name == "" {
		req.Name = "default"
	}
	_, raw, err := h.auth.CreateAPIKey(c.Request.Context(), orgID(c), userID(c), req.Name)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"key": raw})
}

func (h *Handler) DeleteKeyV1(c *gin.Context) {
	keyID, err := uuid.Parse(c.Param("keyID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.auth.DeleteAPIKey(c.Request.Context(), orgID(c), userID(c), keyID); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) Audit(c *gin.Context) {
	events, err := h.auth.Audit(c.Request.Context(), orgID(c), 50)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(events))
	for _, e := range events {
		out = append(out, gin.H{
			"action": e.Action, "resource_type": e.ResourceType, "resource_id": e.ResourceID,
			"details": e.Details, "created_at": e.CreatedAt.Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, gin.H{"events": out})
}

type totpVerifyReq struct {
	Code string `json:"code"`
}

func (h *Handler) TOTPEnroll(c *gin.Context) {
	me, _, err := h.auth.Me(c.Request.Context(), userID(c))
	if err != nil {
		abort(c, err)
		return
	}
	secret, uri, err := h.auth.EnrollTOTP(c.Request.Context(), userID(c), me.Email)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"secret": secret, "uri": uri})
}

func (h *Handler) TOTPVerify(c *gin.Context) {
	var req totpVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.auth.VerifyTOTP(c.Request.Context(), userID(c), req.Code); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "enabled"})
}

func (h *Handler) TOTPDisable(c *gin.Context) {
	if err := h.auth.DisableTOTP(c.Request.Context(), userID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "disabled"})
}

func (h *Handler) MembersV1(c *gin.Context) {
	members, err := h.auth.ListMembers(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(members))
	for _, m := range members {
		out = append(out, gin.H{"user_id": m.UserID, "email": m.Email, "name": m.Name, "role": m.Role})
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) UpdateMemberV1(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("userID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.auth.UpdateMemberRole(c.Request.Context(), orgID(c), userID(c), targetID, domain.Role(req.Role)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AddMemberV1(c *gin.Context) {
	h.AddMember(c)
}

func (h *Handler) AuthStatus(c *gin.Context) {
	registered, sso := h.auth.PublicStatus(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"registered": registered, "sso": sso})
}

func (h *Handler) Logout(c *gin.Context) {
	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie("aether_token", "", -1, "/", "", true, true)
	c.JSON(http.StatusOK, gin.H{"status": "logged_out"})
}

func (h *Handler) Metrics(c *gin.Context) {
	me, _, err := h.auth.Me(c.Request.Context(), userID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"requests": gin.H{"total": 0},
		"user":     gin.H{"email": me.Email},
	})
}

func userID(c *gin.Context) uuid.UUID {
	return c.MustGet(ContextUserID).(uuid.UUID)
}

func orgID(c *gin.Context) uuid.UUID {
	return c.MustGet(ContextOrgID).(uuid.UUID)
}

func setAuthCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie("aether_token", token, 86400*7, "/", "", true, true)
}

func abort(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	case errors.Is(err, domain.ErrForbidden):
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
	case errors.Is(err, domain.ErrInvalidCredentials):
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
	case errors.Is(err, domain.ErrNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, domain.ErrConflict):
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "already exists"})
	case errors.Is(err, domain.ErrEmailTaken):
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "email already registered"})
	case errors.Is(err, domain.ErrValidation):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
