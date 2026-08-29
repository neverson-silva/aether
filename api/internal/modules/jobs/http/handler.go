package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authhttp "aether/internal/modules/auth/http"
	"aether/internal/modules/jobs/application"
	"aether/internal/modules/jobs/domain"
)

type Handler struct {
	jobs *application.Jobs
}

func New(jobs *application.Jobs) *Handler {
	return &Handler{jobs: jobs}
}

type cronReq struct {
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
	Command  string `json:"command"`
}

type cronUpdateReq struct {
	Schedule *string `json:"schedule"`
	Command  *string `json:"command"`
	Enabled  *bool   `json:"enabled"`
}

type workerReq struct {
	Name     string `json:"name"`
	Command  string `json:"command"`
	Replicas *int   `json:"replicas"`
}

type policyReq struct {
	Enabled      bool    `json:"enabled"`
	CPUMin       float64 `json:"cpu_min"`
	CPUMax       float64 `json:"cpu_max"`
	MemMinMB     int     `json:"mem_min_mb"`
	MemMaxMB     int     `json:"mem_max_mb"`
	ScaleUpPct   int     `json:"scale_up_pct"`
	ScaleDownPct int     `json:"scale_down_pct"`
	CooldownMin  int     `json:"cooldown_min"`
}

func (h *Handler) CreateCronJob(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req cronReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	job, err := h.jobs.CreateCronJob(c.Request.Context(), appID, orgID(c), req.Name, req.Schedule, req.Command)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, cronDTO(job))
}

func (h *Handler) ListCronJobs(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	jobs, err := h.jobs.ListCronJobs(c.Request.Context(), appID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(jobs))
	for i := range jobs {
		out = append(out, cronDTO(&jobs[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) ListAllCronJobs(c *gin.Context) {
	jobs, err := h.jobs.ListAllCronJobs(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(jobs))
	for i := range jobs {
		out = append(out, cronDTO(&jobs[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) UpdateCronJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("jobID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req cronUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	job, err := h.jobs.UpdateCronJob(c.Request.Context(), id, orgID(c), req.Schedule, req.Command, req.Enabled)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, cronDTO(job))
}

func (h *Handler) DeleteCronJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("jobID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.jobs.DeleteCronJob(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) CreateWorker(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req workerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	replicas := 1
	if req.Replicas != nil {
		replicas = *req.Replicas
	}
	worker, err := h.jobs.CreateWorker(c.Request.Context(), appID, orgID(c), req.Name, req.Command, replicas)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, workerDTO(worker))
}

func (h *Handler) ListWorkers(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	workers, err := h.jobs.ListWorkers(c.Request.Context(), appID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(workers))
	for i := range workers {
		out = append(out, workerDTO(&workers[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) StartWorker(c *gin.Context) {
	id, err := uuid.Parse(c.Param("workerID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.jobs.StartWorker(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "started"})
}

func (h *Handler) StopWorker(c *gin.Context) {
	id, err := uuid.Parse(c.Param("workerID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.jobs.StopWorker(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

func (h *Handler) DeleteWorker(c *gin.Context) {
	id, err := uuid.Parse(c.Param("workerID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.jobs.DeleteWorker(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) GetPolicy(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	policy, err := h.jobs.GetPolicy(c.Request.Context(), appID, orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, policyDTO(policy))
}

func (h *Handler) SavePolicy(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req policyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	policy, err := h.jobs.SavePolicy(c.Request.Context(), appID, orgID(c), &domain.Policy{
		Enabled: req.Enabled, CPUMin: req.CPUMin, CPUMax: req.CPUMax,
		MemMinMB: req.MemMinMB, MemMaxMB: req.MemMaxMB, ScaleUpPct: req.ScaleUpPct,
		ScaleDownPct: req.ScaleDownPct, CooldownMin: req.CooldownMin,
	})
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, policyDTO(policy))
}

func (h *Handler) PolicyEvents(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	events, err := h.jobs.PolicyEvents(c.Request.Context(), appID, orgID(c), 50)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(events))
	for i := range events {
		out = append(out, gin.H{
			"id": events[i].ID, "app_id": events[i].AppID, "action": events[i].Action,
			"detail": events[i].Detail, "created_at": events[i].CreatedAt,
		})
	}
	c.JSON(http.StatusOK, out)
}

func cronDTO(job *domain.CronJob) gin.H {
	return gin.H{
		"id": job.ID, "app_id": job.AppID, "service_id": job.ServiceID, "service_name": job.ServiceName, "app_name": job.ServiceName, "name": job.Name, "schedule": job.Schedule,
		"command": job.Command, "enabled": job.Enabled, "last_run": job.LastRun,
		"next_run": job.NextRun, "created_at": job.CreatedAt,
	}
}

func workerDTO(worker *domain.Worker) gin.H {
	return gin.H{
		"id": worker.ID, "app_id": worker.AppID, "name": worker.Name, "command": worker.Command,
		"replicas": worker.Replicas, "enabled": worker.Enabled, "status": worker.Status,
		"container_id": worker.ContainerID, "created_at": worker.CreatedAt,
	}
}

func policyDTO(p *domain.Policy) gin.H {
	return gin.H{
		"app_id": p.AppID, "enabled": p.Enabled, "cpu_min": p.CPUMin, "cpu_max": p.CPUMax,
		"mem_min_mb": p.MemMinMB, "mem_max_mb": p.MemMaxMB, "scale_up_pct": p.ScaleUpPct,
		"scale_down_pct": p.ScaleDownPct, "cooldown_min": p.CooldownMin, "updated_at": p.UpdatedAt,
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
