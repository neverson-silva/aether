package core

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"aether/internal/domain"
)

func (c *Core) previewDomain(branch, appName string) string {
	safe := strings.NewReplacer("/", "-", ".", "-", "_", "-").Replace(branch)
	return fmt.Sprintf("%s-%s.preview.aether.local", appName, safe)
}

func (c *Core) previewDomainFor(app *domain.App, branch string) string {
	if app.PreviewDomain != "" {
		return fmt.Sprintf("%s-%s.%s", app.Name, strings.NewReplacer("/", "-", ".", "-", "_", "-").Replace(branch), app.PreviewDomain)
	}
	return c.previewDomain(branch, app.Name)
}

func (c *Core) CreatePreview(appID, branch string) (*domain.Preview, error) {
	app, err := c.Store.GetApp(appID)
	if err != nil {
		return nil, err
	}
	if app.SourceType != domain.SourceGit {
		return nil, fmt.Errorf("previews requerem fonte git")
	}
	if branch == "" {
		branch = "preview"
	}
	existing, err := c.Store.ListPreviews(appID)
	if err == nil {
		for i := range existing {
			if existing[i].Branch == branch && existing[i].Status == "active" {
				return c.refreshPreview(&existing[i])
			}
		}
	}
	p := &domain.Preview{
		ID:        domain.NewID(),
		AppID:     appID,
		Branch:    branch,
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}
	p.Domain = c.previewDomainFor(app, branch)
	if err := c.Store.CreatePreview(p); err != nil {
		return nil, err
	}
	dep, err := c.Deploy(appID, DeployOpts{Trigger: "preview", Commit: branch})
	if err != nil {
		return nil, err
	}
	p.DeploymentID = dep.ID
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		deadline := time.Now().Add(10 * time.Minute)
		for time.Now().Before(deadline) {
			d, err := c.Store.GetDeployment(dep.ID)
			if err == nil {
				if d.Status == domain.DeploymentReady {
					p.ContainerID = d.ContainerID
					c.Store.UpdatePreview(p)
					c.Net.SetRoute(p.Domain, "http://127.0.0.1:"+c.containerPort(ctx, d))
					return
				}
				if d.Status == domain.DeploymentFailed {
					p.Status = "failed"
					c.Store.UpdatePreview(p)
					return
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}()
	return p, nil
}

func (c *Core) refreshPreview(p *domain.Preview) (*domain.Preview, error) {
	dep, err := c.Deploy(p.AppID, DeployOpts{Trigger: "preview", Commit: p.Branch})
	if err != nil {
		return nil, err
	}
	p.DeploymentID = dep.ID
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		deadline := time.Now().Add(10 * time.Minute)
		for time.Now().Before(deadline) {
			d, err := c.Store.GetDeployment(dep.ID)
			if err == nil {
				if d.Status == domain.DeploymentReady {
					p.ContainerID = d.ContainerID
					c.Store.UpdatePreview(p)
					c.Net.SetRoute(p.Domain, "http://127.0.0.1:"+c.containerPort(ctx, d))
					return
				}
				if d.Status == domain.DeploymentFailed {
					p.Status = "failed"
					c.Store.UpdatePreview(p)
					return
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}()
	return p, nil
}

func (c *Core) DeletePreview(id string) error {
	p, err := c.Store.GetPreview(id)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if p.ContainerID != "" {
		c.Driver.Remove(ctx, p.ContainerID, true)
	}
	c.Net.RemoveRoute(p.Domain)
	if p.DeploymentID != "" {
		if d, err := c.Store.GetDeployment(p.DeploymentID); err == nil {
			d.Status = domain.DeploymentCancelled
			d.FinishedAt = time.Now().UTC()
			c.Store.UpdateDeployment(d)
		}
	}
	return c.Store.DeletePreview(id)
}

func (c *Core) ListPreviews(appID string) ([]domain.Preview, error) {
	return c.Store.ListPreviews(appID)
}

func (c *Core) previewNumber(appID string) int64 {
	n, err := c.Store.NextDeploymentNumber(appID)
	if err != nil {
		return time.Now().UnixNano()
	}
	_ = strconv.FormatInt(n, 10)
	return n
}

var _ = log.Printf
