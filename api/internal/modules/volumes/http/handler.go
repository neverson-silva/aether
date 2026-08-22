package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/modules/auth/http"
	"aether/internal/modules/volumes/application"
	"aether/internal/modules/volumes/domain"
)

type Handler struct {
	volumes *application.Volumes
}

func New(volumes *application.Volumes) *Handler {
	return &Handler{volumes: volumes}
}

type backupReq struct {
	DestinationID string `json:"destination_id"`
}

func (h *Handler) BackupVolume(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req backupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	destID, err := uuid.Parse(req.DestinationID)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	backup, err := h.volumes.BackupVolume(c.Request.Context(), appID, orgID(c), destID, c.Param("name"))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id": backup.ID, "org_id": backup.OrgID, "app_id": backup.AppID,
		"path": backup.Path, "size": backup.Size, "kind": backup.Kind,
		"dest": backup.Dest, "created_at": backup.CreatedAt,
	})
}

func (h *Handler) List(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	volumes, err := h.volumes.List(c.Request.Context(), appID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(volumes))
	for i := range volumes {
		out = append(out, gin.H{
			"id": volumes[i].ID, "app_id": volumes[i].AppID,
			"name": volumes[i].Name, "mount_path": volumes[i].MountPath,
		})
	}
	c.JSON(http.StatusOK, gin.H{"volumes": out})
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
