package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	deploydomain "aether/internal/modules/deployments/domain"
	servicesdomain "aether/internal/modules/services/domain"
)

type Watcher struct {
	Store        DeploymentStore
	Runtime      Runtime
	ServiceStore ServiceStateStore
	Notifier     WatcherNotifier
	Logger       *slog.Logger
}

type WatcherNotifier interface {
	NotifyDeploy(ctx context.Context, event deploydomain.DeployEvent)
	NotifyAppState(ctx context.Context, appID uuid.UUID, state string)
}

type ServiceStateNotifier interface {
	NotifyServiceState(ctx context.Context, organizationID, serviceID uuid.UUID, state string)
}

func (w *Watcher) Run(ctx context.Context, _ time.Duration) {
	if w.ServiceStore == nil {
		return
	}
	eventsRuntime, ok := w.Runtime.(EventRuntime)
	if !ok {
		w.log(ctx, "runtime event subscription unavailable", errors.New("runtime does not implement EventRuntime"))
		return
	}
	w.reconcile(ctx)
	for {
		if ctx.Err() != nil {
			return
		}
		subscription, err := eventsRuntime.SubscribeEvents(ctx, nil)
		if err != nil {
			w.log(ctx, "subscribe runtime events", err)
			if !waitReconnect(ctx) {
				return
			}
			continue
		}
		reconnect := w.consumeEvents(ctx, subscription)
		_ = subscription.Close()
		if ctx.Err() != nil {
			return
		}
		w.reconcile(ctx)
		if !reconnect || !waitReconnect(ctx) {
			return
		}
	}
}

func (w *Watcher) consumeEvents(ctx context.Context, subscription RuntimeEventSubscription) bool {
	for {
		select {
		case <-ctx.Done():
			return false
		case _, ok := <-subscription.Events():
			if !ok {
				return true
			}
			w.reconcile(ctx)
		case err, ok := <-subscription.Errors():
			if ok && err != nil {
				w.log(ctx, "runtime event stream interrupted", err)
			}
			return true
		}
	}
}

func waitReconnect(ctx context.Context) bool {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (w *Watcher) reconcile(ctx context.Context) {
	targets, err := w.ServiceStore.ListRuntimeServiceTargets(ctx)
	if err != nil {
		w.log(ctx, "list runtime service targets", err)
		return
	}
	containers, err := w.Runtime.ListContainers(ctx)
	if err != nil {
		w.log(ctx, "list runtime containers", err)
		return
	}
	byService := make(map[uuid.UUID][]servicesdomain.ContainerState)
	for _, item := range containers {
		serviceID, err := uuid.Parse(item.Labels["aether.service-id"])
		if err != nil || serviceID == uuid.Nil {
			continue
		}
		byService[serviceID] = append(byService[serviceID], servicesdomain.ContainerState{ID: item.ID, Name: item.Name, Status: item.State, Healthy: item.Healthy})
	}
	for _, target := range targets {
		state := servicesdomain.ProjectStatus(servicesdomain.Kind(target.Kind), byService[target.ID], target.ActiveDeployment, target.EverDeployed)
		changed, err := w.ServiceStore.UpdateRuntimeStatus(ctx, target.ID, string(state))
		if err != nil {
			w.log(ctx, "persist runtime service status", err)
			continue
		}
		if changed {
			if notifier, ok := w.Notifier.(ServiceStateNotifier); ok {
				notifier.NotifyServiceState(ctx, target.OrganizationID, target.ID, string(state))
			}
		}
	}
}

func (w *Watcher) check(ctx context.Context, last map[uuid.UUID]string) {
	ready, err := w.Store.ListReady(ctx)
	if err != nil {
		w.log(ctx, "list ready", err)
		return
	}
	latest := make(map[uuid.UUID]*deploydomain.Deployment, len(ready))
	for i := range ready {
		dep := &ready[i]
		current := latest[dep.AppID]
		if current == nil || dep.Number > current.Number {
			latest[dep.AppID] = dep
		}
	}
	for _, dep := range latest {
		if dep.ContainerID == "" {
			continue
		}
		state, err := w.Runtime.ContainerState(ctx, dep.ContainerID)
		if err != nil {
			w.log(ctx, "inspect container state", err)
			continue
		}
		switch state {
		case "running":
			w.emitState(ctx, dep.AppID, "running", last)
		case "paused":
			w.emitState(ctx, dep.AppID, "paused", last)
		case "exited", "stopped", "dead":
			w.emitState(ctx, dep.AppID, "error", last)
		default:
			w.log(ctx, "unknown container state", fmt.Errorf("container %s returned state %q", dep.ContainerID, state))
		}
	}
}

func (w *Watcher) emitState(ctx context.Context, appID uuid.UUID, state string, last map[uuid.UUID]string) {
	if last[appID] == state {
		return
	}
	last[appID] = state
	if w.Notifier != nil {
		w.Notifier.NotifyAppState(ctx, appID, state)
	}
}

func (w *Watcher) log(ctx context.Context, msg string, err error) {
	if w.Logger != nil {
		w.Logger.ErrorContext(ctx, msg, "err", err)
	}
}
