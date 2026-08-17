package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"aether/internal/alerts/domain"
	"aether/internal/alerts/infra"
)

type env struct {
	ctx    context.Context
	alerts *Alerts
	notifs *Notifications
	chans  *Channels
	orgID  uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	cipher, err := infra.NewPasswordCipher([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	e := &env{
		ctx: context.Background(), alerts: &Alerts{Store: store},
		notifs: &Notifications{Store: store}, chans: &Channels{Store: store, Keys: cipher},
		orgID: uuid.New(),
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, e.orgID, "Alerts Org", "alerts-org"); err != nil {
		t.Fatalf("criar org: %v", err)
	}
	return e
}

func TestAlertRuleLifecycle(t *testing.T) {
	e := newEnv(t)
	rule, err := e.alerts.CreateRule(e.ctx, e.orgID, &domain.AlertRule{
		Name: "high-cpu", Metric: "cpu", Threshold: 90, WindowS: 60, Severity: "critical",
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	if !rule.Enabled || rule.Severity != "critical" {
		t.Fatalf("rule inesperada: %+v", rule)
	}

	if _, err := e.alerts.CreateRule(e.ctx, e.orgID, &domain.AlertRule{
		Name: "bad", Metric: "not-a-metric", Threshold: 1,
	}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("metric inválida deveria falhar: %v", err)
	}

	disabled := false
	if err := e.alerts.SetEnabled(e.ctx, rule.ID, e.orgID, disabled); err != nil {
		t.Fatalf("set enabled: %v", err)
	}
	rules, _ := e.alerts.ListRules(e.ctx, e.orgID)
	if rules[0].Enabled {
		t.Fatalf("rule deveria estar desabilitada")
	}

	if err := e.alerts.DeleteRule(e.ctx, rule.ID, e.orgID); err != nil {
		t.Fatalf("delete rule: %v", err)
	}
}

func TestAlertEventAndResolve(t *testing.T) {
	e := newEnv(t)
	event, err := e.alerts.CreateEvent(e.ctx, &domain.AlertEvent{
		OrgID: e.orgID, Severity: "critical", Message: "cpu alto", Value: 95, Threshold: 90, Metric: "cpu",
	})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	if err := e.alerts.ResolveEvent(e.ctx, event.ID, e.orgID); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	events, _ := e.alerts.ListEvents(e.ctx, e.orgID, 50)
	if events[0].ResolvedAt == nil {
		t.Fatalf("event deveria estar resolvido")
	}
	if err := e.alerts.ResolveEvent(e.ctx, uuid.New(), e.orgID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("event inexistente deveria falhar: %v", err)
	}
}

func TestNotifications(t *testing.T) {
	e := newEnv(t)
	for _, m := range []string{"deploy completed", "alerta crítico"} {
		if _, err := e.notifs.Create(e.ctx, &domain.Notification{OrgID: e.orgID, Type: "deploy", Message: m}); err != nil {
			t.Fatalf("create notif: %v", err)
		}
	}
	count, err := e.notifs.UnreadCount(e.ctx, e.orgID)
	if err != nil || count != 2 {
		t.Fatalf("unread: %v %d", err, count)
	}
	list, _ := e.notifs.List(e.ctx, e.orgID, 10)
	if err := e.notifs.MarkRead(e.ctx, list[0].ID, e.orgID); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	count, _ = e.notifs.UnreadCount(e.ctx, e.orgID)
	if count != 1 {
		t.Fatalf("esperava 1 unread, got %d", count)
	}
	if err := e.notifs.MarkAllRead(e.ctx, e.orgID); err != nil {
		t.Fatalf("mark all: %v", err)
	}
	count, _ = e.notifs.UnreadCount(e.ctx, e.orgID)
	if count != 0 {
		t.Fatalf("esperava 0 unread, got %d", count)
	}
}

func TestChannels(t *testing.T) {
	e := newEnv(t)
	ch, err := e.chans.Create(e.ctx, e.orgID, "ops-slack", "slack", "https://hooks.slack.com/x")
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}
	if ch.ConfigEnc == "" || ch.ConfigEnc == "https://hooks.slack.com/x" {
		t.Fatalf("config deveria estar encriptada")
	}
	if _, err := e.chans.Create(e.ctx, e.orgID, "bad", "sms", "x"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("tipo inválido deveria falhar: %v", err)
	}
	channels, err := e.chans.List(e.ctx, e.orgID)
	if err != nil || len(channels) != 1 {
		t.Fatalf("list channels: %v %d", err, len(channels))
	}
	if err := e.chans.Delete(e.ctx, ch.ID, e.orgID); err != nil {
		t.Fatalf("delete channel: %v", err)
	}
}
