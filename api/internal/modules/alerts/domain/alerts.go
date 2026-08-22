package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict")
	ErrValidation = errors.New("invalid input")
	ErrForbidden  = errors.New("access denied")
)

type AlertRule struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	Name      string
	Metric    string
	Threshold float64
	WindowS   int
	Severity  string
	Enabled   bool
	TargetApp *uuid.UUID
	CreatedAt time.Time
}

type AlertEvent struct {
	ID         uuid.UUID
	OrgID      uuid.UUID
	RuleID     *uuid.UUID
	AppID      *uuid.UUID
	AppName    string
	Severity   string
	Message    string
	Value      float64
	Threshold  float64
	Metric     string
	CreatedAt  time.Time
	ResolvedAt *time.Time
}

type Notification struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	Type      string
	Message   string
	Payload   string
	Read      bool
	CreatedAt time.Time
}

type Channel struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	Name      string
	Type      string
	ConfigEnc string
	Enabled   bool
	CreatedAt time.Time
}

type PasswordCipher interface {
	Encrypt(plain string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type Store interface {
	ListAlertRules(ctx context.Context, orgID uuid.UUID) ([]AlertRule, error)
	GetAlertRule(ctx context.Context, id uuid.UUID) (*AlertRule, error)
	CreateAlertRule(ctx context.Context, rule *AlertRule) (*AlertRule, error)
	SetAlertRuleEnabled(ctx context.Context, id uuid.UUID, enabled bool) error
	DeleteAlertRule(ctx context.Context, id, orgID uuid.UUID) error

	CreateAlertEvent(ctx context.Context, event *AlertEvent) (*AlertEvent, error)
	ListAlertEventsByOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]AlertEvent, error)
	ResolveAlertEvent(ctx context.Context, id uuid.UUID) error

	CreateNotification(ctx context.Context, notification *Notification) (*Notification, error)
	ListNotifications(ctx context.Context, orgID uuid.UUID, limit int) ([]Notification, error)
	CountUnread(ctx context.Context, orgID uuid.UUID) (int, error)
	MarkRead(ctx context.Context, id, orgID uuid.UUID) error
	MarkAllRead(ctx context.Context, orgID uuid.UUID) error

	CreateChannel(ctx context.Context, channel *Channel) (*Channel, error)
	ListChannelsByOrg(ctx context.Context, orgID uuid.UUID) ([]Channel, error)
	DeleteChannel(ctx context.Context, id, orgID uuid.UUID) error
}
