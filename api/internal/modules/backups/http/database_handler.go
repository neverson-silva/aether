package http

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authdomain "aether/internal/modules/auth/domain"
	authhttp "aether/internal/modules/auth/http"
	"aether/internal/modules/backups/application"
	"aether/internal/modules/backups/domain"
	databasedomain "aether/internal/modules/databases/domain"
)

type DBBackupHandler struct {
	svc *application.DatabaseBackups
}

func NewDatabaseBackupHandler(svc *application.DatabaseBackups) *DBBackupHandler {
	return &DBBackupHandler{svc: svc}
}

func (h *DBBackupHandler) dbID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("dbID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return uuid.Nil, false
	}
	return id, true
}

func (h *DBBackupHandler) ResolveServiceDatabase(c *gin.Context) {
	serviceID, err := uuid.Parse(c.Param("serviceID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	resolver, ok := h.svc.Databases.(interface {
		GetByServiceID(context.Context, uuid.UUID, uuid.UUID) (*databasedomain.Database, error)
	})
	if !ok {
		abort(c, domain.ErrNotFound)
		return
	}
	database, err := resolver.GetByServiceID(c.Request.Context(), serviceID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.Set("aether.database_spec_id", database.ID)
	c.Params = append(c.Params, gin.Param{Key: "dbID", Value: database.ID.String()})
	c.Next()
}

func (h *DBBackupHandler) manage(c *gin.Context) bool {
	role, _ := c.MustGet(authhttp.ContextRole).(string)
	if !authdomain.Role(role).CanManage() {
		abort(c, domain.ErrForbidden)
		return false
	}
	return true
}

type scheduleDTO struct {
	Type      string `json:"type"`
	Minute    int    `json:"minute,omitempty"`
	At        string `json:"at,omitempty"`
	DayOfWeek string `json:"day_of_week,omitempty"`
	StartDate string `json:"start_date,omitempty"`
	Cron      string `json:"cron,omitempty"`
	Timezone  string `json:"timezone"`
}

type retentionDTO struct {
	Type string `json:"type"`
}

type backupConfigReq struct {
	Enabled       bool         `json:"enabled"`
	DestinationID string       `json:"destination_id"`
	PathPrefix    string       `json:"path_prefix"`
	Schedule      scheduleDTO  `json:"schedule"`
	Retention     retentionDTO `json:"retention"`
}

type backupConfigDTO struct {
	ID            string       `json:"id"`
	DatabaseID    string       `json:"database_id"`
	ServiceID     string       `json:"service_id"`
	Enabled       bool         `json:"enabled"`
	DestinationID string       `json:"destination_id"`
	PathPrefix    string       `json:"path_prefix"`
	Schedule      scheduleDTO  `json:"schedule"`
	Retention     retentionDTO `json:"retention"`
	NextRunAt     *string      `json:"next_run_at"`
}

func (h *DBBackupHandler) ListConfigurations(c *gin.Context) {
	dbID, ok := h.dbID(c)
	if !ok {
		return
	}
	configs, err := h.svc.ListConfigurations(c.Request.Context(), dbID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]backupConfigDTO, 0, len(configs))
	for i := range configs {
		out = append(out, configToDTO(&configs[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *DBBackupHandler) GetConfiguration(c *gin.Context) {
	dbID, ok := h.dbID(c)
	if !ok {
		return
	}
	configID, err := uuid.Parse(c.Param("configID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	cfg, err := h.svc.GetConfiguration(c.Request.Context(), dbID, configID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, configToDTO(cfg))
}

func (h *DBBackupHandler) CreateConfiguration(c *gin.Context) {
	if !h.manage(c) {
		return
	}
	dbID, ok := h.dbID(c)
	if !ok {
		return
	}
	var req backupConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	destID, err := uuid.Parse(req.DestinationID)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	cfg := &domain.BackupConfiguration{
		DatabaseID: dbID, Enabled: req.Enabled, DestinationID: destID, PathPrefix: req.PathPrefix,
		Schedule: domain.Schedule{
			Type: domain.ScheduleType(req.Schedule.Type), Minute: req.Schedule.Minute, At: req.Schedule.At,
			DayOfWeek: req.Schedule.DayOfWeek, StartDate: req.Schedule.StartDate, Cron: req.Schedule.Cron,
			Timezone: req.Schedule.Timezone,
		},
		Retention: domain.Retention{Type: domain.RetentionType(req.Retention.Type)},
	}
	saved, err := h.svc.SaveConfiguration(c.Request.Context(), orgID(c), cfg, true)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, configToDTO(saved))
}

func (h *DBBackupHandler) UpdateConfiguration(c *gin.Context) {
	if !h.manage(c) {
		return
	}
	dbID, ok := h.dbID(c)
	if !ok {
		return
	}
	configID, err := uuid.Parse(c.Param("configID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req backupConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	destID, err := uuid.Parse(req.DestinationID)
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	cfg := &domain.BackupConfiguration{
		ID: configID, DatabaseID: dbID, Enabled: req.Enabled, DestinationID: destID, PathPrefix: req.PathPrefix,
		Schedule:  domain.Schedule{Type: domain.ScheduleType(req.Schedule.Type), Minute: req.Schedule.Minute, At: req.Schedule.At, DayOfWeek: req.Schedule.DayOfWeek, StartDate: req.Schedule.StartDate, Cron: req.Schedule.Cron, Timezone: req.Schedule.Timezone},
		Retention: domain.Retention{Type: domain.RetentionType(req.Retention.Type)},
	}
	saved, err := h.svc.SaveConfiguration(c.Request.Context(), orgID(c), cfg, false)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, configToDTO(saved))
}

func (h *DBBackupHandler) DeleteConfiguration(c *gin.Context) {
	if !h.manage(c) {
		return
	}
	dbID, ok := h.dbID(c)
	if !ok {
		return
	}
	configID, err := uuid.Parse(c.Param("configID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.svc.DeleteConfiguration(c.Request.Context(), dbID, configID, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *DBBackupHandler) CreateBackup(c *gin.Context) {
	if !h.manage(c) {
		return
	}
	dbID, ok := h.dbID(c)
	if !ok {
		return
	}
	var req struct {
		ConfigurationID string `json:"configuration_id"`
	}
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			abort(c, domain.ErrValidation)
			return
		}
	}
	var configID uuid.UUID
	if req.ConfigurationID != "" {
		var err error
		configID, err = uuid.Parse(req.ConfigurationID)
		if err != nil {
			abort(c, domain.ErrValidation)
			return
		}
	}
	job, err := h.svc.StartManualBackup(c.Request.Context(), dbID, orgID(c), configID)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusAccepted, backupJobDTOFrom(*job))
}

func (h *DBBackupHandler) ListBackups(c *gin.Context) {
	dbID, ok := h.dbID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	jobs, err := h.svc.ListBackups(c.Request.Context(), dbID, orgID(c), limit)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]backupJobDTO, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, backupJobDTOFrom(j))
	}
	c.JSON(http.StatusOK, out)
}

func (h *DBBackupHandler) GetBackup(c *gin.Context) {
	backupID, err := uuid.Parse(c.Param("backupID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	job, err := h.svc.GetBackup(c.Request.Context(), backupID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, backupJobDTOFrom(*job))
}

func (h *DBBackupHandler) CancelBackup(c *gin.Context) {
	if !h.manage(c) {
		return
	}
	backupID, err := uuid.Parse(c.Param("backupID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.svc.CancelBackup(c.Request.Context(), backupID, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *DBBackupHandler) PreflightRestore(c *gin.Context) {
	backupID, err := uuid.Parse(c.Param("backupID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	targetID := uuid.Nil
	if rawTarget := c.Query("target_database_id"); rawTarget != "" {
		targetID, err = uuid.Parse(rawTarget)
		if err != nil {
			abort(c, domain.ErrValidation)
			return
		}
	} else if resolved, ok := c.Get("aether.database_spec_id"); ok {
		targetID, _ = resolved.(uuid.UUID)
	} else {
		abort(c, domain.ErrValidation)
		return
	}
	res, err := h.svc.RestorePreflight(c.Request.Context(), backupID, targetID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

type restoreReq struct {
	TargetDatabaseID string `json:"target_database_id"`
}

func (h *DBBackupHandler) RequestRestore(c *gin.Context) {
	if !h.manage(c) {
		return
	}
	backupID, err := uuid.Parse(c.Param("backupID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req restoreReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	targetID := uuid.Nil
	if resolved, ok := c.Get("aether.database_spec_id"); ok {
		targetID, _ = resolved.(uuid.UUID)
	} else {
		targetID, err = uuid.Parse(req.TargetDatabaseID)
		if err != nil {
			abort(c, domain.ErrValidation)
			return
		}
	}
	job, err := h.svc.RequestRestore(c.Request.Context(), backupID, targetID, orgID(c))
	if err != nil {
		if job != nil {
			message := job.ErrorMessage
			if message == "" {
				message = "Restore did not reach a terminal state"
			}
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": message, "restore_job": restoreJobDTOFrom(*job)})
			return
		}
		abort(c, err)
		return
	}
	c.JSON(http.StatusAccepted, restoreJobDTOFrom(*job))
}

func (h *DBBackupHandler) ListRestoreJobs(c *gin.Context) {
	dbID, ok := h.dbID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	jobs, err := h.svc.ListRestoreJobs(c.Request.Context(), dbID, orgID(c), limit)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]restoreJobDTO, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, restoreJobDTOFrom(j))
	}
	c.JSON(http.StatusOK, out)
}

func (h *DBBackupHandler) CreateRestore(c *gin.Context) {
	if !h.manage(c) {
		return
	}
	dbID, ok := h.dbID(c)
	if !ok {
		return
	}
	var req struct {
		Filename string `json:"filename"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	job, err := h.svc.CreateUploadRestore(c.Request.Context(), dbID, orgID(c), req.Filename)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, restoreJobDTOFrom(*job))
}

func (h *DBBackupHandler) UploadRestoreFile(c *gin.Context) {
	if !h.manage(c) {
		return
	}
	dbID, ok := h.dbID(c)
	if !ok {
		return
	}
	restoreID, err := uuid.Parse(c.Param("restoreID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	reader, err := c.Request.MultipartReader()
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	for {
		part, partErr := reader.NextPart()
		if partErr == io.EOF {
			break
		}
		if partErr != nil {
			abort(c, partErr)
			return
		}
		if part.FormName() != "file" {
			continue
		}
		expected := int64(-1)
		if raw := c.GetHeader("X-File-Size"); raw != "" {
			if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed >= 0 {
				expected = parsed
			}
		}
		job, writeErr := h.svc.WriteUpload(c.Request.Context(), dbID, restoreID, orgID(c), part, expected)
		if writeErr != nil {
			if job == nil {
				abort(c, writeErr)
				return
			}
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": writeErr.Error(), "restore_job": restoreJobDTOFrom(*job)})
			return
		}
		c.JSON(http.StatusOK, restoreJobDTOFrom(*job))
		return
	}
	abort(c, domain.ErrValidation)
}

func (h *DBBackupHandler) ValidateUpload(c *gin.Context) {
	if !h.manage(c) {
		return
	}
	dbID, ok := h.dbID(c)
	if !ok {
		return
	}
	restoreID, err := uuid.Parse(c.Param("restoreID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	job, err := h.svc.ValidateUpload(c.Request.Context(), dbID, restoreID, orgID(c))
	if err != nil {
		if job == nil {
			abort(c, err)
			return
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error(), "restore_job": restoreJobDTOFrom(*job)})
		return
	}
	c.JSON(http.StatusOK, restoreJobDTOFrom(*job))
}

func (h *DBBackupHandler) StartUploadRestore(c *gin.Context) {
	if !h.manage(c) {
		return
	}
	dbID, ok := h.dbID(c)
	if !ok {
		return
	}
	restoreID, err := uuid.Parse(c.Param("restoreID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	job, err := h.svc.StartUploadRestore(c.Request.Context(), dbID, restoreID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusAccepted, restoreJobDTOFrom(*job))
}

func (h *DBBackupHandler) GetRestore(c *gin.Context) {
	if !h.manage(c) {
		return
	}
	dbID, ok := h.dbID(c)
	if !ok {
		return
	}
	restoreID, err := uuid.Parse(c.Param("restoreID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	job, err := h.svc.GetRestore(c.Request.Context(), dbID, restoreID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, restoreJobDTOFrom(*job))
}

func (h *DBBackupHandler) DeleteRestore(c *gin.Context) {
	if !h.manage(c) {
		return
	}
	dbID, ok := h.dbID(c)
	if !ok {
		return
	}
	restoreID, err := uuid.Parse(c.Param("restoreID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.svc.CancelUploadRestore(c.Request.Context(), dbID, restoreID, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func configToDTO(cfg *domain.BackupConfiguration) backupConfigDTO {
	var next *string
	if cfg.NextRunAt != nil {
		s := cfg.NextRunAt.UTC().Format(time.RFC3339)
		next = &s
	}
	return backupConfigDTO{
		ID: cfg.ID.String(), DatabaseID: cfg.DatabaseID.String(), ServiceID: cfg.ServiceID.String(), Enabled: cfg.Enabled,
		DestinationID: cfg.DestinationID.String(), PathPrefix: cfg.PathPrefix, NextRunAt: next,
		Schedule: scheduleDTO{
			Type: string(cfg.Schedule.Type), Minute: cfg.Schedule.Minute, At: cfg.Schedule.At,
			DayOfWeek: cfg.Schedule.DayOfWeek, StartDate: cfg.Schedule.StartDate, Cron: cfg.Schedule.Cron,
			Timezone: cfg.Schedule.Timezone,
		},
		Retention: retentionDTO{Type: string(cfg.Retention.Type)},
	}
}

type backupJobDTO struct {
	ID            string  `json:"id"`
	DatabaseID    string  `json:"database_id"`
	ServiceID     string  `json:"service_id"`
	Status        string  `json:"status"`
	Trigger       string  `json:"trigger"`
	Engine        string  `json:"engine"`
	EngineVersion string  `json:"engine_version"`
	Format        string  `json:"format"`
	SizeBytes     int64   `json:"size_bytes"`
	Checksum      string  `json:"checksum"`
	StorageKey    string  `json:"storage_key"`
	ErrorCode     string  `json:"error_code"`
	ErrorMessage  string  `json:"error_message"`
	StartedAt     *string `json:"started_at"`
	CompletedAt   *string `json:"completed_at"`
}

func backupJobDTOFrom(j domain.BackupJob) backupJobDTO {
	return backupJobDTO{
		ID: j.ID.String(), DatabaseID: j.DatabaseID.String(), ServiceID: j.ServiceID.String(), Status: string(j.Status),
		Trigger: string(j.Trigger), Engine: j.Engine, EngineVersion: j.EngineVersion,
		Format: j.Format, SizeBytes: j.SizeBytes, Checksum: j.Checksum, StorageKey: j.StorageKey,
		ErrorCode: j.ErrorCode, ErrorMessage: j.ErrorMessage,
		StartedAt: timePtrStr(j.StartedAt), CompletedAt: timePtrStr(j.CompletedAt),
	}
}

type restoreJobDTO struct {
	ID               string  `json:"id"`
	BackupID         string  `json:"backup_id"`
	TargetDatabaseID string  `json:"target_database_id"`
	ServiceID        string  `json:"service_id"`
	Status           string  `json:"status"`
	ErrorCode        string  `json:"error_code"`
	ErrorMessage     string  `json:"error_message"`
	StartedAt        *string `json:"started_at"`
	CompletedAt      *string `json:"completed_at"`
	SourceType       string  `json:"source_type"`
	SourceFilename   string  `json:"source_filename"`
	SourceSize       int64   `json:"source_size"`
	SourceChecksum   string  `json:"source_checksum"`
	SourceFormat     string  `json:"source_format"`
	UploadedBytes    int64   `json:"uploaded_bytes"`
}

func restoreJobDTOFrom(j domain.RestoreJob) restoreJobDTO {
	var backupID string
	if j.BackupID != nil {
		backupID = j.BackupID.String()
	}
	return restoreJobDTO{
		ID: j.ID.String(), BackupID: backupID, TargetDatabaseID: j.TargetDatabaseID.String(), ServiceID: j.ServiceID.String(),
		Status: string(j.Status), ErrorCode: j.ErrorCode, ErrorMessage: j.ErrorMessage,
		StartedAt: timePtrStr(j.StartedAt), CompletedAt: timePtrStr(j.CompletedAt),
		SourceType: string(j.SourceType), SourceFilename: j.SourceFilename,
		SourceSize: j.SourceSize, SourceChecksum: j.SourceChecksum, SourceFormat: j.SourceFormat,
		UploadedBytes: j.UploadedBytes,
	}
}

func timePtrStr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}
