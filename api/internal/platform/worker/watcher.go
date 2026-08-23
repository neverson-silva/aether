package worker

import (
	"context"
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
	for i := range ready {
		dep := &ready[i]
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
		default:
			if err := w.Store.UpdateStatus(ctx, dep.ID, deploydomain.StatusFailed, "container "+state, dep.ImageRef, dep.ContainerID, dep.StartedAt, timePtr(time.Now().UTC())); err != nil {
				w.log(ctx, "mark deployment failed", err)
				continue
			}
			if w.Notifier != nil {
				w.Notifier.NotifyDeploy(ctx, deploydomain.DeployEvent{
					AppID: dep.AppID, DepID: dep.ID, Status: string(deploydomain.StatusFailed), Detail: "container " + state,
				})
			}
			w.emitState(ctx, dep.AppID, "error", last)
		}
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
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
