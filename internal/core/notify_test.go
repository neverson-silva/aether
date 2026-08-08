package core

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"aether/internal/domain"
	"aether/internal/druntime/pubsub"
)

var msgs = make(chan []byte, 256)

func TestNotificationEngineEmitsOnDeploy(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	user, org, err := c.CreateUserAndOrg("nt@aether.local", "nt", "senha-nt")
	if err != nil {
		t.Fatal(err)
	}
	_ = user
	proj, _ := c.CreateProject(org.ID, "ntproj")
	envs, _ := c.ListEnvironments(proj.ID)
	app := &domain.App{ID: domain.NewID(), OrgID: org.ID, ProjectID: proj.ID, EnvironmentID: envs[0].ID, Name: "api", SourceType: domain.SourceImage, Image: "nginx:alpine", Port: 18080}
	c.CreateApp(org.ID, app)

	// registrar um cliente SSE via PubSub distribuído
	sub, err := c.RT.PubSub.Subscribe(context.Background(), "notify:org:"+org.ID, func(ctx context.Context, m pubsub.Message) {
		select {
		case msgs <- m.Data:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	dep, err := c.Deploy(app.ID, DeployOpts{Trigger: "api", TriggeredBy: "tester@aether.local"})
	if err != nil {
		t.Fatal(err)
	}
	_ = dep
	waitDeployment(t, c, dep.ID, domain.DeploymentReady)

	// aguardar ciclo completo (queued, building, starting, healthcheck, ready)
	var gotQueued, gotBuilding, gotStarting, gotHealthcheck, gotReady bool
	for i := 0; i < 6; i++ {
		select {
		case data := <-msgs:
			var n domain.Notification
			if err := json.Unmarshal(data, &n); err != nil {
				continue
			}
			switch n.Type {
			case "deployment.queued":
				gotQueued = true
			case "deployment.building":
				gotBuilding = true
			case "deployment.starting":
				gotStarting = true
			case "deployment.healthcheck":
				gotHealthcheck = true
			case "deployment.ready":
				gotReady = true
			}
			t.Logf("SSE: %s", n.Message)
		case <-time.After(15 * time.Second):
			t.Fatal("nenhuma notificação via SSE")
		}
		if gotQueued && gotBuilding && gotStarting && gotHealthcheck && gotReady {
			break
		}
	}
	if !gotQueued || !gotBuilding || !gotStarting || !gotHealthcheck || !gotReady {
		t.Fatalf("ciclo incompleto: queued=%v building=%v starting=%v healthcheck=%v ready=%v", gotQueued, gotBuilding, gotStarting, gotHealthcheck, gotReady)
	}

	// verificar persistência
	notifs, err := c.Store.ListNotifications(org.ID, "", 10)
	if err != nil || len(notifs) == 0 {
		t.Fatalf("sem notificações persistidas: %d %v", len(notifs), err)
	}
	foundQueued := false
	foundReady := false
	for _, n := range notifs {
		if n.Type == "deployment.queued" {
			foundQueued = true
		}
		if n.Type == "deployment.ready" {
			foundReady = true
		}
	}
	if !foundQueued || !foundReady {
		t.Fatalf("deveria ter queued+ready: %+v", notifs)
	}

	// unread count
	count, err := c.Store.UnreadNotificationCount(org.ID)
	if err != nil || count == 0 {
		t.Fatalf("unread: %d %v", count, err)
	}
	_ = context.Background
}
