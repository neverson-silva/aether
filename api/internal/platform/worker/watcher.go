package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	deploydomain "aether/internal/modules/deployments/domain"
)

type Watcher struct {
	Store    DeploymentStore
	Runtime  Runtime
	Notifier WatcherNotifier
	Logger   *slog.Logger
}

type WatcherNotifier interface {
	NotifyDeploy(ctx context.Context, event deploydomain.DeployEvent)
	NotifyAppState(ctx context.Context, appID uuid.UUID, state string)
}

func (w *Watcher) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	last := map[uuid.UUID]string{}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.check(ctx, last)
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
		w.Logger.Error(msg, "err", err)
	}
}
