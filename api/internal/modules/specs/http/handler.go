package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appsdomain "aether/internal/modules/apps/domain"
	authhttp "aether/internal/modules/auth/http"
	"aether/internal/modules/specs/application"
	"aether/internal/modules/specs/domain"
)

type Handler struct {
	specs    *application.Specs
	analyzer *application.Analyzer
	apps     AppReader
}

type AppReader interface {
	GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error)
}

func New(specs *application.Specs, analyzer *application.Analyzer, apps AppReader) *Handler {
	return &Handler{specs: specs, analyzer: analyzer, apps: apps}
}

func (h *Handler) Detect(c *gin.Context) {
	var req struct {
		GitURL   string `json:"git_url"`
		Branch   string `json:"git_branch"`
		UploadID string `json:"upload_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if req.GitURL == "" {
		abort(c, domain.ErrValidation)
		return
	}
	res, err := h.analyzer.DetectRepo(c.Request.Context(), req.GitURL, req.Branch)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Analyze(c *gin.Context) {
	var req struct {
		GitURL   string `json:"git_url"`
		Branch   string `json:"git_branch"`
		UploadID string `json:"upload_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if req.GitURL == "" && req.UploadID == "" {
		abort(c, domain.ErrValidation)
		return
	}
	plan, err := h.analyzer.AnalyzeRepo(c.Request.Context(), req.GitURL, req.Branch, req.UploadID)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, plan)
}

func (h *Handler) PlanPreview(c *gin.Context) {
	var req struct {
		Plan *domain.Plan `json:"plan"`
		Port int          `json:"port"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if req.Plan == nil {
		abort(c, domain.ErrValidation)
		return
	}
	preview, err := h.analyzer.PlanPreview(req.Plan, req.Port)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, preview)
}

func (h *Handler) UploadZip(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		abort(c, err)
		return
	}
	upload, err := h.analyzer.SaveZipUpload(header.Filename, data)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, upload)
}

func (h *Handler) AppDetect(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	app, err := h.apps.GetApp(c.Request.Context(), appID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	if app.GitURL == "" {
		abort(c, domain.ErrValidation)
		return
	}
	res, err := h.analyzer.DetectRepo(c.Request.Context(), app.GitURL, app.GitBranch)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AppPlan(c *gin.Context) {
	plan, err := h.planForApp(c)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, plan)
}

func (h *Handler) GetAppPlan(c *gin.Context) {
	plan, err := h.planForApp(c)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, plan)
}

func (h *Handler) planForApp(c *gin.Context) (*domain.Plan, error) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		return nil, domain.ErrValidation
	}
	app, err := h.apps.GetApp(c.Request.Context(), appID, orgID(c))
	if err != nil {
		return nil, err
	}
	if app.GitURL == "" {
		return nil, domain.ErrValidation
	}
	return h.analyzer.AnalyzeRepo(c.Request.Context(), app.GitURL, app.GitBranch, "")
}

func (h *Handler) AppSpec(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	spec, err := h.specs.AppSpec(c.Request.Context(), appID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, spec)
}

func (h *Handler) ExportRuntime(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	spec, err := h.specs.AppSpec(c.Request.Context(), appID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	runtime := c.Query("runtime")
	switch runtime {
	case "", "compose":
		content, _ := h.specs.ExportCompose(spec)
		serveExport(c, "docker-compose.yml", "application/x-yaml", content)
	case "kubernetes", "k8s":
		content, _ := h.specs.ExportKubernetes(spec)
		serveExport(c, spec.Name+".deployment.yaml", "application/x-yaml", content)
	case "nomad":
		content, _ := h.specs.ExportNomad(spec)
		serveExport(c, spec.Name+".nomad.hcl", "text/plain", content)
	default:
		abort(c, domain.ErrValidation)
	}
}

func (h *Handler) Compare(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	depA, err := uuid.Parse(c.Query("a"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	depB, err := uuid.Parse(c.Query("b"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	diff, err := h.specs.Compare(c.Request.Context(), appID, orgID(c), depA, depB)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, diff)
}

func (h *Handler) SystemSummary(c *gin.Context) {
	summary, err := h.specs.SystemSummary(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, summary)
}

func serveExport(c *gin.Context, filename, contentType, content string) {
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.String(http.StatusOK, strings.TrimRight(content, "\n"))
}

func orgID(c *gin.Context) uuid.UUID {
	return c.MustGet(authhttp.ContextOrgID).(uuid.UUID)
}

func abort(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, domain.ErrValidation):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
