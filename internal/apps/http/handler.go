package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"aether/internal/apps/application"
	"aether/internal/apps/domain"
	authhttp "aether/internal/auth/http"
	variablesApp "aether/internal/variables/application"
)

type Handler struct {
	apps     *application.Apps
	resolver *variablesApp.Resolver
}

func New(apps *application.Apps) *Handler {
	return &Handler{apps: apps}
}

// WithResolver injeta o resolver de variáveis efetivas (usado pelo endpoint
// de effective variables, que mascarar secrets).
func (h *Handler) WithResolver(r *variablesApp.Resolver) *Handler {
	h.resolver = r
	return h
}

type projectReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

type envReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	IsDefault   bool   `json:"is_default"`
}

type appReq struct {
	Name           string       `json:"name"`
	EnvironmentID  string       `json:"environment_id"`
	SourceType     string       `json:"source_type"`
	Image          string       `json:"image"`
	GitURL         string       `json:"git_url"`
	GitBranch      string       `json:"git_branch"`
	UploadID       string       `json:"upload_id"`
	Dockerfile     string       `json:"dockerfile"`
	Port           *int         `json:"port"`
	CPUs           string       `json:"cpus"`
	MemMB          *int         `json:"mem_mb"`
	Resources      resourcesReq `json:"resources"`
	HealthCheck    hcReq        `json:"health_check"`
	BuildType      string       `json:"build_type"`
	PreviewDomain  string       `json:"preview_domain"`
	ImageRetention *int         `json:"image_retention"`
	StorageMB      *int         `json:"storage_mb"`
	InstallCommand string       `json:"install_command"`
	BuildCommand   string       `json:"build_command"`
	StartCommand   string       `json:"start_command"`
	RootFolder     string       `json:"root_folder"`
	DistFolder     string       `json:"dist_folder"`
	WatchPaths     string       `json:"watch_paths"`
}

type resourcesReq struct {
	CPUs  string `json:"cpus"`
	MemMB *int   `json:"mem_mb"`
}

type hcReq struct {
	Enabled    bool   `json:"enabled"`
	Path       string `json:"path"`
	IntervalMS *int   `json:"interval_ms"`
	TimeoutMS  *int   `json:"timeout_ms"`
	Retries    *int   `json:"retries"`
}

type envVarReq struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

type webhookReq struct {
	Secret string `json:"secret"`
}

func (h *Handler) CreateProject(c *gin.Context) {
	var req projectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	project, err := h.apps.CreateProject(c.Request.Context(), orgID(c), req.Name, req.Description, req.Color)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, projectDTO(project))
}

func (h *Handler) GetProject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	project, err := h.apps.GetProject(c.Request.Context(), id, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, projectDTO(project))
}

func (h *Handler) ListProjects(c *gin.Context) {
	projects, err := h.apps.ListProjects(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(projects))
	for _, p := range projects {
		out = append(out, projectDTO(&p))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) UpdateProject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req projectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	project, err := h.apps.UpdateProject(c.Request.Context(), id, orgID(c), req.Name, req.Description, req.Color)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, projectDTO(project))
}

func (h *Handler) DeleteProject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.apps.DeleteProject(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) CreateEnvironment(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req envReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	env, err := h.apps.CreateEnvironment(c.Request.Context(), projectID, req.Name, req.Description, req.Color, req.IsDefault)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, envDTO(env))
}

func (h *Handler) ListEnvironments(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	envs, err := h.apps.ListEnvironments(c.Request.Context(), projectID)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(envs))
	for _, e := range envs {
		out = append(out, envDTO(&e))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) UpdateEnvironment(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	id, err := uuid.Parse(c.Param("envID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req envReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	env, err := h.apps.UpdateEnvironment(c.Request.Context(), id, projectID, req.Name, req.Description, req.Color)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, envDTO(env))
}

func (h *Handler) DeleteEnvironment(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	id, err := uuid.Parse(c.Param("envID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.apps.DeleteEnvironment(c.Request.Context(), id, projectID); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) CreateApp(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	app, err := appFromReq(&appReq{}, c, projectID)
	if err != nil {
		abort(c, err)
		return
	}
	created, err := h.apps.CreateApp(c.Request.Context(), orgID(c), projectID, app)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, appDTO(created))
}

func (h *Handler) GetApp(c *gin.Context) {
	id, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	app, err := h.apps.GetApp(c.Request.Context(), id, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	vars, err := h.apps.ListEnv(c.Request.Context(), id, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	env := make([]gin.H, 0, len(vars))
	for _, v := range vars {
		env = append(env, gin.H{"name": v.Name, "value": v.Value, "secret": v.Secret})
	}
	c.JSON(http.StatusOK, gin.H{"app": appDTO(app), "env": env})
}

func (h *Handler) ListApps(c *gin.Context) {
	var projectID *uuid.UUID
	if raw := c.Query("project_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			abort(c, domain.ErrValidation)
			return
		}
		projectID = &id
	}
	apps, err := h.apps.ListApps(c.Request.Context(), orgID(c), projectID)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(apps))
	if h.apps.LatestDeployments != nil {
		ids := make([]uuid.UUID, 0, len(apps))
		for i := range apps {
			ids = append(ids, apps[i].ID)
		}
		latest, lerr := h.apps.LatestDeployments(c.Request.Context(), ids)
		if lerr != nil {
			latest = nil
		}
		for i := range apps {
			dto := appDTO(&apps[i])
			if st, ok := latest[apps[i].ID]; ok {
				dto["latest_deployment"] = gin.H{"status": st}
			}
			out = append(out, dto)
		}
	} else {
		for i := range apps {
			out = append(out, appDTO(&apps[i]))
		}
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) UpdateApp(c *gin.Context) {
	id, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	app, err := appFromReq(&appReq{}, c, uuid.Nil)
	if err != nil {
		abort(c, err)
		return
	}
	updated, err := h.apps.UpdateApp(c.Request.Context(), id, orgID(c), app)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, appDTO(updated))
}

func (h *Handler) DeleteApp(c *gin.Context) {
	id, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.apps.DeleteApp(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) SetEnv(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req envVarReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.apps.SetEnv(c.Request.Context(), appID, orgID(c), req.Name, req.Value, req.Secret); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ListEnv(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	vars, err := h.apps.ListEnv(c.Request.Context(), appID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(vars))
	for _, v := range vars {
		out = append(out, gin.H{"name": v.Name, "value": v.Value, "secret": v.Secret})
	}
	c.JSON(http.StatusOK, gin.H{"vars": out})
}

func (h *Handler) EffectiveVariables(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if _, err := h.apps.GetApp(c.Request.Context(), appID, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	if h.resolver == nil {
		abort(c, domain.ErrNotFound)
		return
	}
	resolved, err := h.resolver.Resolved(c.Request.Context(), appID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(resolved))
	for _, v := range resolved {
		value := v.Value
		if v.Secret && value != "" {
			value = "******"
		}
		out = append(out, gin.H{"key": v.Key, "value": value, "source": v.Source, "secret": v.Secret})
	}
	c.JSON(http.StatusOK, gin.H{"variables": out})
}

func (h *Handler) DeleteEnv(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	name := c.Param("name")
	if err := h.apps.DeleteEnv(c.Request.Context(), appID, orgID(c), name); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) SetWebhook(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req webhookReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.apps.SetWebhook(c.Request.Context(), appID, orgID(c), req.Secret); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func appFromReq(req *appReq, c *gin.Context, projectID uuid.UUID) (*domain.App, error) {
	if err := c.ShouldBindJSON(req); err != nil {
		return nil, domain.ErrValidation
	}
	app := &domain.App{
		Name: req.Name, SourceType: req.SourceType, Image: req.Image, GitURL: req.GitURL,
		GitBranch: req.GitBranch, UploadID: req.UploadID, Dockerfile: req.Dockerfile, CPUs: req.CPUs,
		BuildType: req.BuildType, PreviewDomain: req.PreviewDomain,
		InstallCommand: req.InstallCommand, BuildCommand: req.BuildCommand,
		StartCommand: req.StartCommand, RootFolder: req.RootFolder, DistFolder: req.DistFolder,
		WatchPaths:  req.WatchPaths,
		HealthCheck: domain.HealthCheck{Enabled: req.HealthCheck.Enabled, Path: req.HealthCheck.Path},
	}
	if req.Port != nil {
		app.Port = *req.Port
	}
	if req.MemMB != nil {
		app.MemMB = *req.MemMB
	}
	if req.Resources.CPUs != "" {
		app.CPUs = req.Resources.CPUs
	}
	if req.Resources.MemMB != nil {
		app.MemMB = *req.Resources.MemMB
	}
	if req.ImageRetention != nil {
		app.ImageRetention = *req.ImageRetention
	}
	if req.StorageMB != nil {
		app.StorageMB = *req.StorageMB
	}
	if req.HealthCheck.IntervalMS != nil {
		app.HealthCheck.IntervalMS = *req.HealthCheck.IntervalMS
	}
	if req.HealthCheck.TimeoutMS != nil {
		app.HealthCheck.TimeoutMS = *req.HealthCheck.TimeoutMS
	}
	if req.HealthCheck.Retries != nil {
		app.HealthCheck.Retries = *req.HealthCheck.Retries
	}
	if req.EnvironmentID != "" {
		id, err := uuid.Parse(req.EnvironmentID)
		if err != nil {
			return nil, domain.ErrValidation
		}
		app.EnvironmentID = &id
	}
	if projectID != uuid.Nil {
		app.ProjectID = projectID
	}
	return app, nil
}

func projectDTO(p *domain.Project) gin.H {
	return gin.H{
		"id": p.ID, "org_id": p.OrgID, "name": p.Name, "slug": p.Slug,
		"description": p.Description, "color": p.Color,
		"created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
	}
}

func envDTO(e *domain.Environment) gin.H {
	return gin.H{
		"id": e.ID, "project_id": e.ProjectID, "name": e.Name, "slug": e.Slug,
		"description": e.Description, "color": e.Color, "is_default": e.IsDefault,
		"created_at": e.CreatedAt, "updated_at": e.UpdatedAt,
	}
}

func appDTO(a *domain.App) gin.H {
	return gin.H{
		"id": a.ID, "org_id": a.OrgID, "project_id": a.ProjectID, "environment_id": a.EnvironmentID,
		"name": a.Name, "source_type": a.SourceType, "image": a.Image, "git_url": a.GitURL,
		"git_branch": a.GitBranch, "upload_id": a.UploadID, "dockerfile": a.Dockerfile, "port": a.Port, "cpus": a.CPUs,
		"mem_mb": a.MemMB, "build_type": a.BuildType, "preview_domain": a.PreviewDomain,
		"image_retention": a.ImageRetention,
		"storage_mb":      a.StorageMB, "install_command": a.InstallCommand, "build_command": a.BuildCommand,
		"start_command": a.StartCommand, "root_folder": a.RootFolder, "dist_folder": a.DistFolder,
		"watch_paths": a.WatchPaths, "created_at": a.CreatedAt, "updated_at": a.UpdatedAt,
		"resources": gin.H{
			"cpus": a.CPUs, "mem_mb": a.MemMB,
		},
		"server_id": "", "cluster_id": "", "volumes": []gin.H{},
		"health_check": gin.H{
			"enabled": a.HealthCheck.Enabled, "path": a.HealthCheck.Path,
			"interval_ms": a.HealthCheck.IntervalMS, "timeout_ms": a.HealthCheck.TimeoutMS,
			"retries": a.HealthCheck.Retries,
		},
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
