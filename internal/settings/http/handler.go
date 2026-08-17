package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/auth/http"
	"aether/internal/settings/application"
	"aether/internal/settings/domain"
)

type Handler struct {
	settings *application.Settings
	login    func(ctx context.Context, email, name string) (user any, token string, err error)
}

func New(settings *application.Settings) *Handler {
	return &Handler{settings: settings}
}

func (h *Handler) WithSSOLogin(login func(ctx context.Context, email, name string) (user any, token string, err error)) *Handler {
	h.login = login
	return h
}

type s3Req struct {
	Name      string `json:"name"`
	Endpoint  string `json:"endpoint"`
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

func (h *Handler) GetBranding(c *gin.Context) {
	branding, err := h.settings.GetBranding(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, brandingDTO(branding))
}

func (h *Handler) SaveBranding(c *gin.Context) {
	var req struct {
		Name         string `json:"name"`
		LogoURL      string `json:"logo_url"`
		PrimaryColor string `json:"primary_color"`
		AccentColor  string `json:"accent_color"`
		DarkMode     bool   `json:"dark_mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	branding, err := h.settings.SaveBranding(c.Request.Context(), orgID(c), &domain.Branding{
		Name: req.Name, LogoURL: req.LogoURL, PrimaryColor: req.PrimaryColor,
		AccentColor: req.AccentColor, DarkMode: req.DarkMode,
	})
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, brandingDTO(branding))
}

func (h *Handler) CreateS3(c *gin.Context) {
	var req s3Req
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	dest, err := h.settings.CreateS3(c.Request.Context(), orgID(c), req.Name, req.Endpoint, req.Bucket, req.Region, req.AccessKey, req.SecretKey)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, s3DTO(dest))
}

func (h *Handler) ListS3(c *gin.Context) {
	dests, err := h.settings.ListS3(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(dests))
	for i := range dests {
		out = append(out, s3DTO(&dests[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) DeleteS3(c *gin.Context) {
	id, err := uuid.Parse(c.Param("destID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.settings.DeleteS3(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

type oidcReq struct {
	Name         string `json:"name"`
	Issuer       string `json:"issuer"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Scopes       string `json:"scopes"`
}

func (h *Handler) ListSSO(c *gin.Context) {
	providers, err := h.settings.ListOIDC(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(providers))
	for i := range providers {
		out = append(out, gin.H{
			"id": providers[i].ID, "name": providers[i].Name, "issuer": providers[i].Issuer,
			"client_id": providers[i].ClientID, "scopes": providers[i].Scopes,
			"enabled": providers[i].Enabled, "created_at": providers[i].CreatedAt,
		})
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) CreateSSO(c *gin.Context) {
	var req oidcReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	provider, err := h.settings.CreateOIDC(c.Request.Context(), orgID(c), req.Name, req.Issuer, req.ClientID, req.ClientSecret, req.Scopes)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id": provider.ID, "name": provider.Name, "issuer": provider.Issuer,
		"client_id": provider.ClientID, "scopes": provider.Scopes, "enabled": provider.Enabled,
	})
}

func (h *Handler) DeleteSSO(c *gin.Context) {
	id, err := uuid.Parse(c.Param("providerID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.settings.DeleteOIDC(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) PublicSSO(c *gin.Context) {
	providers, err := h.settings.PublicOIDC(c.Request.Context())
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(providers))
	for i := range providers {
		out = append(out, gin.H{"id": providers[i].ID, "name": providers[i].Name})
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) PublicSSOAuthURL(c *gin.Context) {
	id, err := uuid.Parse(c.Param("providerID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	authURL, err := h.settings.OIDCAuthURL(c.Request.Context(), id)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": authURL})
}

func (h *Handler) AuthURL(c *gin.Context) {
	id, err := uuid.Parse(c.Param("providerID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	authURL, err := h.settings.OIDCAuthURL(c.Request.Context(), id)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": authURL})
}

func (h *Handler) SSOCallback(c *gin.Context) {
	id, err := uuid.Parse(c.Param("providerID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code: " + c.Query("error")})
		return
	}
	oidcUser, err := h.settings.OIDCCallback(c.Request.Context(), id, code)
	if err != nil {
		abort(c, err)
		return
	}
	if h.login == nil {
		abort(c, errors.New("login unavailable"))
		return
	}
	user, token, err := h.login(c.Request.Context(), oidcUser.Email, oidcUser.Name)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user, "org_id": id})
}

func brandingDTO(b *domain.Branding) gin.H {
	return gin.H{
		"org_id": b.OrgID, "name": b.Name, "logo_url": b.LogoURL,
		"primary_color": b.PrimaryColor, "accent_color": b.AccentColor,
		"dark_mode": b.DarkMode, "updated_at": b.UpdatedAt,
	}
}

func s3DTO(d *domain.S3Destination) gin.H {
	return gin.H{
		"id": d.ID, "org_id": d.OrgID, "name": d.Name, "endpoint": d.Endpoint,
		"bucket": d.Bucket, "region": d.Region, "created_at": d.CreatedAt,
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
