package application

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"aether/internal/modules/alerts/domain"
)

type Alerts struct {
	Store domain.Store
}

var metrics = map[string]bool{
	"cpu": true, "memory": true, "disk": true, "network_rx": true, "network_tx": true, "requests": true,
}

var severities = map[string]bool{
	"info": true, "warning": true, "critical": true,
}

func (a *Alerts) ListRules(ctx context.Context, orgID uuid.UUID) ([]domain.AlertRule, error) {
	return a.Store.ListAlertRules(ctx, orgID)
}

func (a *Alerts) CreateRule(ctx context.Context, orgID uuid.UUID, rule *domain.AlertRule) (*domain.AlertRule, error) {
	rule.OrgID = orgID
	rule.Name = strings.TrimSpace(rule.Name)
	if rule.Name == "" {
		return nil, domain.ErrValidation
	}
	if !metrics[rule.Metric] {
		return nil, domain.ErrValidation
	}
	if rule.Threshold < 0 {
		return nil, domain.ErrValidation
	}
	if rule.WindowS <= 0 {
		rule.WindowS = 30
	}
	if rule.Severity == "" {
		rule.Severity = "warning"
	}
	if !severities[rule.Severity] {
		return nil, domain.ErrValidation
	}
	rule.Enabled = true
	return a.Store.CreateAlertRule(ctx, rule)
}

func (a *Alerts) SetEnabled(ctx context.Context, ruleID, orgID uuid.UUID, enabled bool) error {
	rule, err := a.Store.GetAlertRule(ctx, ruleID)
	if err != nil {
		return err
	}
	if rule.OrgID != orgID {
		return domain.ErrNotFound
	}
	return a.Store.SetAlertRuleEnabled(ctx, ruleID, enabled)
}

func (a *Alerts) DeleteRule(ctx context.Context, ruleID, orgID uuid.UUID) error {
	return a.Store.DeleteAlertRule(ctx, ruleID, orgID)
}

func (a *Alerts) ListEvents(ctx context.Context, orgID uuid.UUID, limit int) ([]domain.AlertEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return a.Store.ListAlertEventsByOrg(ctx, orgID, limit)
}

func (a *Alerts) ResolveEvent(ctx context.Context, eventID, orgID uuid.UUID) error {
	events, err := a.Store.ListAlertEventsByOrg(ctx, orgID, 200)
	if err != nil {
		return err
	}
	for i := range events {
		if events[i].ID == eventID {
			return a.Store.ResolveAlertEvent(ctx, eventID)
		}
	}
	return domain.ErrNotFound
}

func (a *Alerts) CreateEvent(ctx context.Context, event *domain.AlertEvent) (*domain.AlertEvent, error) {
	return a.Store.CreateAlertEvent(ctx, event)
}

type Notifications struct {
	Store domain.Store
}

func (n *Notifications) List(ctx context.Context, orgID uuid.UUID, limit int) ([]domain.Notification, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return n.Store.ListNotifications(ctx, orgID, limit)
}

func (n *Notifications) UnreadCount(ctx context.Context, orgID uuid.UUID) (int, error) {
	return n.Store.CountUnread(ctx, orgID)
}

func (n *Notifications) MarkRead(ctx context.Context, id, orgID uuid.UUID) error {
	return n.Store.MarkRead(ctx, id, orgID)
}

func (n *Notifications) MarkAllRead(ctx context.Context, orgID uuid.UUID) error {
	return n.Store.MarkAllRead(ctx, orgID)
}

func (n *Notifications) Create(ctx context.Context, notification *domain.Notification) (*domain.Notification, error) {
	return n.Store.CreateNotification(ctx, notification)
}

type Channels struct {
	Store domain.Store
	Keys  domain.PasswordCipher
}

var channelTypes = map[string]bool{
	"email": true, "webhook": true, "slack": true, "telegram": true,
}

func (c *Channels) Create(ctx context.Context, orgID uuid.UUID, name, channelType, config string) (*domain.Channel, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrValidation
	}
	if !channelTypes[channelType] {
		return nil, domain.ErrValidation
	}
	enc, err := c.Keys.Encrypt(config)
	if err != nil {
		return nil, err
	}
	return c.Store.CreateChannel(ctx, &domain.Channel{OrgID: orgID, Name: name, Type: channelType, ConfigEnc: enc, Enabled: true})
}

func (c *Channels) List(ctx context.Context, orgID uuid.UUID) ([]domain.Channel, error) {
	return c.Store.ListChannelsByOrg(ctx, orgID)
}

func (c *Channels) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	return c.Store.DeleteChannel(ctx, id, orgID)
}
