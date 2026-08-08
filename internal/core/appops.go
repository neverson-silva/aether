package core

import (
	"context"
	"time"

	"aether/internal/domain"
)

func (c *Core) appContainer(ctx context.Context, appID string) (string, error) {
	deploys, err := c.Store.ListDeployments(appID, 1)
	if err != nil {
		return "", err
	}
	if len(deploys) == 0 || deploys[0].ContainerID == "" {
		return "", errNoContainer
	}
	return deploys[0].ContainerID, nil
}

var errNoContainer = &appNoContainerError{}

type appNoContainerError struct{}

func (e *appNoContainerError) Error() string { return "app has no running container yet" }

func (c *Core) AppStart(appID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cid, err := c.appContainer(ctx, appID)
	if err != nil {
		return err
	}
	info, err := c.Driver.Inspect(ctx, cid)
	if err != nil {
		return err
	}
	if info.State == "running" {
		c.PublishAppState(appID, "running")
		return nil
	}
	if err := c.Driver.Start(ctx, cid); err != nil {
		return err
	}
	c.PublishAppState(appID, "running")
	return nil
}

func (c *Core) AppStop(appID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cid, err := c.appContainer(ctx, appID)
	if err != nil {
		return err
	}
	info, err := c.Driver.Inspect(ctx, cid)
	if err != nil {
		return err
	}
	if info.State == "exited" || info.State == "stopped" {
		c.PublishAppState(appID, "exited")
		return nil
	}
	c.stopContainerLogCollector(appID)
	if err := c.Driver.Stop(ctx, cid); err != nil {
		return err
	}
	c.PublishAppState(appID, "exited")
	return nil
}

func (c *Core) AppRestart(appID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cid, err := c.appContainer(ctx, appID)
	if err != nil {
		return err
	}
	if err := c.Driver.Restart(ctx, cid); err != nil {
		return err
	}
	c.PublishAppState(appID, "running")
	app, aerr := c.Store.GetApp(appID)
	if aerr != nil {
		return nil
	}
	dep, derr := c.Store.LastReadyDeployment(appID, 1<<62)
	if derr == nil {
		c.startContainerLogCollector(app, dep)
	}
	return nil
}

func (c *Core) AppState(appID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cid, err := c.appContainer(ctx, appID)
	if err != nil {
		return "no_container"
	}
	info, err := c.Driver.Inspect(ctx, cid)
	if err != nil {
		return "unknown"
	}
	return info.State
}

func (c *Core) AppRebuild(appID, triggeredBy string) (*domain.Deployment, error) {
	return c.Deploy(appID, DeployOpts{Trigger: "rebuild", TriggeredBy: triggeredBy})
}
