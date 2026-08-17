package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"aether/internal/alerts/application"
	"aether/internal/alerts/domain"
	authhttp "aether/internal/auth/http"
)

type Handler struct {
	alerts        *application.Alerts
	notifications *application.Notifications
	channels      *application.Channels
}

func New(alerts *application.Alerts, notifications *application.Notifications, channels *application.Channels) *Handler {
	return &Handler{alerts: alerts, notifications: notifications, channels: channels}
}

type alertRuleReq struct {
	Name      string  `json:"name"`
	Metric    string  `json:"metric"`
	Threshold float64 `json:"threshold"`
	WindowS   int     `json:"window_s"`
	Severity  string  `json:"severity"`
	TargetApp string  `json:"target_app"`
}

type enabledReq struct {
	Enabled *bool `json:"enabled"`
}

type channelReq struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Config string `json:"config"`
}

func (h *Handler) ListRules(c *gin.Context) {
	rules, err := h.alerts.ListRules(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(rules))
	for i := range rules {
		out = append(out, ruleDTO(&rules[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) CreateRule(c *gin.Context) {
	var req alertRuleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var target *uuid.UUID
	if req.TargetApp != "" {
		id, err := uuid.Parse(req.TargetApp)
		if err != nil {
			abort(c, domain.ErrValidation)
			return
		}
		target = &id
	}
	rule, err := h.alerts.CreateRule(c.Request.Context(), orgID(c), &domain.AlertRule{
		Name: req.Name, Metric: req.Metric, Threshold: req.Threshold, WindowS: req.WindowS,
		Severity: req.Severity, TargetApp: target,
	})
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, ruleDTO(rule))
}

func (h *Handler) PatchRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("ruleID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	var req enabledReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if req.Enabled != nil {
		if err := h.alerts.SetEnabled(c.Request.Context(), id, orgID(c), *req.Enabled); err != nil {
			abort(c, err)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) DeleteRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("ruleID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.alerts.DeleteRule(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ListEvents(c *gin.Context) {
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	events, err := h.alerts.ListEvents(c.Request.Context(), orgID(c), limit)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(events))
	for i := range events {
		out = append(out, eventDTO(&events[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) ResolveEvent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("eventID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.alerts.ResolveEvent(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ListNotifications(c *gin.Context) {
	list, err := h.notifications.List(c.Request.Context(), orgID(c), 50)
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(list))
	for i := range list {
		out = append(out, notificationDTO(&list[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	n, err := h.notifications.UnreadCount(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": n})
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, err := uuid.Parse(c.Param("notifID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.notifications.MarkRead(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.notifications.MarkAllRead(c.Request.Context(), orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) CreateChannel(c *gin.Context) {
	var req channelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	channel, err := h.channels.Create(c.Request.Context(), orgID(c), req.Name, req.Type, req.Config)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, channelDTO(channel))
}

func (h *Handler) ListChannels(c *gin.Context) {
	channels, err := h.channels.List(c.Request.Context(), orgID(c))
	if err != nil {
		abort(c, err)
		return
	}
	out := make([]gin.H, 0, len(channels))
	for i := range channels {
		out = append(out, channelDTO(&channels[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) DeleteChannel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("channelID"))
	if err != nil {
		abort(c, domain.ErrValidation)
		return
	}
	if err := h.channels.Delete(c.Request.Context(), id, orgID(c)); err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func ruleDTO(r *domain.AlertRule) gin.H {
	return gin.H{
		"id": r.ID, "org_id": r.OrgID, "name": r.Name, "metric": r.Metric,
		"threshold": r.Threshold, "window_s": r.WindowS, "severity": r.Severity,
		"enabled": r.Enabled, "target_app": r.TargetApp, "created_at": r.CreatedAt,
	}
}

func eventDTO(e *domain.AlertEvent) gin.H {
	return gin.H{
		"id": e.ID, "org_id": e.OrgID, "rule_id": e.RuleID, "app_id": e.AppID,
		"app_name": e.AppName, "severity": e.Severity, "message": e.Message,
		"value": e.Value, "threshold": e.Threshold, "metric": e.Metric,
		"created_at": e.CreatedAt, "resolved_at": e.ResolvedAt,
	}
}

func notificationDTO(n *domain.Notification) gin.H {
	return gin.H{
		"id": n.ID, "org_id": n.OrgID, "type": n.Type, "message": n.Message,
		"payload": n.Payload, "read": n.Read, "created_at": n.CreatedAt,
	}
}

func channelDTO(ch *domain.Channel) gin.H {
	return gin.H{
		"id": ch.ID, "org_id": ch.OrgID, "name": ch.Name, "type": ch.Type,
		"enabled": ch.Enabled, "created_at": ch.CreatedAt,
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
