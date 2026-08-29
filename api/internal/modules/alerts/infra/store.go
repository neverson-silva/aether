package infra

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"aether/internal/modules/alerts/domain"
	gen "aether/internal/platform/infrastructure/pg/gen"
)

type Store struct {
	q  gen.Querier
	db *sql.DB
}

func NewStore(pool *pgxpool.Pool) *Store {
	db := stdlib.OpenDBFromPool(pool)
	return &Store{q: gen.New(db), db: db}
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) ListAlertRules(ctx context.Context, orgID uuid.UUID) ([]domain.AlertRule, error) {
	rows, err := s.q.ListAlertRules(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.AlertRule, 0, len(rows))
	for _, r := range rows {
		out = append(out, *ruleFromRow(r))
	}
	return out, nil
}

func (s *Store) GetAlertRule(ctx context.Context, id uuid.UUID) (*domain.AlertRule, error) {
	row, err := s.q.GetAlertRule(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return ruleFromRow(row), nil
}

func (s *Store) CreateAlertRule(ctx context.Context, rule *domain.AlertRule) (*domain.AlertRule, error) {
	row, err := s.q.CreateAlertRule(ctx, gen.CreateAlertRuleParams{
		OrgID: rule.OrgID, Name: rule.Name, Metric: rule.Metric,
		Threshold: float32(rule.Threshold), WindowS: int32(rule.WindowS), Severity: rule.Severity,
		TargetApp: nullUUID(rule.TargetApp),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return ruleFromRow(row), nil
}

func (s *Store) SetAlertRuleEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	return mapErr(s.q.SetAlertRuleEnabled(ctx, gen.SetAlertRuleEnabledParams{ID: id, Enabled: enabled}))
}

func (s *Store) DeleteAlertRule(ctx context.Context, id, orgID uuid.UUID) error {
	return mapErr(s.q.DeleteAlertRule(ctx, gen.DeleteAlertRuleParams{ID: id, OrgID: orgID}))
}

func (s *Store) CreateAlertEvent(ctx context.Context, event *domain.AlertEvent) (*domain.AlertEvent, error) {
	row, err := s.q.CreateAlertEvent(ctx, gen.CreateAlertEventParams{
		OrgID: event.OrgID, RuleID: nullUUID(event.RuleID), AppID: nullUUID(event.AppID),
		AppName: event.AppName, Severity: event.Severity, Message: event.Message,
		Value: float32(event.Value), Threshold: float32(event.Threshold), Metric: event.Metric,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return eventFromRow(row), nil
}

func (s *Store) ListAlertEventsByOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]domain.AlertEvent, error) {
	rows, err := s.q.ListAlertEventsByOrg(ctx, gen.ListAlertEventsByOrgParams{OrgID: orgID, Limit: int32(limit)})
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.AlertEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, *eventFromRow(r))
	}
	return out, nil
}

func (s *Store) ResolveAlertEvent(ctx context.Context, id uuid.UUID) error {
	return mapErr(s.q.ResolveAlertEvent(ctx, id))
}

func (s *Store) CreateNotification(ctx context.Context, notification *domain.Notification) (*domain.Notification, error) {
	row, err := s.q.CreateNotification(ctx, gen.CreateNotificationParams{
		OrgID: notification.OrgID, Type: notification.Type, Message: notification.Message, Payload: notification.Payload,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return notificationFromRow(row), nil
}

func (s *Store) ListNotifications(ctx context.Context, orgID uuid.UUID, limit int) ([]domain.Notification, error) {
	rows, err := s.q.ListNotifications(ctx, gen.ListNotificationsParams{OrgID: orgID, Limit: int32(limit)})
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Notification, 0, len(rows))
	for _, r := range rows {
		out = append(out, *notificationFromRow(r))
	}
	return out, nil
}

func (s *Store) CountUnread(ctx context.Context, orgID uuid.UUID) (int, error) {
	n, err := s.q.CountUnreadNotifications(ctx, orgID)
	if err != nil {
		return 0, mapErr(err)
	}
	return int(n), nil
}

func (s *Store) MarkRead(ctx context.Context, id, orgID uuid.UUID) error {
	return mapErr(s.q.MarkNotificationRead(ctx, gen.MarkNotificationReadParams{ID: id, OrgID: orgID}))
}

func (s *Store) MarkAllRead(ctx context.Context, orgID uuid.UUID) error {
	return mapErr(s.q.MarkAllNotificationsRead(ctx, orgID))
}

func (s *Store) CreateChannel(ctx context.Context, channel *domain.Channel) (*domain.Channel, error) {
	row, err := s.q.CreateChannel(ctx, gen.CreateChannelParams{
		OrgID: channel.OrgID, Name: channel.Name, Type: channel.Type, ConfigEnc: channel.ConfigEnc,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return channelFromRow(row), nil
}

func (s *Store) ListChannelsByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.Channel, error) {
	rows, err := s.q.ListChannelsByOrg(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Channel, 0, len(rows))
	for _, r := range rows {
		out = append(out, *channelFromRow(r))
	}
	return out, nil
}

func (s *Store) DeleteChannel(ctx context.Context, id, orgID uuid.UUID) error {
	return mapErr(s.q.DeleteChannel(ctx, gen.DeleteChannelParams{ID: id, OrgID: orgID}))
}

func ruleFromRow(row gen.AlertRule) *domain.AlertRule {
	return &domain.AlertRule{
		ID: row.ID, OrgID: row.OrgID, Name: row.Name, Metric: row.Metric,
		Threshold: float64(row.Threshold), WindowS: int(row.WindowS), Severity: row.Severity,
		Enabled: row.Enabled, TargetApp: uuidPtr(row.TargetApp), ServiceID: uuidPtr(row.ServiceID), CreatedAt: row.CreatedAt,
	}
}

func eventFromRow(row gen.AlertEvent) *domain.AlertEvent {
	return &domain.AlertEvent{
		ID: row.ID, OrgID: row.OrgID, RuleID: uuidPtr(row.RuleID), AppID: uuidPtr(row.AppID), ServiceID: uuidPtr(row.ServiceID),
		AppName: row.AppName, Severity: row.Severity, Message: row.Message,
		Value: float64(row.Value), Threshold: float64(row.Threshold), Metric: row.Metric,
		CreatedAt: row.CreatedAt, ResolvedAt: nullTimePtr(row.ResolvedAt),
	}
}

func notificationFromRow(row gen.Notification) *domain.Notification {
	return &domain.Notification{
		ID: row.ID, OrgID: row.OrgID, Type: row.Type, Message: row.Message,
		Payload: row.Payload, Read: row.Read, CreatedAt: row.CreatedAt,
	}
}

func channelFromRow(row gen.NotificationChannel) *domain.Channel {
	return &domain.Channel{
		ID: row.ID, OrgID: row.OrgID, Name: row.Name, Type: row.Type,
		ConfigEnc: row.ConfigEnc, Enabled: row.Enabled, CreatedAt: row.CreatedAt,
	}
}

func nullUUID(v *uuid.UUID) uuid.NullUUID {
	if v == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *v, Valid: true}
}

func uuidPtr(v uuid.NullUUID) *uuid.UUID {
	if !v.Valid {
		return nil
	}
	return &v.UUID
}

func nullTimePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return domain.ErrConflict
		case "23503":
			return domain.ErrConflict
		case "23502", "22P02", "23514":
			return domain.ErrValidation
		}
	}
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}
