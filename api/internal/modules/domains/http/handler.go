package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/modules/auth/http"
	"aether/internal/modules/domains/application"
	"aether/internal/modules/domains/domain"
)

type Handler struct {
	domains *application.Domains
}

func New(domains *application.Domains) *Handler {
	return &Handler{domains: domains}
}

type addDomainReq struct {
	Host          string `json:"host"`
	HTTPS         bool   `json:"https"`
	Path          string `json:"path"`
	InternalPath  string `json:"internal_path"`
	StripPath     bool   `json:"strip_path"`
	ContainerPort int    `json:"container_port"`
}

type generateDomainReq struct {
	HTTPS *bool `json:"https"`
}

type createPreviewReq struct {
	Branch string `json:"branch"`
}

func (h *Handler) AddDomain(c *gin.Context) {
	serviceID, serviceType, err := resourceID(c)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req addDomainReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	dom, err := h.domains.Add(c.Request.Context(), serviceID, orgID(c), serviceType, application.AddDomainInput{
		Host: req.Host, HTTPS: req.HTTPS, Path: req.Path, InternalPath: req.InternalPath,
		StripPath: req.StripPath, ContainerPort: req.ContainerPort,
	})
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, domainDTO(dom))
}

func (h *Handler) GenerateFreeDomain(c *gin.Context) {
	serviceID, serviceType, err := resourceID(c)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	https := true
	var req generateDomainReq
	if err := c.ShouldBindJSON(&req); err == nil && req.HTTPS != nil {
		https = *req.HTTPS
	}
	dom, err := h.domains.GenerateFreeDomain(c.Request.Context(), serviceID, orgID(c), serviceType, https)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, domainDTO(dom))
}

func (h *Handler) UpdateDomain(c *gin.Context) {
	serviceID, serviceType, err := resourceID(c)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	domainID, err := uuid.Parse(c.Param("domainID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req addDomainReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if req.Host == "" {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.domains.UpdateDomain(c.Request.Context(), serviceID, orgID(c), serviceType, domainID, application.AddDomainInput{
		Host: req.Host, HTTPS: req.HTTPS, Path: req.Path, InternalPath: req.InternalPath,
		StripPath: req.StripPath, ContainerPort: req.ContainerPort,
	}); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (h *Handler) VerifyDomain(c *gin.Context) {
	serviceID, serviceType, err := resourceID(c)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	domainID, err := uuid.Parse(c.Param("domainID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.domains.Verify(c.Request.Context(), serviceID, orgID(c), serviceType, domainID); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "verifying"})
}

func (h *Handler) ReprovisionDomain(c *gin.Context) {
	serviceID, serviceType, err := resourceID(c)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	domainID, err := uuid.Parse(c.Param("domainID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.domains.Reprovision(c.Request.Context(), serviceID, orgID(c), serviceType, domainID); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "provisioning"})
}

func (h *Handler) GetDomainStatus(c *gin.Context) {
	serviceID, serviceType, err := resourceID(c)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	domainID, err := uuid.Parse(c.Param("domainID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	dom, err := h.domains.GetDomain(c.Request.Context(), serviceID, orgID(c), serviceType, domainID)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, domainDTO(dom))
}

func (h *Handler) ListDomains(c *gin.Context) {
	serviceID, serviceType, err := resourceID(c)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	domains, err := h.domains.List(c.Request.Context(), serviceID, orgID(c), serviceType)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(domains))
	for i := range domains {
		out = append(out, domainDTO(&domains[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) RemoveDomain(c *gin.Context) {
	serviceID, serviceType, err := resourceID(c)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.domains.Remove(c.Request.Context(), serviceID, orgID(c), serviceType, c.Param("host")); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) CreatePreview(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req createPreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	preview, err := h.domains.CreatePreview(c.Request.Context(), appID, orgID(c), req.Branch)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, previewDTO(preview))
}

func (h *Handler) ListPreviews(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	previews, err := h.domains.ListPreviews(c.Request.Context(), appID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(previews))
	for i := range previews {
		out = append(out, previewDTO(&previews[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) DeletePreview(c *gin.Context) {
	id, err := uuid.Parse(c.Param("previewID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.domains.DeletePreview(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) Certificates(c *gin.Context) {
	certs, err := h.domains.Certificates(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(certs))
	for i := range certs {
		out = append(out, gin.H{
			"id": certs[i].DomainID, "app_id": certs[i].AppID, "app_name": certs[i].AppName,
			"host": certs[i].Host, "https": certs[i].HTTPS, "cert_status": certs[i].CertState,
			"created_at": certs[i].CreatedAt,
		})
	}
	c.JSON(http.StatusOK, out)
}

func domainDTO(d *domain.Domain) gin.H {
	return gin.H{
		"id": d.ID, "app_id": d.AppID, "service_type": d.ServiceType, "server_id": d.ServerID, "host": d.Host, "https": d.HTTPS,
		"path": d.Path, "internal_path": d.InternalPath, "strip_path": d.StripPath,
		"container_port": d.ContainerPort, "status": d.Status, "cert_status": d.CertStatus,
		"created_at": d.CreatedAt, "updated_at": d.UpdatedAt,
	}
}

func resourceID(c *gin.Context) (uuid.UUID, string, error) {
	if appID := c.Param("appID"); appID != "" {
		id, err := uuid.Parse(appID)
		return id, application.ServiceTypeApp, err
	}
	id, err := uuid.Parse(c.Param("dbID"))
	return id, application.ServiceTypeDB, err
}

func previewDTO(p *domain.Preview) gin.H {
	return gin.H{
		"id": p.ID, "app_id": p.AppID, "branch": p.Branch, "deployment_id": p.DeploymentID,
		"container_id": p.ContainerID, "domain": p.Domain, "status": p.Status,
		"created_at": p.CreatedAt,
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
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "already exists"})
	case errors.Is(err, domain.ErrValidation):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
	case errors.Is(err, domain.ErrForbidden):
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
