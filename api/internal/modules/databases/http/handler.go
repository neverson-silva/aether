package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/modules/auth/http"
	"aether/internal/modules/databases/application"
	"aether/internal/modules/databases/domain"
	"aether/internal/platform/hostinfo"
)

type Handler struct {
	databases *application.Databases
	studio    *application.Studio
}

func New(databases *application.Databases, studio *application.Studio) *Handler {
	return &Handler{databases: databases, studio: studio}
}

type createDBReq struct {
	ProjectID     string `json:"project_id"`
	EnvironmentID string `json:"environment_id"`
	Name          string `json:"name"`
	Engine        string `json:"engine"`
	Version       string `json:"version"`
	User          string `json:"user"`
	Password      string `json:"password"`
	MemMB         *int   `json:"mem_mb"`
	StorageMB     *int   `json:"storage_mb"`
}

func (h *Handler) Create(c *gin.Context) {
	var req createDBReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	memMB, storageMB := 0, 0
	if req.MemMB != nil {
		memMB = *req.MemMB
	}
	if req.StorageMB != nil {
		storageMB = *req.StorageMB
	}
	var database *domain.Database
	if req.EnvironmentID == "" {
		database, err = h.databases.Create(c.Request.Context(), orgID(c), projectID, req.Name, domain.Engine(req.Engine), req.Version, req.User, req.Password, memMB, storageMB)
	} else {
		environmentID, parseErr := uuid.Parse(req.EnvironmentID)
		if parseErr != nil {
			abort(c, domain.ErrValidation)
			return
		}
		database, err = h.databases.CreateInEnvironment(c.Request.Context(), orgID(c), projectID, environmentID, req.Name, domain.Engine(req.Engine), req.Version, req.User, req.Password, memMB, storageMB)
	}
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, databaseDTO(database))
}

func (h *Handler) List(c *gin.Context) {
	dbs, err := h.databases.List(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(dbs))
	for i := range dbs {
		out = append(out, databaseDTO(&dbs[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("dbID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	db, err := h.databases.Get(c.Request.Context(), id, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	dsn, _ := h.databases.ConnectionString(c.Request.Context(), id, orgID(c))
	c.JSON(http.StatusOK, gin.H{"database": databaseDTO(db), "dsn": dsn, "public_host": hostinfo.PublicIP()})
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("dbID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.databases.Delete(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) Deploy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("dbID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	db, err := h.databases.Deploy(c.Request.Context(), id, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, databaseDTO(db))
}

func (h *Handler) Rebuild(c *gin.Context) {
	id, err := uuid.Parse(c.Param("dbID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	db, err := h.databases.Rebuild(c.Request.Context(), id, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, databaseDTO(db))
}

func (h *Handler) Start(c *gin.Context) {
	id, err := uuid.Parse(c.Param("dbID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	db, err := h.databases.Start(c.Request.Context(), id, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, databaseDTO(db))
}

func (h *Handler) Stop(c *gin.Context) {
	id, err := uuid.Parse(c.Param("dbID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	db, err := h.databases.Stop(c.Request.Context(), id, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, databaseDTO(db))
}

func (h *Handler) ListDeployments(c *gin.Context) {
	id, err := uuid.Parse(c.Param("dbID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	limit := 25
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	deps, err := h.databases.ListDeployments(c.Request.Context(), id, orgID(c), limit)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(deps))
	for i := range deps {
		var started, finished any
		if deps[i].StartedAt != nil {
			started = deps[i].StartedAt
		}
		if deps[i].FinishedAt != nil {
			finished = deps[i].FinishedAt
		}
		out = append(out, gin.H{
			"id": deps[i].ID, "number": deps[i].Number, "status": deps[i].Status,
			"trigger": deps[i].Trigger, "triggered_by": deps[i].TriggeredBy,
			"container_id": deps[i].ContainerID, "error": deps[i].Error,
			"created_at": deps[i].CreatedAt, "started_at": started, "finished_at": finished,
		})
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) DeploymentLog(c *gin.Context) {
	dbID, err := uuid.Parse(c.Param("dbID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	depID, err := uuid.Parse(c.Param("depID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	limit := 200
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	content, err := h.databases.DeploymentLogs(c.Request.Context(), dbID, depID, orgID(c), limit)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"content": content})
}

func databaseDTO(db *domain.Database) gin.H {
	return gin.H{
		"id": db.ID, "service_id": db.ServiceID, "org_id": db.OrgID, "project_id": db.ProjectID, "environment_id": db.EnvironmentID, "name": db.Name,
		"engine": db.Engine, "version": db.Version, "port": db.Port, "db_name": db.DBName,
		"user": db.User, "mem_mb": db.MemMB, "storage_mb": db.StorageMB, "status": db.Status,
		"container_id": db.ContainerID, "created_at": db.CreatedAt,
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
	case errors.Is(err, domain.ErrDatabaseUnavailable):
		c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"error": err.Error()})
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
