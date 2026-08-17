package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/auth/http"
	"aether/internal/databases/application"
	"aether/internal/databases/domain"
)

type Handler struct {
	databases *application.Databases
}

func New(databases *application.Databases) *Handler {
	return &Handler{databases: databases}
}

type createDBReq struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Engine    string `json:"engine"`
	Version   string `json:"version"`
	MemMB     *int   `json:"mem_mb"`
	StorageMB *int   `json:"storage_mb"`
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
	db, err := h.databases.Create(c.Request.Context(), orgID(c), projectID, req.Name, domain.Engine(req.Engine), req.Version, memMB, storageMB)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, databaseDTO(db))
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
	c.JSON(http.StatusOK, gin.H{"database": databaseDTO(db), "dsn": dsn})
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

func databaseDTO(db *domain.Database) gin.H {
	return gin.H{
		"id": db.ID, "org_id": db.OrgID, "project_id": db.ProjectID, "name": db.Name,
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
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
