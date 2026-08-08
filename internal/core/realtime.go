package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"aether/internal/db"
	"aether/internal/domain"
	"aether/internal/druntime/pubsub"
	"aether/internal/events"
)

type NotificationEngine struct {
	store *db.Store
	pub   pubsub.PubSub
}

func newNotificationEngine(store *db.Store, pub pubsub.PubSub) *NotificationEngine {
	return &NotificationEngine{store: store, pub: pub}
}

func (e *NotificationEngine) subscribe(bus *events.Bus) {
	bus.Subscribe(e.onEvent)
}

func (e *NotificationEngine) onEvent(ctx context.Context, ev events.Event) {
	switch ev.Type {
	case "deployment.created", "deployment.building", "deployment.starting", "deployment.healthcheck", "app.deployed", "app.deploy_failed", "app.rolled_back":
		e.onDeployEvent(ctx, ev)
	case "server.registered", "server.marked_unhealthy", "server.recovered":
		e.onServerEvent(ctx, ev)
	case "backup.started", "backup.finished", "backup.failed":
		e.onBackupEvent(ctx, ev)
	}
}

func (e *NotificationEngine) emit(orgID, typ, message string, payload any) {
	raw := "{}"
	if b, err := json.Marshal(payload); err == nil {
		raw = string(b)
	}
	n := domain.Notification{
		ID:        domain.NewID(),
		OrgID:     orgID,
		Type:      typ,
		Message:   message,
		Payload:   raw,
		CreatedAt: time.Now().UTC(),
	}
	if err := e.store.CreateNotification(&n); err != nil {
		return
	}
	if raw, err := json.Marshal(n); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = e.pub.Publish(ctx, "notify:org:"+orgID, raw)
	}
}

func (e *NotificationEngine) onDeployEvent(ctx context.Context, ev events.Event) {
	var appID string
	if ev.AggregateType == "deployment" {
		var payload map[string]any
		_ = json.Unmarshal(ev.Payload, &payload)
		appID, _ = payload["app_id"].(string)
	} else {
		appID = ev.AggregateID
	}
	app, err := e.store.GetApp(appID)
	if err != nil {
		return
	}
	org, _ := e.store.GetOrg(app.OrgID)
	proj, _ := e.store.GetProject(app.ProjectID)
	env := ""
	if app.EnvironmentID != "" {
		if en, err := e.store.GetEnvironment(app.EnvironmentID); err == nil {
			env = en.Name
		}
	}
	projectName, orgName := app.ProjectID, app.OrgID
	if proj != nil {
		projectName = proj.Name
	}
	if org != nil {
		orgName = org.Name
	}

	var payload map[string]any
	_ = json.Unmarshal(ev.Payload, &payload)
	deployNum := int64(0)
	if n, ok := payload["number"].(float64); ok {
		deployNum = int64(n)
	}
	triggeredBy := ""
	if deploys, err := e.store.ListDeployments(app.ID, 1); err == nil && len(deploys) > 0 {
		triggeredBy = deploys[0].TriggeredBy
	}

	msg := ""
	var notifType string
	switch ev.Type {
	case "deployment.created":
		notifType = "deployment.queued"
		who := "manual"
		if triggeredBy != "" {
			who = triggeredBy
		}
		msg = "deploy queued · " + app.Name + " · triggered by " + who
	case "deployment.building":
		notifType = "deployment.building"
		buildMethod, _ := payload["build_method"].(string)
		if buildMethod == "" {
			buildMethod = "dockerfile"
		}
		msg = "Building " + app.Name
		if commit, ok := payload["commit"].(string); ok && commit != "" {
			msg += " · commit " + truncate(commit, 8)
		}
		msg += " · " + buildMethod
	case "deployment.starting":
		notifType = "deployment.starting"
		msg = "Starting " + app.Name
		if cid, ok := payload["container_id"].(string); ok && cid != "" {
			msg += " · container " + truncate(cid, 12)
		}
	case "deployment.healthcheck":
		notifType = "deployment.healthcheck"
		path, _ := payload["path"].(string)
		if path == "" {
			path = "/"
		}
		msg = "Health check " + app.Name + " · " + path
	case "app.deployed":
		notifType = "deployment.ready"
		msg = "✅ " + app.Name + " deployed successfully"
		if duration := e.deployDurationMs(ev); duration > 0 {
			msg += fmt.Sprintf(" · %ds", duration/1000)
		}
		msg += " · port " + strconv.Itoa(app.Port)
	case "app.deploy_failed":
		notifType = "deployment.failed"
		errMsg, _ := payload["error"].(string)
		msg = "❌ " + app.Name + " deploy failed"
		if errMsg != "" {
			msg += " · " + truncate(errMsg, 120)
		}
	case "app.rolled_back":
		notifType = "deployment.rolled_back"
		msg = "↩️ " + app.Name + " rolled back"
	}
	_ = deployNum

	e.emit(app.OrgID, notifType, msg, map[string]any{
		"service_id":       app.ID,
		"service_name":     app.Name,
		"project_id":       app.ProjectID,
		"project_name":     projectName,
		"environment_id":   app.EnvironmentID,
		"environment_name": env,
		"org_id":           app.OrgID,
		"org_name":         orgName,
		"triggered_by":     triggeredBy,
	})
}

func (e *NotificationEngine) deployDurationMs(ev events.Event) int64 {
	var payload map[string]any
	_ = json.Unmarshal(ev.Payload, &payload)
	if depID, ok := payload["deployment_id"].(string); ok {
		if dep, err := e.store.GetDeployment(depID); err == nil && !dep.StartedAt.IsZero() {
			end := dep.FinishedAt
			if end.IsZero() {
				end = time.Now()
			}
			return end.Sub(dep.StartedAt).Milliseconds()
		}
	}
	return 0
}

func (e *NotificationEngine) onBackupEvent(ctx context.Context, ev events.Event) {
	var payload map[string]any
	_ = json.Unmarshal(ev.Payload, &payload)
	orgID, _ := payload["org_id"].(string)
	if orgID == "" {
		return
	}
	dbName, _ := payload["database"].(string)
	msg := ""
	switch ev.Type {
	case "backup.started":
		msg = "⏳ Backup started · " + dbName
	case "backup.finished":
		size, _ := payload["size"].(float64)
		msg = "✅ Backup finished · " + dbName
		if size > 0 {
			msg += fmt.Sprintf(" · %.1f MB", size/1024/1024)
		}
	case "backup.failed":
		msg = "❌ Backup failed · " + dbName
		if e, ok := payload["error"].(string); ok && e != "" {
			msg += " · " + truncate(e, 120)
		}
	}
	e.emit(orgID, ev.Type, msg, payload)
}

func (e *NotificationEngine) onServerEvent(ctx context.Context, ev events.Event) {
	var payload map[string]any
	_ = json.Unmarshal(ev.Payload, &payload)
	name, _ := payload["name"].(string)
	if name == "" {
		return
	}
	msg := ""
	switch ev.Type {
	case "server.registered":
		msg = "🆕 server " + name + " registered"
	case "server.marked_unhealthy":
		msg = "⚠️ server " + name + " marked unhealthy"
	case "server.recovered":
		msg = "✅ server " + name + " recovered"
	}
	orgs, _ := e.store.ListOrgs()
	for _, o := range orgs {
		e.emit(o.ID, ev.Type, msg, map[string]any{"server_id": payload["server_id"], "server_name": name})
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func (c *Core) EmitOrg(orgID, typ, message string, payload any) {
	c.notify.emit(orgID, typ, message, payload)
}

func (c *Core) EmitAll(typ, message string, payload any) {
	orgs, err := c.Store.ListOrgs()
	if err != nil {
		return
	}
	for _, o := range orgs {
		c.notify.emit(o.ID, typ, message, payload)
	}
}

func (c *Core) publishDeployEvent(app *domain.App, dep *domain.Deployment, eventType string, extra map[string]any) {
	payload := map[string]any{
		"app_id":        app.ID,
		"deployment_id": dep.ID,
		"number":        dep.Number,
		"image":         dep.ImageRef,
		"trigger":       dep.Trigger,
		"triggered_by":  dep.TriggeredBy,
	}
	for k, v := range extra {
		payload[k] = v
	}
	_ = c.Bus.Publish(context.Background(), "app", app.ID, eventType, payload, nil)
}
