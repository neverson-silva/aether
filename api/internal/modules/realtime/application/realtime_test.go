package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	appsdomain "aether/internal/modules/apps/domain"
	deploydomain "aether/internal/modules/deployments/domain"
	"aether/internal/modules/realtime/domain"
	"aether/internal/platform/druntime"
	rtmemory "aether/internal/platform/druntime/adapter/memory"
)

type fakeAppStore struct{}

func (fakeAppStore) GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error) {
	return nil, errors.New("not found")
}

func (fakeAppStore) GetAppByID(ctx context.Context, id uuid.UUID) (*appsdomain.App, error) {
	return nil, errors.New("not found")
}

func (fakeAppStore) ListAppsByOrg(ctx context.Context, orgID uuid.UUID) ([]appsdomain.App, error) {
	return nil, nil
}

type fakeDeploymentStore struct{}

func (fakeDeploymentStore) ListByApp(ctx context.Context, appID uuid.UUID, limit int) ([]deploydomain.Deployment, error) {
	return nil, nil
}

func (fakeDeploymentStore) GetDeployment(ctx context.Context, id uuid.UUID) (*deploydomain.Deployment, error) {
	return nil, errors.New("not found")
}

type fakePortReader struct{}

func (fakePortReader) Port(ctx context.Context, containerID string) (string, error) {
	return "", errors.New("no port")
}

func newRealtime() *Realtime {
	rt, err := rtmemory.New(context.Background(), druntime.Config{Backend: "memory"})
	if err != nil {
		panic(err)
	}
	return &Realtime{
		Presence: rt.Presence, PubSub: rt.PubSub,
		Apps: fakeAppStore{}, Deployments: fakeDeploymentStore{}, Ports: fakePortReader{},
	}
}

func TestPresenceFlow(t *testing.T) {
	rt := newRealtime()
	ctx := context.Background()
	if err := rt.Join(ctx, "app:1", "user-1"); err != nil {
		t.Fatalf("join: %v", err)
	}
	if err := rt.Heartbeat(ctx, "app:1", "user-1"); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	count, members, err := rt.Count(ctx, "app:1")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 || len(members) != 1 || members[0] != "user-1" {
		t.Fatalf("count/members: %d %v", count, members)
	}
	if err := rt.Leave(ctx, "app:1", "user-1"); err != nil {
		t.Fatalf("leave: %v", err)
	}
	count, _, _ = rt.Count(ctx, "app:1")
	if count != 0 {
		t.Fatalf("count após leave: %d", count)
	}
}

func TestPresenceValidation(t *testing.T) {
	rt := newRealtime()
	if err := rt.Join(context.Background(), "", "user-1"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("scope vazio: %v", err)
	}
	if _, _, err := rt.Count(context.Background(), ""); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("count scope vazio: %v", err)
	}
}

func TestPublishRecentEvents(t *testing.T) {
	rt := newRealtime()
	org := uuid.New()
	if err := rt.PublishEvent(context.Background(), org, domain.Event{Type: "deploy.created", Aggregate: "deployment"}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	events, err := rt.RecentEvents(context.Background(), org, 10)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(events) != 1 || events[0].Type != "deploy.created" {
		t.Fatalf("events: %+v", events)
	}
}

func TestSubscribeEvents(t *testing.T) {
	rt := newRealtime()
	org := uuid.New()
	received := make(chan domain.Event, 1)
	sub, err := rt.SubscribeEvents(context.Background(), org, func(event domain.Event) {
		received <- event
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	if err := rt.PublishEvent(context.Background(), org, domain.Event{Type: "app.started"}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	select {
	case event := <-received:
		if event.Type != "app.started" {
			t.Fatalf("event: %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("event não recebido")
	}
}

func TestMetrics(t *testing.T) {
	rt := newRealtime()
	metrics := rt.Metrics(context.Background())
	if metrics.Backend != "memory" {
		t.Fatalf("backend: %s", metrics.Backend)
	}
}

func TestReadyContainer(t *testing.T) {
	rt := newRealtime()
	if _, err := rt.ReadyContainer(context.Background(), uuid.New(), uuid.New()); err == nil {
		t.Fatalf("deveria falhar sem app")
	}
}

func TestNotifyDeployPublishesEvent(t *testing.T) {
	rt := newRealtime()
	org := uuid.New()
	appID := uuid.New()
	rt.Apps = fakeAppStoreWith{app: &appsdomain.App{ID: appID, OrgID: org}}
	received := make(chan domain.Event, 1)
	sub, err := rt.SubscribeEvents(context.Background(), org, func(event domain.Event) {
		received <- event
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	rt.NotifyDeploy(context.Background(), deploydomain.DeployEvent{AppID: appID, DepID: uuid.New(), Status: "ready"})
	select {
	case event := <-received:
		if event.Type != "deploy.ready" {
			t.Fatalf("event: %+v", event)
		}
		if event.AppID != appID.String() || event.CorrelationID == "" || event.Seq == 0 {
			t.Fatalf("envelope incompleto: %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("event não recebido")
	}
}

func TestAuthorizeScopes(t *testing.T) {
	org := uuid.New()
	otherOrg := uuid.New()
	appID := uuid.New()
	depID := uuid.New()
	rt := newRealtime()
	rt.Apps = fakeAppStoreWith{app: &appsdomain.App{ID: appID, OrgID: org}}
	rt.Deployments = fakeDeploymentStoreWith{dep: &deploydomain.Deployment{ID: depID, AppID: appID}}
	ctx := context.Background()

	if err := rt.Authorize(ctx, "org", org); err != nil {
		t.Fatalf("org deveria passar: %v", err)
	}
	if err := rt.Authorize(ctx, "app:"+appID.String(), org); err != nil {
		t.Fatalf("app da org deveria passar: %v", err)
	}
	if err := rt.Authorize(ctx, "deployment:"+depID.String(), org); err != nil {
		t.Fatalf("deployment da org deveria passar: %v", err)
	}
	if err := rt.Authorize(ctx, "app:"+appID.String(), otherOrg); err == nil {
		t.Fatalf("app de outra org deveria falhar")
	}
	if err := rt.Authorize(ctx, "deployment:"+depID.String(), otherOrg); err == nil {
		t.Fatalf("deployment de outra org deveria falhar")
	}
	if err := rt.Authorize(ctx, "app:not-a-uuid", org); err == nil {
		t.Fatalf("scope inválido deveria falhar")
	}
	if err := rt.Authorize(ctx, "server:1", org); err == nil {
		t.Fatalf("scope desconhecido deveria falhar")
	}
	if err := rt.Authorize(ctx, "deployment:"+uuid.New().String(), org); err == nil {
		t.Fatalf("deployment inexistente deveria falhar")
	}
}

type fakeDeploymentStoreWith struct {
	dep *deploydomain.Deployment
}

func (f fakeDeploymentStoreWith) ListByApp(ctx context.Context, appID uuid.UUID, limit int) ([]deploydomain.Deployment, error) {
	return nil, nil
}

func (f fakeDeploymentStoreWith) GetDeployment(ctx context.Context, id uuid.UUID) (*deploydomain.Deployment, error) {
	if f.dep != nil && f.dep.ID == id {
		return f.dep, nil
	}
	return nil, errors.New("not found")
}

type fakeAppStoreWith struct {
	app *appsdomain.App
}

func (f fakeAppStoreWith) GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error) {
	if f.app.ID == id && f.app.OrgID == orgID {
		return f.app, nil
	}
	return nil, errors.New("not found")
}

func (f fakeAppStoreWith) GetAppByID(ctx context.Context, id uuid.UUID) (*appsdomain.App, error) {
	if f.app.ID == id {
		return f.app, nil
	}
	return nil, errors.New("not found")
}

func (f fakeAppStoreWith) ListAppsByOrg(ctx context.Context, orgID uuid.UUID) ([]appsdomain.App, error) {
	return nil, nil
}
