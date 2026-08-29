package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/modules/auth/http"
	"aether/internal/modules/backups/application"
	"aether/internal/modules/backups/domain"
)

type Handler struct {
	backups *application.Backups
}

func New(backups *application.Backups) *Handler {
	return &Handler{backups: backups}
}

type restoreDBReq struct {
	BackupID string `json:"backup_id"`
}

func (h *Handler) CreateDatabaseBackup(c *gin.Context) {
	dbID, err := uuid.Parse(c.Param("dbID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	backup, err := h.backups.CreateDatabase(c.Request.Context(), dbID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, backupDTO(backup))
}

func (h *Handler) RestoreDatabase(c *gin.Context) {
	dbID, err := uuid.Parse(c.Param("dbID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req restoreDBReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	backupID, err := uuid.Parse(req.BackupID)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if h.backups.Async == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "async restore backend is not configured"})
		return
	}
	job, err := h.backups.Async.RequestRestore(c.Request.Context(), backupID, dbID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{
		"id": job.ID, "backup_id": job.BackupID, "target_database_id": job.TargetDatabaseID,
		"service_id": job.ServiceID,
		"status": job.Status, "error_code": job.ErrorCode, "error_message": job.ErrorMessage,
	})
}

func (h *Handler) CreateStateBackup(c *gin.Context) {
	backup, err := h.backups.CreateState(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, backupDTO(backup))
}

func (h *Handler) List(c *gin.Context) {
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	backups, err := h.backups.List(c.Request.Context(), orgID(c), limit)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(backups))
	for i := range backups {
		out = append(out, backupDTO(&backups[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) RestoreState(c *gin.Context) {
	backupID, err := uuid.Parse(c.Param("backupID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.backups.RestoreState(c.Request.Context(), backupID, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "restored"})
}

func backupDTO(b *domain.Backup) gin.H {
	var appID any
	if b.DatabaseID != nil {
		appID = b.DatabaseID
	} else {
		appID = ""
	}
	return gin.H{
		"id": b.ID, "path": b.Path, "size": b.Size, "created_at": b.CreatedAt,
		"kind": b.Kind, "dest": b.Dest, "app_id": appID, "service_id": b.ServiceID,
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
