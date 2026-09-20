package application

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	deploydomain "aether/internal/modules/deployments/domain"
)

type AppOps struct {
	Deployments *Deployments
	Runtime     ContainerRuntime
}

type ContainerRuntime interface {
	ContainerState(ctx context.Context, containerID string) (string, error)
	Start(ctx context.Context, containerID string) error
	Stop(ctx context.Context, containerID string) error
	Restart(ctx context.Context, containerID string) error
	Remove(ctx context.Context, containerID string) error
	RemoveByLabel(ctx context.Context, label string) error
}

func (o *AppOps) State(ctx context.Context, appID, orgID uuid.UUID) (string, error) {
	container, err := o.latestContainer(ctx, appID, orgID)
	if err != nil {
		return "no_container", nil
	}
	state, err := o.Runtime.ContainerState(ctx, container)
	if err != nil {
		return "unknown", nil
	}
	return state, nil
}

func (o *AppOps) Start(ctx context.Context, appID, orgID uuid.UUID) (string, error) {
	container, err := o.latestContainer(ctx, appID, orgID)
	if err != nil {
		return "", err
	}
	if err := o.Runtime.Start(ctx, container); err != nil {
		return "", err
	}
	return o.State(ctx, appID, orgID)
}

func (o *AppOps) Stop(ctx context.Context, appID, orgID uuid.UUID) (string, error) {
	container, err := o.latestContainer(ctx, appID, orgID)
	if err != nil {
		return "", err
	}
	if err := o.Runtime.Stop(ctx, container); err != nil {
		return "", err
	}
	return o.State(ctx, appID, orgID)
}

func (o *AppOps) Restart(ctx context.Context, appID, orgID uuid.UUID) (string, error) {
	container, err := o.latestContainer(ctx, appID, orgID)
	if err != nil {
		return "", err
	}
	if err := o.Runtime.Restart(ctx, container); err != nil {
		return "", err
	}
	return o.State(ctx, appID, orgID)
}

// Delete para e remove o container do app (se houver). Idempotente.
func (o *AppOps) Delete(ctx context.Context, appID, orgID uuid.UUID) error {
	return o.DeleteService(ctx, appID, appID, orgID)
}

func (o *AppOps) StartService(ctx context.Context, appID, serviceID, orgID uuid.UUID) (string, error) {
	return o.start(ctx, appID, serviceID, orgID)
}

func (o *AppOps) StopService(ctx context.Context, appID, serviceID, orgID uuid.UUID) (string, error) {
	return o.stop(ctx, appID, serviceID, orgID)
}

func (o *AppOps) RestartService(ctx context.Context, appID, serviceID, orgID uuid.UUID) (string, error) {
	container, err := o.latestContainer(ctx, appID, orgID)
	if err != nil {
		return "", err
	}
	if err := o.Runtime.Restart(ctx, container); err != nil {
		return "", err
	}
	return o.State(ctx, appID, orgID)
}

func (o *AppOps) DeleteService(ctx context.Context, appID, serviceID, orgID uuid.UUID) error {
	if err := o.Runtime.RemoveByLabel(ctx, "aether.service-id="+serviceID.String()); err == nil {
		return nil
	}
	container, err := o.latestContainer(ctx, appID, orgID)
	if err != nil {
		return nil
	}
	return o.Runtime.Remove(ctx, container)
}

func (o *AppOps) start(ctx context.Context, appID, serviceID, orgID uuid.UUID) (string, error) {
	container, err := o.latestContainer(ctx, appID, orgID)
	if err != nil {
		return "", err
	}
	if err := o.Runtime.Start(ctx, container); err != nil {
		return "", err
	}
	return o.State(ctx, appID, orgID)
}

func (o *AppOps) stop(ctx context.Context, appID, serviceID, orgID uuid.UUID) (string, error) {
	container, err := o.latestContainer(ctx, appID, orgID)
	if err != nil {
		return "", err
	}
	if err := o.Runtime.Stop(ctx, container); err != nil {
		return "", err
	}
	return o.State(ctx, appID, orgID)
}

func (o *AppOps) Rebuild(ctx context.Context, appID, orgID uuid.UUID, by string) (*deploydomain.Deployment, error) {
	return o.Deployments.Deploy(ctx, appID, orgID, DeployOpts{Trigger: "rebuild", TriggeredBy: by})
}

func (o *AppOps) States(ctx context.Context, orgID uuid.UUID) (map[uuid.UUID]string, error) {
	apps, err := o.Deployments.Apps.ListAppsByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	states := make(map[uuid.UUID]string, len(apps))
	for _, app := range apps {
		state, _ := o.State(ctx, app.ID, orgID)
		states[app.ID] = state
	}
	return states, nil
}

func (o *AppOps) latestContainer(ctx context.Context, appID, orgID uuid.UUID) (string, error) {
	if _, err := o.Deployments.Apps.GetApp(ctx, appID, orgID); err != nil {
		return "", err
	}
	deps, err := o.Deployments.Store.ListByApp(ctx, appID, 1)
	if err != nil || len(deps) == 0 {
		return "", deploydomain.ErrNotFound
	}
	if deps[0].ContainerID == "" {
		return "", deploydomain.ErrNotFound
	}
	return deps[0].ContainerID, nil
}

func (o *AppOps) Timeline(ctx context.Context, appID, orgID uuid.UUID) ([]TimelineEntry, error) {
	if _, err := o.Deployments.Apps.GetApp(ctx, appID, orgID); err != nil {
		return nil, err
	}
	deps, err := o.Deployments.Store.ListByApp(ctx, appID, 50)
	if err != nil {
		return nil, err
	}
	out := make([]TimelineEntry, 0, len(deps))
	for _, dep := range deps {
		out = append(out, TimelineEntry{
			ID: dep.ID, Number: dep.Number, Status: string(dep.Status),
			Trigger: dep.Trigger, TriggeredBy: dep.TriggeredBy, CreatedAt: dep.CreatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

type TimelineEntry struct {
	ID          uuid.UUID
	Number      int
	Status      string
	Trigger     string
	TriggeredBy string
	CreatedAt   time.Time
}
