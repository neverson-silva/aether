package application

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	alertsdomain "aether/internal/alerts/domain"
	appsdomain "aether/internal/apps/domain"
	deploydomain "aether/internal/deployments/domain"
	"aether/internal/druntime/presence"
	"aether/internal/druntime/pubsub"
	"aether/internal/hostinfo"
	"aether/internal/realtime/domain"
)

type Realtime struct {
	Presence      presence.Presence
	PubSub        pubsub.PubSub
	Apps          AppStore
	Deployments   DeploymentStore
	Ports         PortReader
	Log           EventLog
	Notifications NotificationsSink

	mu    sync.Mutex
	stats map[string]*domain.NetAppStat
}

type NotificationsSink interface {
	Create(ctx context.Context, notification *alertsdomain.Notification) (*alertsdomain.Notification, error)
}

type EventLog interface {
	Append(ctx context.Context, orgID uuid.UUID, ev domain.Event) (int64, error)
	Recent(ctx context.Context, orgID uuid.UUID, limit int) ([]domain.Event, error)
	Replay(ctx context.Context, orgID uuid.UUID, afterSeq int64, limit int) ([]domain.Event, error)
}

type AppStore interface {
	GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error)
	GetAppByID(ctx context.Context, id uuid.UUID) (*appsdomain.App, error)
	ListAppsByOrg(ctx context.Context, orgID uuid.UUID) ([]appsdomain.App, error)
}

type DeploymentStore interface {
	ListByApp(ctx context.Context, appID uuid.UUID, limit int) ([]deploydomain.Deployment, error)
	GetDeployment(ctx context.Context, id uuid.UUID) (*deploydomain.Deployment, error)
}

type PortReader interface {
	Port(ctx context.Context, containerID string) (hostPort string, err error)
}

func (r *Realtime) Metrics(ctx context.Context) domain.Metrics {
	metrics := domain.Metrics{Backend: "memory"}
	if r.PubSub != nil {
		if subscribers, err := r.PubSub.Subscribers(ctx); err == nil {
			metrics.Subscribers = subscribers
			metrics.TotalChannels = len(subscribers)
		}
	}
	return metrics
}

func (r *Realtime) Join(ctx context.Context, scope, member string) error {
	if scope == "" {
		return domain.ErrValidation
	}
	return r.Presence.Join(ctx, scope, member, 60*time.Second)
}

func (r *Realtime) Heartbeat(ctx context.Context, scope, member string) error {
	if scope == "" {
		return domain.ErrValidation
	}
	return r.Presence.Heartbeat(ctx, scope, member, 60*time.Second)
}

func (r *Realtime) Leave(ctx context.Context, scope, member string) error {
	return r.Presence.Leave(ctx, scope, member)
}

func (r *Realtime) Count(ctx context.Context, scope string) (int64, []string, error) {
	if scope == "" {
		return 0, nil, domain.ErrValidation
	}
	count, err := r.Presence.Count(ctx, scope)
	if err != nil {
		return 0, nil, err
	}
	members, err := r.Presence.Members(ctx, scope)
	return count, members, err
}

func (r *Realtime) NotifyDeploy(ctx context.Context, event deploydomain.DeployEvent) {
	app, err := r.Apps.GetAppByID(ctx, event.AppID)
	if err != nil {
		return
	}
	payload, _ := json.Marshal(event)
	msg := deployMessage(app.Name, event)
	_ = r.PublishEvent(ctx, app.OrgID, domain.Event{
		Type: "deploy." + event.Status, Aggregate: "deployment",
		AppID:         event.AppID.String(),
		ResourceType:  "deployment",
		ResourceID:    event.DepID.String(),
		CorrelationID: event.DepID.String(),
		Message:       msg,
		Payload:       payload,
	})
	if r.Notifications != nil && notifiableDeployStatus(event.Status) {
		_, _ = r.Notifications.Create(ctx, &alertsdomain.Notification{
			OrgID:   app.OrgID,
			Type:    "deploy." + event.Status,
			Message: msg,
			Payload: string(payload),
		})
	}
}

func notifiableDeployStatus(status string) bool {
	switch status {
	case "queued", "ready", "failed", "rolled_back", "cancelled":
		return true
	}
	return false
}

func (r *Realtime) NotifyDeployLog(ctx context.Context, appID, depID uuid.UUID, line string) {
	app, err := r.Apps.GetAppByID(ctx, appID)
	if err != nil {
		return
	}
	payload, _ := json.Marshal(map[string]string{
		"line": line,
		"ts":   time.Now().UTC().Format(time.RFC3339),
	})
	_ = r.PublishEvent(ctx, app.OrgID, domain.Event{
		Type: "deploy.build.log", Aggregate: "deployment",
		AppID: appID.String(), ResourceType: "deployment", ResourceID: depID.String(),
		CorrelationID: depID.String(), Message: line, Payload: payload, Ephemeral: true,
	})
}

func (r *Realtime) NotifyAppState(ctx context.Context, appID uuid.UUID, state string) {
	app, err := r.Apps.GetAppByID(ctx, appID)
	if err != nil {
		return
	}
	payload, _ := json.Marshal(map[string]string{"app_id": appID.String(), "state": state})
	_ = r.PublishEvent(ctx, app.OrgID, domain.Event{
		Type: "app.state", Aggregate: "app",
		AppID: appID.String(), ResourceType: "app", ResourceID: appID.String(),
		Payload: payload, Ephemeral: true,
	})
}

func deployMessage(appName string, event deploydomain.DeployEvent) string {
	if event.Detail != "" {
		if strings.HasPrefix(event.Detail, "container stopped") {
			return appName + " stopped"
		}
		return "Deploy failed for " + appName + ": " + event.Detail
	}
	switch event.Status {
	case "queued":
		return "Deploy of " + appName + " started"
	case "building":
		return "Building " + appName
	case "starting":
		return "Starting " + appName
	case "health_checking":
		return "Health checking " + appName
	case "ready":
		return "Deploy of " + appName + " completed"
	case "rolled_back":
		return "Deploy of " + appName + " rolled back"
	case "cancelled":
		return "Deploy of " + appName + " cancelled"
	default:
		return "Deploy of " + appName + ": " + event.Status
	}
}

func (r *Realtime) PublishEvent(ctx context.Context, orgID uuid.UUID, event domain.Event) error {
	event.ID = uuid.NewString()
	if event.Version == 0 {
		event.Version = 1
	}
	event.OrgID = orgID.String()
	if event.CorrelationID == "" {
		event.CorrelationID = event.ID
	}
	event.TS = time.Now().UTC()
	if !event.Ephemeral {
		if r.Log == nil {
			r.Log = &memoryLog{}
		}
		seq, err := r.Log.Append(ctx, orgID, event)
		if err != nil {
			return err
		}
		event.Seq = seq
	}
	if r.PubSub == nil {
		return nil
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return r.PubSub.Publish(ctx, "notify:org:"+orgID.String(), data)
}

func (r *Realtime) RecentEvents(ctx context.Context, orgID uuid.UUID, limit int) ([]domain.Event, error) {
	if r.Log == nil {
		r.Log = &memoryLog{}
	}
	return r.Log.Recent(ctx, orgID, limit)
}

func (r *Realtime) ReplayEvents(ctx context.Context, orgID uuid.UUID, afterSeq int64, limit int) ([]domain.Event, error) {
	if r.Log == nil {
		r.Log = &memoryLog{}
	}
	return r.Log.Replay(ctx, orgID, afterSeq, limit)
}

func (r *Realtime) Authorize(ctx context.Context, scope string, orgID uuid.UUID) error {
	switch {
	case scope == "org":
		return nil
	case strings.HasPrefix(scope, "app:"):
		id, err := uuid.Parse(strings.TrimPrefix(scope, "app:"))
		if err != nil {
			return domain.ErrForbidden
		}
		_, err = r.Apps.GetApp(ctx, id, orgID)
		return err
	case strings.HasPrefix(scope, "deployment:"):
		id, err := uuid.Parse(strings.TrimPrefix(scope, "deployment:"))
		if err != nil {
			return domain.ErrForbidden
		}
		dep, err := r.Deployments.GetDeployment(ctx, id)
		if err != nil {
			return domain.ErrForbidden
		}
		_, err = r.Apps.GetApp(ctx, dep.AppID, orgID)
		return err
	default:
		return domain.ErrForbidden
	}
}

func (r *Realtime) SubscribeEvents(ctx context.Context, orgID uuid.UUID, handler func(domain.Event)) (pubsub.Subscription, error) {
	if r.PubSub == nil {
		return nil, nil
	}
	return r.PubSub.Subscribe(ctx, "notify:org:"+orgID.String(), func(_ context.Context, msg pubsub.Message) {
		handler(domain.ParseEvent(msg.Data))
	}, pubsub.WithBuffer(256))
}

func (r *Realtime) ReadyContainer(ctx context.Context, appID, orgID uuid.UUID) (string, error) {
	if _, err := r.Apps.GetApp(ctx, appID, orgID); err != nil {
		return "", err
	}
	deployments, err := r.Deployments.ListByApp(ctx, appID, 1)
	if err != nil {
		return "", err
	}
	for _, deployment := range deployments {
		if deployment.Status == deploydomain.StatusReady && deployment.ContainerID != "" {
			return deployment.ContainerID, nil
		}
	}
	return "", errors.New("no active container")
}

func (r *Realtime) Probe(ctx context.Context, orgID uuid.UUID) []domain.NetAppStat {
	r.mu.Lock()
	if r.stats == nil {
		r.stats = make(map[string]*domain.NetAppStat)
	}
	r.mu.Unlock()
	apps, err := r.Apps.ListAppsByOrg(ctx, orgID)
	if err != nil {
		return nil
	}
	for i := range apps {
		r.probeApp(ctx, &apps[i])
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.NetAppStat, 0, len(r.stats))
	for _, st := range r.stats {
		out = append(out, *st)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (r *Realtime) probeApp(ctx context.Context, app *appsdomain.App) {
	deployments, err := r.Deployments.ListByApp(ctx, app.ID, 1)
	if err != nil || len(deployments) == 0 {
		return
	}
	deployment := deployments[0]
	if deployment.ContainerID == "" || deployment.Status != deploydomain.StatusReady {
		return
	}
	hostPort, err := r.Ports.Port(ctx, deployment.ContainerID)
	if err != nil || hostPort == "" {
		return
	}
	host, port := splitHostPort(hostPort)
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	start := time.Now()
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Get("http://" + addr)
	ms := float64(time.Since(start).Microseconds()) / 1000
	sample := domain.NetSample{At: time.Now(), Ms: ms, OK: err == nil}
	if err == nil {
		if resp.StatusCode >= 500 {
			sample.OK = false
		}
		sample.H3 = resp.Header.Get("Alt-Svc") != ""
		resp.Body.Close()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	st, ok := r.stats[app.ID.String()]
	if !ok {
		st = &domain.NetAppStat{AppID: app.ID.String(), Name: app.Name, Addr: addr}
		r.stats[app.ID.String()] = st
	}
	st.Samples = append(st.Samples, sample)
	if len(st.Samples) > 120 {
		st.Samples = st.Samples[len(st.Samples)-120:]
	}
	var lats []float64
	okCount := 0
	for _, x := range st.Samples {
		if x.OK {
			lats = append(lats, x.Ms)
			okCount++
		}
		if x.H3 {
			st.H3 = true
		}
	}
	sort.Float64s(lats)
	st.P50 = percentile(lats, 50)
	st.P95 = percentile(lats, 95)
	if len(st.Samples) > 0 {
		st.Uptime = float64(okCount) / float64(len(st.Samples)) * 100
	}
}

func splitHostPort(hostPort string) (string, int) {
	host, portStr, err := net.SplitHostPort(hostPort)
	if err != nil {
		if p, perr := strconv.Atoi(hostPort); perr == nil {
			return hostinfo.PublicIP(), p
		}
		return hostinfo.PublicIP(), 80
	}
	port, _ := strconv.Atoi(portStr)
	if host == "" || host == "0.0.0.0" {
		host = hostinfo.PublicIP()
	}
	if port == 0 {
		port = 80
	}
	return host, port
}

func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := (len(sorted) * p) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

type memoryLog struct {
	mu   sync.Mutex
	orgs map[string][]domain.Event
}

func (m *memoryLog) Append(ctx context.Context, orgID uuid.UUID, ev domain.Event) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.orgs == nil {
		m.orgs = make(map[string][]domain.Event)
	}
	key := orgID.String()
	list := m.orgs[key]
	seq := int64(len(list)) + 1
	ev.Seq = seq
	list = append(list, ev)
	if len(list) > 200 {
		list = list[len(list)-200:]
	}
	m.orgs[key] = list
	return seq, nil
}

func (m *memoryLog) Recent(ctx context.Context, orgID uuid.UUID, limit int) ([]domain.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	list := m.orgs[orgID.String()]
	out := make([]domain.Event, 0, limit)
	for i := len(list) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, list[i])
	}
	return out, nil
}

func (m *memoryLog) Replay(ctx context.Context, orgID uuid.UUID, afterSeq int64, limit int) ([]domain.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	list := m.orgs[orgID.String()]
	out := make([]domain.Event, 0, limit)
	for _, ev := range list {
		if ev.Seq > afterSeq {
			out = append(out, ev)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}
