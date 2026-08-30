package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"

	appsapplication "aether/internal/modules/apps/application"
	authhttp "aether/internal/modules/auth/http"
	databasesdomain "aether/internal/modules/databases/domain"
	deployapplication "aether/internal/modules/deployments/application"
	deploydomain "aether/internal/modules/deployments/domain"
	domainsapplication "aether/internal/modules/domains/application"
	domainsdomain "aether/internal/modules/domains/domain"
	realtimedomain "aether/internal/modules/realtime/domain"
	servicedomain "aether/internal/modules/services/domain"
	templatesapplication "aether/internal/modules/templates/application"
	"aether/internal/platform/druntime/events"
	"aether/internal/platform/druntime/queue"
	"aether/internal/platform/worker"
)

type Handler struct {
	db        *pgxpool.Pool
	appDeploy interface {
		Deploy(context.Context, uuid.UUID, uuid.UUID, deployapplication.DeployOpts) (*deploydomain.Deployment, error)
		Cancel(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*deploydomain.Deployment, error)
	}
	appOps interface {
		Start(context.Context, uuid.UUID, uuid.UUID) (string, error)
		Stop(context.Context, uuid.UUID, uuid.UUID) (string, error)
		Restart(context.Context, uuid.UUID, uuid.UUID) (string, error)
		Delete(context.Context, uuid.UUID, uuid.UUID) error
		Timeline(context.Context, uuid.UUID, uuid.UUID) ([]deployapplication.TimelineEntry, error)
	}
	appServiceOps interface {
		StartService(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (string, error)
		StopService(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (string, error)
		RestartService(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (string, error)
		DeleteService(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
	}
	appWebhook interface {
		SetWebhook(context.Context, uuid.UUID, uuid.UUID, string) error
	}
	compose interface {
		Up(context.Context, uuid.UUID, uuid.UUID) error
		Start(context.Context, uuid.UUID, uuid.UUID) error
		Stop(context.Context, uuid.UUID, uuid.UUID) error
		Restart(context.Context, uuid.UUID, uuid.UUID) error
		Down(context.Context, uuid.UUID, uuid.UUID) error
		Delete(context.Context, uuid.UUID, uuid.UUID) error
		Timeline(context.Context, uuid.UUID, uuid.UUID) ([]realtimedomain.Event, error)
	}
	database interface {
		Deploy(context.Context, uuid.UUID, uuid.UUID) (*databasesdomain.Database, error)
		Start(context.Context, uuid.UUID, uuid.UUID) (*databasesdomain.Database, error)
		Stop(context.Context, uuid.UUID, uuid.UUID) (*databasesdomain.Database, error)
		Delete(context.Context, uuid.UUID, uuid.UUID) error
	}
	databaseConnection interface {
		ConnectionStringByServiceID(context.Context, uuid.UUID, uuid.UUID) (string, error)
	}
	domains interface {
		Add(context.Context, uuid.UUID, uuid.UUID, string, domainsapplication.AddDomainInput) (*domainsdomain.Domain, error)
		Remove(context.Context, uuid.UUID, uuid.UUID, string, string) error
		GenerateFreeDomain(context.Context, uuid.UUID, uuid.UUID, string, bool) (*domainsdomain.Domain, error)
	}
	environment interface {
		SetEnv(context.Context, uuid.UUID, uuid.UUID, string, string, bool) error
		DeleteEnv(context.Context, uuid.UUID, uuid.UUID, string) error
	}
	logsDir         string
	runtime         worker.Runtime
	deploymentQueue queue.Queue
	notifier        interface {
		NotifyDeploy(context.Context, deploydomain.DeployEvent)
	}
	deploymentCanceller interface {
		CancelDeployment(uuid.UUID) bool
	}
}

func (h *Handler) WithNotifier(notifier interface {
	NotifyDeploy(context.Context, deploydomain.DeployEvent)
}) *Handler {
	h.notifier = notifier
	return h
}

type composeRuntime struct {
	up interface {
		Up(context.Context, uuid.UUID, uuid.UUID) error
		Start(context.Context, uuid.UUID, uuid.UUID) error
		Stop(context.Context, uuid.UUID, uuid.UUID) error
		Restart(context.Context, uuid.UUID, uuid.UUID) error
		Down(context.Context, uuid.UUID, uuid.UUID) error
	}
	delete interface {
		Delete(context.Context, uuid.UUID, uuid.UUID) error
	}
	timeline interface {
		Timeline(context.Context, uuid.UUID, uuid.UUID) ([]realtimedomain.Event, error)
	}
}

func (r composeRuntime) Up(ctx context.Context, id, orgID uuid.UUID) error {
	return r.up.Up(ctx, id, orgID)
}
func (r composeRuntime) Start(ctx context.Context, id, orgID uuid.UUID) error {
	return r.up.Start(ctx, id, orgID)
}
func (r composeRuntime) Stop(ctx context.Context, id, orgID uuid.UUID) error {
	return r.up.Stop(ctx, id, orgID)
}
func (r composeRuntime) Restart(ctx context.Context, id, orgID uuid.UUID) error {
	return r.up.Restart(ctx, id, orgID)
}
func (r composeRuntime) Down(ctx context.Context, id, orgID uuid.UUID) error {
	return r.up.Down(ctx, id, orgID)
}
func (r composeRuntime) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	return r.delete.Delete(ctx, id, orgID)
}
func (r composeRuntime) Timeline(ctx context.Context, id, orgID uuid.UUID) ([]realtimedomain.Event, error) {
	return r.timeline.Timeline(ctx, id, orgID)
}

func New(db *pgxpool.Pool) *Handler {
	return &Handler{db: db}
}

func (h *Handler) WithRuntime(runtime worker.Runtime) *Handler {
	h.runtime = runtime
	return h
}

func (h *Handler) WithDeploymentQueue(deploymentQueue queue.Queue) *Handler {
	h.deploymentQueue = deploymentQueue
	return h
}

func (h *Handler) WithDeploymentCanceller(canceller interface {
	CancelDeployment(uuid.UUID) bool
}) *Handler {
	h.deploymentCanceller = canceller
	return h
}

func (h *Handler) WithRuntimes(appDeploy *deployapplication.Deployments, appOps *deployapplication.AppOps, compose *templatesapplication.Compose, composeDelete interface {
	Delete(context.Context, uuid.UUID, uuid.UUID) error
}, database interface {
	Deploy(context.Context, uuid.UUID, uuid.UUID) (*databasesdomain.Database, error)
	Start(context.Context, uuid.UUID, uuid.UUID) (*databasesdomain.Database, error)
	Stop(context.Context, uuid.UUID, uuid.UUID) (*databasesdomain.Database, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
}) *Handler {
	h.appDeploy = appDeploy
	h.appOps = appOps
	if serviceOps, ok := any(appOps).(interface {
		StartService(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (string, error)
		StopService(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (string, error)
		RestartService(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (string, error)
		DeleteService(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
	}); ok {
		h.appServiceOps = serviceOps
	}
	h.compose = composeRuntime{up: compose, delete: composeDelete, timeline: compose}
	h.database = database
	if connection, ok := database.(interface {
		ConnectionStringByServiceID(context.Context, uuid.UUID, uuid.UUID) (string, error)
	}); ok {
		h.databaseConnection = connection
	}
	return h
}

func (h *Handler) WithDomains(domains interface {
	Add(context.Context, uuid.UUID, uuid.UUID, string, domainsapplication.AddDomainInput) (*domainsdomain.Domain, error)
	Remove(context.Context, uuid.UUID, uuid.UUID, string, string) error
	GenerateFreeDomain(context.Context, uuid.UUID, uuid.UUID, string, bool) (*domainsdomain.Domain, error)
}) *Handler {
	h.domains = domains
	return h
}

func (h *Handler) WithEnvironment(environment *appsapplication.Apps) *Handler {
	h.environment = environment
	return h
}

func (h *Handler) WithDeploymentLogs(logsDir string) *Handler {
	h.logsDir = logsDir
	return h
}

func (h *Handler) WithAppWebhook(webhook interface {
	SetWebhook(context.Context, uuid.UUID, uuid.UUID, string) error
}) *Handler {
	h.appWebhook = webhook
	return h
}

func (h *Handler) List(c *gin.Context) {
	query := `SELECT id, org_id, project_id, environment_id, name, kind,
CASE WHEN status = 'unknown' AND NOT EXISTS (SELECT 1 FROM deployments WHERE deployments.service_id = services.id) THEN 'pending' ELSE status END, created_at, updated_at,
CASE kind WHEN 'app' THEN (SELECT id FROM apps WHERE service_id = services.id)
WHEN 'compose' THEN (SELECT id FROM compose_apps WHERE service_id = services.id)
WHEN 'database' THEN (SELECT id FROM databases WHERE service_id = services.id) END
FROM services
WHERE org_id = $1 AND deleted_at IS NULL
  AND ((kind = 'app' AND EXISTS (SELECT 1 FROM apps WHERE apps.service_id = services.id))
    OR (kind = 'compose' AND EXISTS (SELECT 1 FROM compose_apps WHERE compose_apps.service_id = services.id))
    OR (kind = 'database' AND EXISTS (SELECT 1 FROM databases WHERE databases.service_id = services.id)))
  AND ($2::uuid IS NULL OR project_id = $2)
  AND ($3::uuid IS NULL OR environment_id = $3)
ORDER BY created_at, name`
	var projectID, environmentID *uuid.UUID
	if value := c.Query("project_id"); value != "" {
		parsed, err := uuid.Parse(value)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
			return
		}
		projectID = &parsed
	}
	if value := c.Query("environment_id"); value != "" {
		parsed, err := uuid.Parse(value)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid environment_id"})
			return
		}
		environmentID = &parsed
	}
	rows, err := h.db.Query(c.Request.Context(), query, orgID(c), projectID, environmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	defer rows.Close()
	services := make([]gin.H, 0)
	for rows.Next() {
		service, err := scanService(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		if specID, ok := service["spec_id"].(*uuid.UUID); ok && specID != nil {
			if serviceStatus, statusOK := service["status"].(string); statusOK {
				serviceID := service["id"].(uuid.UUID)
				states, statesErr := runtimeContainerStates(c, h.runtime, serviceID, *specID)
				if statesErr == nil && len(states) > 0 {
					containers := make([]gin.H, 0, len(states))
					for _, state := range states {
						containers = append(containers, gin.H{"id": state.ID, "name": state.Name, "status": state.Status, "healthy": state.Healthy})
					}
					service["runtime"] = gin.H{"containers": containers}
					service["status"] = string(servicedomain.ProjectStatus(servicedomain.Kind(service["kind"].(string)), states, serviceStatus == string(servicedomain.StatusDeploying), true))
				} else {
					service["status"] = serviceStatus
				}
			}
			h.enrichSpec(c, service, servicedomain.Kind(service["kind"].(string)), *specID)
		}
		services = append(services, service)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, services)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	row := h.db.QueryRow(c.Request.Context(), `
SELECT id, org_id, project_id, environment_id, name, kind,
CASE WHEN status = 'unknown' AND NOT EXISTS (SELECT 1 FROM deployments WHERE deployments.service_id = services.id) THEN 'pending' ELSE status END, created_at, updated_at,
CASE kind WHEN 'app' THEN (SELECT id FROM apps WHERE service_id = services.id)
WHEN 'compose' THEN (SELECT id FROM compose_apps WHERE service_id = services.id)
WHEN 'database' THEN (SELECT id FROM databases WHERE service_id = services.id) END
FROM services
WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL
  AND ((kind = 'app' AND EXISTS (SELECT 1 FROM apps WHERE apps.service_id = services.id))
    OR (kind = 'compose' AND EXISTS (SELECT 1 FROM compose_apps WHERE compose_apps.service_id = services.id))
    OR (kind = 'database' AND EXISTS (SELECT 1 FROM databases WHERE databases.service_id = services.id)))`, id, orgID(c))
	service, err := scanService(row)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	if specID, ok := service["spec_id"].(*uuid.UUID); ok && specID != nil {
		if serviceStatus, statusOK := service["status"].(string); statusOK {
			service["status"] = h.projectedServiceStatus(c, service["id"].(uuid.UUID), *specID, servicedomain.Kind(service["kind"].(string)), serviceStatus)
		}
		h.enrichDetails(c, service, servicedomain.Kind(service["kind"].(string)), *specID)
	}
	c.JSON(http.StatusOK, service)
}

func (h *Handler) projectedServiceStatus(c *gin.Context, serviceID, specID uuid.UUID, kind servicedomain.Kind, storedStatus string) servicedomain.Status {
	var latestStatus string
	var deployments int
	if err := h.db.QueryRow(c.Request.Context(), `SELECT COUNT(*), COALESCE((SELECT status FROM deployments WHERE service_id = $1 ORDER BY number DESC LIMIT 1), '') FROM deployments WHERE service_id = $1`, serviceID).Scan(&deployments, &latestStatus); err != nil {
		deployments = 0
	}
	active := latestStatus == "queued" || latestStatus == "building" || latestStatus == "starting" || latestStatus == "health_checking"
	if latestStatus == "failed" || latestStatus == "error" || latestStatus == "cancelled" {
		return servicedomain.StatusFailed
	}
	states, err := runtimeContainerStates(c, h.runtime, serviceID, specID)
	if err == nil && len(states) > 0 {
		return servicedomain.ProjectStatus(kind, states, active, deployments > 0)
	}
	if active {
		return servicedomain.StatusDeploying
	}
	switch strings.ToLower(storedStatus) {
	case "pending", "creating", "validating", "unknown":
		if deployments == 0 {
			return servicedomain.StatusPending
		}
	case "running", "healthy", "ready":
		return servicedomain.StatusRunning
	case "stopped", "exited":
		return servicedomain.StatusStopped
	case "failed", "error":
		return servicedomain.StatusFailed
	}
	return servicedomain.StatusPending
}

func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	kind, specID, err := h.resolve(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	var input struct {
		Name           *string `json:"name"`
		Port           *int    `json:"port"`
		BuildType      *string `json:"build_type"`
		ImageRetention *int    `json:"image_retention"`
		Resources      *struct {
			CPUs      *string `json:"cpus"`
			MemMB     *int    `json:"mem_mb"`
			StorageMB *int    `json:"storage_mb"`
		} `json:"resources"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service update"})
		return
	}
	name := (*string)(nil)
	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		if trimmed == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "service name is required"})
			return
		}
		name = &trimmed
	}
	if input.BuildType != nil {
		switch *input.BuildType {
		case "dockerfile", "buildpacks", "custom", "compose":
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid build type"})
			return
		}
		if kind != string(servicedomain.KindApp) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "build type is only available for applications"})
			return
		}
	}
	if _, err := h.db.Exec(c.Request.Context(), `UPDATE services SET name = COALESCE($2, name), updated_at = now() WHERE id = $1 AND org_id = $3`, id, name, orgID(c)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service could not be updated"})
		return
	}
	if kind == string(servicedomain.KindApp) {
		var cpus *string
		var memMB, storageMB *int
		if input.Resources != nil {
			cpus = input.Resources.CPUs
			memMB = input.Resources.MemMB
			storageMB = input.Resources.StorageMB
		}
		if _, err := h.db.Exec(c.Request.Context(), `UPDATE apps SET port = COALESCE($2, port), cpus = COALESCE($3, cpus), mem_mb = COALESCE($4, mem_mb), storage_mb = COALESCE($5, storage_mb), image_retention = COALESCE($6, image_retention), build_type = COALESCE($7, build_type), updated_at = now() WHERE id = $1`, specID, input.Port, cpus, memMB, storageMB, input.ImageRetention, input.BuildType); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "application specification could not be updated"})
			return
		}
	}
	if kind == string(servicedomain.KindDatabase) && input.Port != nil {
		if _, err := h.db.Exec(c.Request.Context(), `UPDATE databases SET port = $2, updated_at = now() WHERE id = $1`, specID, *input.Port); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database specification could not be updated"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"service_id": id})
}

func (h *Handler) enrichDetails(c *gin.Context, service gin.H, kind servicedomain.Kind, specID uuid.UUID) {
	states, err := runtimeContainerStates(c, h.runtime, service["id"].(uuid.UUID), specID)
	if err == nil {
		containers := make([]gin.H, 0, len(states))
		for _, state := range states {
			containers = append(containers, gin.H{"id": state.ID, "name": state.Name, "status": state.Status, "healthy": state.Healthy})
		}
		service["runtime"] = gin.H{"containers": containers}
	}
	service["volumes"] = h.serviceVolumes(c, service["id"].(uuid.UUID))
	h.enrichSpec(c, service, kind, specID)
}

func (h *Handler) Volumes(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	if _, _, err = h.resolve(c, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"service_id": id, "volumes": h.serviceVolumes(c, id)})
}

func (h *Handler) serviceVolumes(c *gin.Context, serviceID uuid.UUID) []gin.H {
	rows, err := h.db.Query(c.Request.Context(), `SELECT id, name, mount_path FROM app_volumes WHERE service_id = $1 ORDER BY name`, serviceID)
	if err != nil {
		return []gin.H{}
	}
	defer rows.Close()
	volumes := make([]gin.H, 0)
	for rows.Next() {
		var id uuid.UUID
		var name, mountPath string
		if rows.Scan(&id, &name, &mountPath) == nil {
			volumes = append(volumes, gin.H{"id": id, "service_id": serviceID, "name": name, "mount_path": mountPath})
		}
	}
	return volumes
}

func (h *Handler) enrichSpec(c *gin.Context, service gin.H, kind servicedomain.Kind, specID uuid.UUID) {
	var spec gin.H
	switch kind {
	case servicedomain.KindApp:
		var sourceType, image, gitURL, gitBranch, dockerfile, composeFile, buildType, cpus string
		var port, storageMB, memMB, imageRetention int
		if err := h.db.QueryRow(c.Request.Context(), `SELECT source_type, image, git_url, git_branch, dockerfile, compose_file, build_type, port, cpus, storage_mb, mem_mb, image_retention FROM apps WHERE id = $1`, specID).Scan(&sourceType, &image, &gitURL, &gitBranch, &dockerfile, &composeFile, &buildType, &port, &cpus, &storageMB, &memMB, &imageRetention); err == nil {
			spec = gin.H{"source_type": sourceType, "image": image, "git_url": gitURL, "git_branch": gitBranch, "dockerfile": dockerfile, "compose_file": composeFile, "build_type": buildType, "port": port, "cpus": cpus, "storage_mb": storageMB, "mem_mb": memMB, "image_retention": imageRetention}
		}
	case servicedomain.KindCompose:
		var compose string
		var port int
		if err := h.db.QueryRow(c.Request.Context(), `SELECT compose, port FROM compose_apps WHERE id = $1`, specID).Scan(&compose, &port); err == nil {
			if port == 0 {
				if parsed, found, parseErr := templatesapplication.PublishedPort(compose); parseErr == nil && found {
					port = parsed
				}
			}
			spec = gin.H{"compose": compose, "port": port}
		}
	case servicedomain.KindDatabase:
		var engine, version, dbName, user string
		var port, memMB, storageMB int
		if err := h.db.QueryRow(c.Request.Context(), `SELECT engine, version, port, db_name, db_user, mem_mb, storage_mb FROM databases WHERE id = $1`, specID).Scan(&engine, &version, &port, &dbName, &user, &memMB, &storageMB); err == nil {
			spec = gin.H{"engine": engine, "version": version, "port": port, "database_name": dbName, "user": user, "mem_mb": memMB, "storage_mb": storageMB}
		}
	}
	if spec != nil {
		service["spec"] = spec
	}
}

func (h *Handler) Connection(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	kind, _, err := h.resolve(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	if kind != string(servicedomain.KindDatabase) || h.databaseConnection == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service connection is unavailable"})
		return
	}
	dsn, err := h.databaseConnection.ConnectionStringByServiceID(c.Request.Context(), id, orgID(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service connection is unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"dsn": dsn})
}

func (h *Handler) DeployService(ctx context.Context, id, organizationID uuid.UUID, trigger, commit string) (any, error) {
	kind, specID, err := h.resolveService(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	switch kind {
	case string(servicedomain.KindApp):
		if h.appDeploy == nil {
			return nil, errors.New("application deployment is not configured")
		}
		return h.appDeploy.Deploy(ctx, specID, organizationID, deployapplication.DeployOpts{ServiceID: id, Trigger: trigger, CommitSHA: commit})
	case string(servicedomain.KindCompose):
		deploymentID, err := h.EnqueueServiceDeployment(ctx, id, specID, organizationID, string(servicedomain.KindCompose), trigger)
		if err != nil {
			return nil, err
		}
		return gin.H{"deployment_id": deploymentID}, nil
	case string(servicedomain.KindDatabase):
		deploymentID, err := h.EnqueueServiceDeployment(ctx, id, specID, organizationID, string(servicedomain.KindDatabase), trigger)
		if err != nil {
			return nil, err
		}
		return gin.H{"deployment_id": deploymentID}, nil
	default:
		return nil, errors.New("unsupported service kind")
	}
}

func (h *Handler) Action(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	kind, specID, err := h.resolve(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	action := c.Param("action")
	ctx := c.Request.Context()
	var result any
	handled := false
	switch kind {
	case string(servicedomain.KindApp):
		if action == "deploy" && h.appDeploy != nil {
			handled = true
			result, err = h.appDeploy.Deploy(ctx, specID, orgID(c), deployapplication.DeployOpts{ServiceID: id, Trigger: "api"})
		} else if h.appOps != nil {
			switch action {
			case "start":
				handled = true
				if h.appServiceOps != nil {
					result, err = h.appServiceOps.StartService(ctx, specID, id, orgID(c))
				} else {
					result, err = h.appOps.Start(ctx, specID, orgID(c))
				}
			case "stop":
				handled = true
				if h.appServiceOps != nil {
					result, err = h.appServiceOps.StopService(ctx, specID, id, orgID(c))
				} else {
					result, err = h.appOps.Stop(ctx, specID, orgID(c))
				}
			case "restart":
				handled = true
				if h.appServiceOps != nil {
					result, err = h.appServiceOps.RestartService(ctx, specID, id, orgID(c))
				} else {
					result, err = h.appOps.Restart(ctx, specID, orgID(c))
				}
			case "delete":
				handled = true
				if h.appServiceOps != nil {
					err = h.appServiceOps.DeleteService(ctx, specID, id, orgID(c))
				} else {
					err = h.appOps.Delete(ctx, specID, orgID(c))
				}
			}
		}
	case string(servicedomain.KindCompose):
		if h.compose != nil {
			switch action {
			case "deploy", "start":
				handled = true
				if action == "deploy" {
					var deploymentID uuid.UUID
					deploymentID, err = h.enqueueServiceDeployment(ctx, id, specID, orgID(c), kind)
					result = gin.H{"deployment_id": deploymentID}
				} else {
					err = h.compose.Start(ctx, specID, orgID(c))
				}
			case "stop":
				handled = true
				err = h.compose.Stop(ctx, specID, orgID(c))
			case "restart":
				handled = true
				err = h.compose.Restart(ctx, specID, orgID(c))
			case "delete":
				handled = true
				err = h.compose.Delete(ctx, specID, orgID(c))
			}
		}
	case string(servicedomain.KindDatabase):
		if h.database != nil {
			switch action {
			case "deploy":
				handled = true
				var deploymentID uuid.UUID
				deploymentID, err = h.enqueueServiceDeployment(ctx, id, specID, orgID(c), kind)
				result = gin.H{"deployment_id": deploymentID}
			case "start":
				handled = true
				result, err = h.database.Start(ctx, specID, orgID(c))
			case "stop":
				handled = true
				result, err = h.database.Stop(ctx, specID, orgID(c))
			case "restart":
				handled = true
				_, err = h.database.Stop(ctx, specID, orgID(c))
				if err == nil {
					result, err = h.database.Start(ctx, specID, orgID(c))
				}
			case "delete":
				handled = true
				err = h.database.Delete(ctx, specID, orgID(c))
			}
		}
	}
	if !handled && err == nil {
		err = errors.New("unsupported service action")
	}
	if err == nil && action == "delete" {
		var result pgconn.CommandTag
		if result, err = h.db.Exec(ctx, `DELETE FROM services WHERE id = $1 AND org_id = $2`, id, orgID(c)); err != nil || result.RowsAffected() != 1 {
			err = errors.New("service identity could not be removed")
		}
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"service_id": id, "action": action, "result": result})
}

func (h *Handler) enqueueServiceDeployment(ctx context.Context, serviceID, specID, organizationID uuid.UUID, kind string) (uuid.UUID, error) {
	return h.EnqueueServiceDeployment(ctx, serviceID, specID, organizationID, kind, "deploy")
}

func (h *Handler) EnqueueServiceDeployment(ctx context.Context, serviceID, specID, organizationID uuid.UUID, kind, trigger string) (uuid.UUID, error) {
	if h.deploymentQueue == nil {
		return uuid.Nil, errors.New("deployment queue is not configured")
	}
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)
	var active uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM deployments WHERE service_id = $1 AND status IN ('queued', 'building', 'starting', 'health_checking') ORDER BY created_at, id LIMIT 1`, serviceID).Scan(&active); err == nil {
		return uuid.Nil, deploydomain.ErrConflict
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	var deploymentID uuid.UUID
	appID := any(nil)
	composeYAML := ""
	numberQuery := `SELECT COALESCE(MAX(number), 0) + 1 FROM deployments WHERE service_id = $1`
	if kind == string(servicedomain.KindApp) {
		appID = specID
		numberQuery = `SELECT COALESCE(MAX(number), 0) + 1 FROM deployments WHERE app_id = $1`
	} else if kind == string(servicedomain.KindCompose) {
		if err := tx.QueryRow(ctx, `SELECT compose FROM compose_apps WHERE id = $1`, specID).Scan(&composeYAML); err != nil {
			return uuid.Nil, err
		}
	}
	var number int
	numberKey := serviceID
	if kind == string(servicedomain.KindApp) {
		numberKey = specID
	}
	if err := tx.QueryRow(ctx, numberQuery, numberKey).Scan(&number); err != nil {
		return uuid.Nil, err
	}
	if err := tx.QueryRow(ctx, `INSERT INTO deployments (app_id, service_id, number, status, trigger, triggered_by, compose_yaml) VALUES ($1, $2, $3, 'queued', $4, 'user', $5) RETURNING id`, appID, serviceID, number, trigger, composeYAML).Scan(&deploymentID); err != nil {
		return uuid.Nil, err
	}
	payload, err := json.Marshal(queue.Job{ID: deploymentID.String(), Type: "deployment.execute", DeploymentID: deploymentID.String(), AppID: specID.String(), OrgID: organizationID.String(), Payload: mustJSON(map[string]string{
		"kind": kind, "service_id": serviceID.String(), "spec_id": specID.String(), "org_id": organizationID.String(), "deployment_id": deploymentID.String(),
	})})
	if err != nil {
		return uuid.Nil, err
	}
	eventPayload, err := json.Marshal(events.Event{ID: deploymentID.String(), Type: "deployment.queued", AggregateType: "deployment", AggregateID: deploymentID.String(), Payload: payload, TS: time.Now().UTC()})
	if err != nil {
		return uuid.Nil, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO outbox_events (id, topic, event_type, aggregate_type, aggregate_id, payload) VALUES ($1, 'deployments', 'deployment.queued', 'deployment', $2, $3)`, deploymentID, deploymentID.String(), eventPayload); err != nil {
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	slog.Info("deployment enqueued", "deployment_id", deploymentID, "service_id", serviceID, "kind", kind, "org_id", organizationID)
	if h.notifier != nil {
		h.notifier.NotifyDeploy(ctx, deploydomain.DeployEvent{ServiceID: serviceID, DepID: deploymentID, Status: string(deploydomain.StatusQueued)})
	}
	return deploymentID, nil
}

func mustJSON(value map[string]string) []byte {
	raw, _ := json.Marshal(value)
	return raw
}

func (h *Handler) Timeline(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	kind, specID, err := h.resolve(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	var result any
	switch kind {
	case string(servicedomain.KindApp):
		result, err = h.appOps.Timeline(c.Request.Context(), specID, orgID(c))
	case string(servicedomain.KindCompose):
		result, err = h.compose.Timeline(c.Request.Context(), specID, orgID(c))
	default:
		result = []any{}
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) Logs(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	_, specID, err := h.resolve(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	containers, err := serviceContainers(c, h.runtime, id, specID)
	if err != nil || len(containers) == 0 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "service logs unavailable"})
		return
	}
	if selected := strings.TrimSpace(c.Query("container")); selected != "" {
		filtered := make([]string, 0, 1)
		for _, container := range containers {
			parts := strings.SplitN(container, "|", 2)
			if selected == parts[0] || len(parts) == 2 && selected == parts[1] {
				filtered = append(filtered, container)
				break
			}
		}
		if len(filtered) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
			return
		}
		containers = filtered
	}
	var output []byte
	for _, container := range containers {
		containerID := strings.SplitN(container, "|", 2)[0]
		logs, logsErr := h.runtime.LogTail(c.Request.Context(), containerID, 200)
		if logsErr == nil || len(logs) > 0 {
			output = append(output, []byte(strings.Join(logs, "\n"))...)
			output = append(output, '\n')
		}
	}
	if c.Query("follow") == "1" {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.String(http.StatusOK, "data: %s\n\n", strings.ReplaceAll(string(output), "\n", "\ndata: "))
		return
	}
	c.JSON(http.StatusOK, gin.H{"service_id": id, "logs": string(output)})
}

func (h *Handler) Containers(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	_, specID, err := h.resolve(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	items, err := runtimeContainers(c, h.runtime, id, specID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "service containers unavailable"})
		return
	}
	containers := make([]gin.H, 0, len(items))
	for _, item := range items {
		containers = append(containers, gin.H{"id": item.ID, "name": item.Name, "status": item.State})
	}
	c.JSON(http.StatusOK, containers)
}

func runtimeContainers(c *gin.Context, runtime worker.Runtime, serviceID, specID uuid.UUID) ([]worker.ContainerInfo, error) {
	if runtime == nil {
		return nil, errors.New("container runtime unavailable")
	}
	var items []worker.ContainerInfo
	var err error
	if metadataRuntime, ok := runtime.(worker.ContainerMetadataRuntime); ok {
		items, err = metadataRuntime.ListContainerMetadata(c.Request.Context())
	} else {
		items, err = runtime.ListContainers(c.Request.Context())
	}
	if err != nil {
		return nil, err
	}
	values := []uuid.UUID{serviceID}
	if specID != serviceID {
		values = append(values, specID)
	}
	matched := make([]worker.ContainerInfo, 0)
	for _, item := range items {
		for _, value := range values {
			if item.Labels["aether.service-id"] == value.String() {
				matched = append(matched, item)
				break
			}
		}
	}
	return matched, nil
}

func serviceContainers(c *gin.Context, runtime worker.Runtime, serviceID, specID uuid.UUID) ([]string, error) {
	items, err := runtimeContainers(c, runtime, serviceID, specID)
	if err != nil {
		return nil, err
	}
	containers := make([]string, 0, len(items))
	for _, item := range items {
		containers = append(containers, item.ID+"|"+item.Name)
	}
	return containers, nil
}

func runtimeContainerStates(c *gin.Context, runtime worker.Runtime, serviceID, specID uuid.UUID) ([]servicedomain.ContainerState, error) {
	items, err := runtimeContainers(c, runtime, serviceID, specID)
	if err != nil {
		return nil, err
	}
	states := make([]servicedomain.ContainerState, 0, len(items))
	for _, item := range items {
		states = append(states, servicedomain.ContainerState{ID: item.ID, Name: item.Name, Status: item.State, Healthy: item.Healthy})
	}
	return states, nil
}

func (h *Handler) Stats(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	_, specID, err := h.resolve(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	state := string(servicedomain.StatusUnknown)
	if statusErr := h.db.QueryRow(c.Request.Context(), `SELECT status FROM services WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&state); statusErr != nil {
		state = string(servicedomain.StatusUnknown)
	}
	items, err := runtimeContainers(c, h.runtime, id, specID)
	containerStats := make([]gin.H, 0)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"state": state, "stats": gin.H{"cpu_percent": 0, "mem_bytes": 0, "mem_limit": 0, "mem_percent": 0}, "containers": containerStats})
		return
	}
	var cpu, used, limit float64
	for _, item := range items {
		if !item.HasStats {
			continue
		}
		cpu += item.Stats.CPUPercent
		used += float64(item.Stats.MemBytes)
		limit += float64(item.Stats.MemLimit)
		containerStats = append(containerStats, gin.H{"id": item.ID, "name": item.Name, "cpu_percent": item.Stats.CPUPercent, "mem_bytes": item.Stats.MemBytes, "mem_limit": item.Stats.MemLimit, "mem_percent": item.Stats.MemPercent})
	}
	memPercent := float64(0)
	if limit > 0 {
		memPercent = used / limit * 100
	}
	if state == string(servicedomain.StatusUnknown) && used == 0 && limit == 0 {
		c.JSON(http.StatusOK, gin.H{"state": state, "stats": gin.H{"cpu_percent": 0, "mem_bytes": 0, "mem_limit": 0, "mem_percent": 0}, "containers": containerStats})
		return
	}
	c.JSON(http.StatusOK, gin.H{"state": state, "stats": gin.H{"cpu_percent": cpu, "mem_bytes": int64(used), "mem_limit": int64(limit), "mem_percent": memPercent}, "containers": containerStats})
}

func (h *Handler) Deployments(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	_, _, err = h.resolve(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	rows, err := h.db.Query(c.Request.Context(), `
SELECT d.id, d.number, d.status, d.created_at, d.started_at, d.finished_at
FROM deployments d
WHERE d.service_id = $1
ORDER BY d.number DESC LIMIT 50`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	defer rows.Close()
	result := make([]gin.H, 0)
	for rows.Next() {
		var deploymentID uuid.UUID
		var number int
		var status string
		var createdAt, startedAt, finishedAt any
		if err := rows.Scan(&deploymentID, &number, &status, &createdAt, &startedAt, &finishedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		result = append(result, gin.H{"id": deploymentID, "service_id": id, "number": number, "status": status, "created_at": createdAt, "started_at": startedAt, "finished_at": finishedAt})
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) CancelDeployment(c *gin.Context) {
	serviceID, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	deploymentID, err := uuid.Parse(c.Param("deploymentID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deployment id"})
		return
	}
	var status string
	var imageRef, containerID string
	if err := h.db.QueryRow(c.Request.Context(), `
SELECT status, image_ref, container_id
FROM deployments
JOIN services s ON s.id = deployments.service_id AND s.org_id = $3 AND s.deleted_at IS NULL
WHERE deployments.id = $1 AND deployments.service_id = $2`, deploymentID, serviceID, orgID(c)).Scan(&status, &imageRef, &containerID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "deployment not found"})
		return
	}
	if status != string(deploydomain.StatusQueued) && status != string(deploydomain.StatusBuilding) && status != string(deploydomain.StatusStarting) && status != string(deploydomain.StatusHealthChecking) {
		c.JSON(http.StatusConflict, gin.H{"error": "deployment is not active"})
		return
	}
	if _, err := h.db.Exec(c.Request.Context(), `UPDATE deployments SET status = 'cancelled', error = 'deployment cancelled by user', finished_at = now() WHERE id = $1 AND service_id = $2 AND status IN ('queued', 'building', 'starting', 'health_checking')`, deploymentID, serviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "deployment cancellation failed"})
		return
	}
	if h.deploymentCanceller != nil {
		h.deploymentCanceller.CancelDeployment(deploymentID)
	}
	if h.notifier != nil {
		h.notifier.NotifyDeploy(c.Request.Context(), deploydomain.DeployEvent{ServiceID: serviceID, DepID: deploymentID, Status: string(deploydomain.StatusCancelled), Detail: "deployment cancelled by user"})
	}
	c.JSON(http.StatusOK, gin.H{"service_id": serviceID, "deployment_id": deploymentID, "status": deploydomain.StatusCancelled, "image_ref": imageRef, "container_id": containerID})
}

func (h *Handler) DeploymentLog(c *gin.Context) {
	serviceID, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	deploymentID, err := uuid.Parse(c.Param("deploymentID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deployment id"})
		return
	}
	kind, specID, err := h.resolve(c, serviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	var number int
	var status, deploymentError string
	if err := h.db.QueryRow(c.Request.Context(), `SELECT number, status, error FROM deployments WHERE id = $1 AND service_id = $2`, deploymentID, serviceID).Scan(&number, &status, &deploymentError); err != nil {
		if kind == string(servicedomain.KindCompose) && deploymentID == serviceID {
			content := h.composeRuntimeLog(c, serviceID, specID)
			status = "ready"
			var composeStatus string
			if statusErr := h.db.QueryRow(c.Request.Context(), `SELECT status FROM compose_apps WHERE id = $1`, specID).Scan(&composeStatus); statusErr == nil {
				switch composeStatus {
				case "pending":
					status = "queued"
				case "deploying":
					status = "starting"
				case "stopped":
					status = "cancelled"
				case "error":
					status = "failed"
				}
			}
			c.JSON(http.StatusOK, gin.H{"service_id": serviceID, "number": 1, "status": status, "error": "", "content": content})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "deployment not found"})
		return
	}
	content := ""
	if h.logsDir != "" {
		if raw, readErr := os.ReadFile(filepath.Join(h.logsDir, "deployments", deploymentID.String()+".log")); readErr == nil {
			content = string(raw)
		}
	}
	c.JSON(http.StatusOK, gin.H{"service_id": serviceID, "number": number, "status": status, "error": deploymentError, "content": content})
}

func (h *Handler) composeRuntimeLog(c *gin.Context, serviceID, specID uuid.UUID) string {
	containers, err := serviceContainers(c, h.runtime, serviceID, specID)
	if err != nil {
		return ""
	}
	var builder strings.Builder
	for _, container := range containers {
		parts := strings.SplitN(container, "|", 2)
		if len(parts) == 0 || parts[0] == "" {
			continue
		}
		lines, logErr := h.runtime.LogTail(c.Request.Context(), parts[0], 200)
		if logErr != nil && len(lines) == 0 {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		if len(parts) == 2 {
			builder.WriteString("[" + parts[1] + "]\n")
		}
		builder.WriteString(strings.Join(lines, "\n"))
	}
	return builder.String()
}

func (h *Handler) Domains(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	if _, _, err = h.resolve(c, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	rows, err := h.db.Query(c.Request.Context(), `SELECT id, service_type, server_id, host, https, path, internal_path, strip_path, container_port, status, cert_status, created_at, updated_at FROM domains WHERE service_id = $1 ORDER BY host`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	defer rows.Close()
	result := make([]gin.H, 0)
	for rows.Next() {
		var domainID uuid.UUID
		var serviceType, host, status, certStatus string
		var serverID *uuid.UUID
		var path, internalPath string
		var stripPath bool
		var containerPort int
		var https bool
		var createdAt, updatedAt any
		if err := rows.Scan(&domainID, &serviceType, &serverID, &host, &https, &path, &internalPath, &stripPath, &containerPort, &status, &certStatus, &createdAt, &updatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		result = append(result, gin.H{"id": domainID, "service_id": id, "service_type": serviceType, "server_id": serverID, "host": host, "https": https, "path": path, "internal_path": internalPath, "strip_path": stripPath, "container_port": containerPort, "status": status, "cert_status": certStatus, "created_at": createdAt, "updated_at": updatedAt})
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) Environment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	_, _, err = h.resolve(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	rows, err := h.db.Query(c.Request.Context(), `SELECT name, value, secret FROM app_env WHERE service_id = $1 ORDER BY name`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	defer rows.Close()
	result := make([]gin.H, 0)
	for rows.Next() {
		var name, value string
		var secret bool
		if err := rows.Scan(&name, &value, &secret); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		if secret {
			value = ""
		}
		result = append(result, gin.H{"name": name, "value": value, "secret": secret})
	}
	c.JSON(http.StatusOK, gin.H{"env": result})
}

func (h *Handler) AddDomain(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil || h.domains == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	kind, specID, err := h.resolve(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	var input struct {
		Host          string `json:"host"`
		HTTPS         bool   `json:"https"`
		ContainerPort int    `json:"container_port"`
		Path          string `json:"path"`
		InternalPath  string `json:"internal_path"`
		StripPath     bool   `json:"strip_path"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain"})
		return
	}
	serviceType := kind
	if kind == string(servicedomain.KindDatabase) {
		serviceType = domainsapplication.ServiceTypeDB
	}
	domain, err := h.domains.Add(c.Request.Context(), specID, orgID(c), serviceType, domainsapplication.AddDomainInput{Host: input.Host, HTTPS: input.HTTPS, ContainerPort: input.ContainerPort, Path: input.Path, InternalPath: input.InternalPath, StripPath: input.StripPath})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.db.Exec(c.Request.Context(), `UPDATE domains SET service_id = $1 WHERE id = $2`, id, domain.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not link domain to service"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": domain.ID, "service_id": id, "host": domain.Host, "https": domain.HTTPS, "status": domain.Status, "cert_status": domain.CertStatus})
}

func (h *Handler) RemoveDomain(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil || h.domains == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	kind, specID, err := h.resolve(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	serviceType := kind
	if kind == string(servicedomain.KindDatabase) {
		serviceType = domainsapplication.ServiceTypeDB
	}
	if err := h.domains.Remove(c.Request.Context(), specID, orgID(c), serviceType, c.Param("host")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GenerateDomain(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil || h.domains == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	kind, specID, err := h.resolve(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	serviceType := kind
	if kind == string(servicedomain.KindDatabase) {
		serviceType = domainsapplication.ServiceTypeDB
	}
	domain, err := h.domains.GenerateFreeDomain(c.Request.Context(), specID, orgID(c), serviceType, true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.db.Exec(c.Request.Context(), `UPDATE domains SET service_id = $1 WHERE id = $2`, id, domain.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not link domain to service"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": domain.ID, "service_id": id, "host": domain.Host, "https": domain.HTTPS, "status": domain.Status, "cert_status": domain.CertStatus})
}

func (h *Handler) SetEnvironment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	_, _, err = h.resolve(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	var input struct {
		Name   string `json:"name"`
		Value  string `json:"value"`
		Secret bool   `json:"secret"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid environment variable"})
		return
	}
	if strings.TrimSpace(input.Name) == "" || strings.Contains(input.Name, "=") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid environment variable"})
		return
	}
	if _, err := h.db.Exec(c.Request.Context(), `INSERT INTO app_env (service_id, name, value, secret) VALUES ($1, $2, $3, $4) ON CONFLICT (service_id, name) DO UPDATE SET value = EXCLUDED.value, secret = EXCLUDED.secret`, id, input.Name, input.Value, input.Secret); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save environment variable"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ImportMissingEnvVars(ctx context.Context, serviceID, organizationID uuid.UUID, names []string) (int, error) {
	if _, _, err := h.resolveService(ctx, organizationID, serviceID); err != nil {
		return 0, err
	}
	inserted := 0
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || strings.Contains(name, "=") {
			continue
		}
		result, err := h.db.Exec(ctx, `INSERT INTO app_env (service_id, name, value, secret) VALUES ($1, $2, '', false) ON CONFLICT (service_id, name) DO NOTHING`, serviceID, name)
		if err != nil {
			return inserted, err
		}
		inserted += int(result.RowsAffected())
	}
	return inserted, nil
}

func (h *Handler) DeleteEnvironment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	_, _, err = h.resolve(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	if _, err := h.db.Exec(c.Request.Context(), `DELETE FROM app_env WHERE service_id = $1 AND name = $2`, id, c.Param("name")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete environment variable"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) CronJobs(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	kind, _, err := h.resolve(c, id)
	if err != nil || kind != string(servicedomain.KindApp) {
		c.JSON(http.StatusNotFound, gin.H{"error": "service schedules are unavailable"})
		return
	}
	rows, err := h.db.Query(c.Request.Context(), `
SELECT id, name, schedule, command, enabled, last_run, next_run, created_at
FROM cron_jobs WHERE service_id = $1 ORDER BY created_at`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	defer rows.Close()
	result := make([]gin.H, 0)
	for rows.Next() {
		var jobID uuid.UUID
		var name, schedule, command string
		var enabled bool
		var lastRun, nextRun, createdAt any
		if err := rows.Scan(&jobID, &name, &schedule, &command, &enabled, &lastRun, &nextRun, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		result = append(result, gin.H{"id": jobID, "service_id": id, "name": name, "schedule": schedule, "command": command, "enabled": enabled, "last_run": lastRun, "next_run": nextRun, "created_at": createdAt})
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) CreateCronJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	kind, specID, err := h.resolve(c, id)
	if err != nil || kind != string(servicedomain.KindApp) {
		c.JSON(http.StatusNotFound, gin.H{"error": "service schedules are unavailable"})
		return
	}
	var input struct {
		Name     string `json:"name"`
		Schedule string `json:"schedule"`
		Command  string `json:"command"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Command) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cron job"})
		return
	}
	if _, err := cron.ParseStandard(input.Schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cron schedule"})
		return
	}
	var jobID uuid.UUID
	var createdAt any
	name := strings.TrimSpace(input.Name)
	command := strings.TrimSpace(input.Command)
	err = h.db.QueryRow(c.Request.Context(), `
INSERT INTO cron_jobs (app_id, service_id, name, schedule, command, enabled)
VALUES ($1, $2, $3, $4, $5, true)
RETURNING id, created_at`, specID, id, name, input.Schedule, command).Scan(&jobID, &createdAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cron job could not be created"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": jobID, "service_id": id, "name": name, "schedule": input.Schedule, "command": command, "enabled": true, "created_at": createdAt})
}

func (h *Handler) DeleteCronJob(c *gin.Context) {
	serviceID, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	if _, _, err := h.resolve(c, serviceID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	jobID, err := uuid.Parse(c.Param("jobID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cron job id"})
		return
	}
	result, err := h.db.Exec(c.Request.Context(), `DELETE FROM cron_jobs WHERE id = $1 AND service_id = $2`, jobID, serviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cron job could not be deleted"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "cron job not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) SetWebhook(c *gin.Context) {
	serviceID, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	kind, specID, err := h.resolve(c, serviceID)
	if err != nil || kind != string(servicedomain.KindApp) || h.appWebhook == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service webhook is unavailable"})
		return
	}
	var input struct {
		Secret string `json:"secret"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook"})
		return
	}
	if err := h.appWebhook.SetWebhook(c.Request.Context(), specID, orgID(c), input.Secret); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "webhook could not be saved"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "service_id": serviceID})
}

func parseMemory(value string) (int64, int64) {
	parts := strings.SplitN(strings.TrimSpace(value), "/", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	return parseMemoryValue(parts[0]), parseMemoryValue(parts[1])
}

func parseMemoryValue(value string) int64 {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return 0
	}
	number, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	multiplier := float64(1)
	if len(fields) > 1 {
		switch strings.ToUpper(fields[1]) {
		case "KB", "KIB":
			multiplier = 1024
		case "MB", "MIB":
			multiplier = 1024 * 1024
		case "GB", "GIB":
			multiplier = 1024 * 1024 * 1024
		}
	}
	return int64(number * multiplier)
}

func (h *Handler) resolve(c *gin.Context, id uuid.UUID) (string, uuid.UUID, error) {
	return h.resolveService(c.Request.Context(), orgID(c), id)
}

func (h *Handler) resolveService(ctx context.Context, organizationID, id uuid.UUID) (string, uuid.UUID, error) {
	var kind string
	if err := h.db.QueryRow(ctx, `SELECT kind FROM services WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL`, id, organizationID).Scan(&kind); err != nil {
		return "", uuid.Nil, err
	}
	var specID uuid.UUID
	var query string
	switch kind {
	case string(servicedomain.KindApp):
		query = `SELECT id FROM apps WHERE service_id = $1`
	case string(servicedomain.KindCompose):
		query = `SELECT id FROM compose_apps WHERE service_id = $1`
	case string(servicedomain.KindDatabase):
		query = `SELECT id FROM databases WHERE service_id = $1`
	default:
		return "", uuid.Nil, errors.New("unsupported service kind")
	}
	if err := h.db.QueryRow(ctx, query, id).Scan(&specID); err != nil {
		return "", uuid.Nil, err
	}
	return kind, specID, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanService(row rowScanner) (gin.H, error) {
	var id, organizationID, projectID uuid.UUID
	var environmentID *uuid.UUID
	var name, kind, status string
	var specID *uuid.UUID
	var createdAt, updatedAt any
	if err := row.Scan(&id, &organizationID, &projectID, &environmentID, &name, &kind, &status, &createdAt, &updatedAt, &specID); err != nil {
		return nil, err
	}
	serviceKind := servicedomain.Kind(kind)
	return gin.H{
		"id": id, "org_id": organizationID, "project_id": projectID, "environment_id": environmentID,
		"name": name, "kind": kind, "status": status,
		"spec_id":      specID,
		"capabilities": servicedomain.CapabilitiesFor(serviceKind),
		"created_at":   createdAt, "updated_at": updatedAt,
	}, nil
}

func orgID(c *gin.Context) uuid.UUID {
	return c.MustGet(authhttp.ContextOrgID).(uuid.UUID)
}
