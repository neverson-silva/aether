package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/modules/auth/http"
	"aether/internal/modules/snapshots/application"
	"aether/internal/modules/snapshots/domain"
)

type Handler struct {
	snapshots *application.Snapshots
}

func New(snapshots *application.Snapshots) *Handler {
	return &Handler{snapshots: snapshots}
}

type createSnapshotReq struct {
	AppID  string `json:"app_id"`
	Volume string `json:"volume"`
	Name   string `json:"name"`
}

type createScheduleReq struct {
	AppID      string `json:"app_id"`
	Volume     string `json:"volume"`
	NamePrefix string `json:"name_prefix"`
	Cron       string `json:"cron"`
	Retention  *int   `json:"retention"`
	Enabled    bool   `json:"enabled"`
}

func (h *Handler) Create(c *gin.Context) {
	var req createSnapshotReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	appID, err := parseOptional(req.AppID)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	snapshot, err := h.snapshots.Create(c.Request.Context(), orgID(c), appID, req.Volume, req.Name)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, snapshotDTO(snapshot))
}

func (h *Handler) List(c *gin.Context) {
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	snapshots, err := h.snapshots.List(c.Request.Context(), orgID(c), limit)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(snapshots))
	for i := range snapshots {
		out = append(out, snapshotDTO(&snapshots[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) ListForService(c *gin.Context) {
	serviceID, err := parseServiceID(c)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	limit := 50
	if value := c.Query("limit"); value != "" {
		if parsed, parseErr := strconv.Atoi(value); parseErr == nil {
			limit = parsed
		}
	}
	snapshots, err := h.snapshots.ListForService(c.Request.Context(), orgID(c), serviceID, limit)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(snapshots))
	for index := range snapshots {
		out = append(out, snapshotDTO(&snapshots[index]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) CreateForService(c *gin.Context) {
	serviceID, err := parseServiceID(c)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req struct {
		Volume string `json:"volume"`
		Name   string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	snapshot, err := h.snapshots.CreateForService(c.Request.Context(), serviceID, orgID(c), req.Volume, req.Name)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, snapshotDTO(snapshot))
}

func (h *Handler) Restore(c *gin.Context) {
	id, err := uuid.Parse(c.Param("snapshotID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.snapshots.Restore(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "restored"})
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("snapshotID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.snapshots.Delete(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) ListSchedules(c *gin.Context) {
	schedules, err := h.snapshots.ListSchedules(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(schedules))
	for i := range schedules {
		out = append(out, scheduleDTO(&schedules[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) ListSchedulesForService(c *gin.Context) {
	serviceID, err := parseServiceID(c)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	schedules, err := h.snapshots.ListSchedulesForService(c.Request.Context(), orgID(c), serviceID)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(schedules))
	for index := range schedules {
		out = append(out, scheduleDTO(&schedules[index]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) CreateScheduleForService(c *gin.Context) {
	serviceID, err := parseServiceID(c)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req struct {
		Volume     string `json:"volume"`
		NamePrefix string `json:"name_prefix"`
		Cron       string `json:"cron"`
		Retention  *int   `json:"retention"`
		Enabled    bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	retention := 7
	if req.Retention != nil {
		retention = *req.Retention
	}
	schedule, err := h.snapshots.CreateScheduleForService(c.Request.Context(), serviceID, orgID(c), req.Volume, req.NamePrefix, req.Cron, retention, req.Enabled)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, scheduleDTO(schedule))
}

func (h *Handler) CreateSchedule(c *gin.Context) {
	var req createScheduleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	appID, err := parseOptional(req.AppID)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	retention := 7
	if req.Retention != nil {
		retention = *req.Retention
	}
	schedule, err := h.snapshots.CreateSchedule(c.Request.Context(), orgID(c), appID, req.Volume, req.NamePrefix, req.Cron, retention, req.Enabled)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, scheduleDTO(schedule))
}

func (h *Handler) DeleteSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("scheduleID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.snapshots.DeleteSchedule(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func snapshotDTO(s *domain.Snapshot) gin.H {
	return gin.H{
		"id": s.ID, "org_id": s.OrgID, "app_id": s.AppID, "service_id": s.ServiceID, "volume": s.Volume,
		"name": s.Name, "size": s.Size, "chunks": s.Chunks, "dedup_saved": s.DedupSaved,
		"created_at": s.CreatedAt,
	}
}

func scheduleDTO(s *domain.Schedule) gin.H {
	return gin.H{
		"id": s.ID, "org_id": s.OrgID, "app_id": s.AppID, "service_id": s.ServiceID, "volume": s.Volume,
		"name_prefix": s.NamePrefix, "cron": s.Cron, "retention": s.Retention,
		"enabled": s.Enabled, "last_run": s.LastRun, "next_run": s.NextRun, "created_at": s.CreatedAt,
	}
}

func parseOptional(raw string) (*uuid.UUID, error) {
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func parseServiceID(c *gin.Context) (uuid.UUID, error) {
	return uuid.Parse(c.Param("serviceID"))
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
