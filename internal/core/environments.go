package core

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"aether/internal/domain"
)

var (
	ErrEnvLast        = errors.New("cannot delete the last environment of the project")
	ErrEnvHasServices = errors.New("environment still has services; move or delete them first")
	ErrEnvExists      = errors.New("environment with this name already exists")
	ErrEnvNotFound    = errors.New("environment not found")
)

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func Slugify(name string) string {
	slug := slugRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	return strings.Trim(slug, "-")
}

func (c *Core) ListEnvironments(projectID string) ([]domain.Environment, error) {
	return c.Store.ListEnvironments(projectID)
}

func (c *Core) CreateEnvironment(projectID, name, description, color string) (*domain.Environment, error) {
	slug := Slugify(name)
	if slug == "" {
		return nil, fmt.Errorf("invalid environment name")
	}
	envs, err := c.Store.ListEnvironments(projectID)
	if err != nil {
		return nil, err
	}
	for _, e := range envs {
		if e.Slug == slug {
			return nil, ErrEnvExists
		}
	}
	env := &domain.Environment{
		ID:          "env-" + domain.NewID(),
		ProjectID:   projectID,
		Name:        strings.TrimSpace(name),
		Slug:        slug,
		Description: description,
		Color:       color,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := c.Store.CreateEnvironment(env); err != nil {
		return nil, err
	}
	return env, nil
}

func (c *Core) UpdateEnvironment(projectID, envID, name, description, color string) (*domain.Environment, error) {
	env, err := c.Store.GetEnvironment(envID)
	if err != nil {
		return nil, ErrEnvNotFound
	}
	if env.ProjectID != projectID {
		return nil, ErrEnvNotFound
	}
	slug := Slugify(name)
	if slug == "" {
		return nil, fmt.Errorf("invalid environment name")
	}
	envs, err := c.Store.ListEnvironments(projectID)
	if err != nil {
		return nil, err
	}
	for _, e := range envs {
		if e.ID != envID && e.Slug == slug {
			return nil, ErrEnvExists
		}
	}
	env.Name = strings.TrimSpace(name)
	env.Slug = slug
	env.Description = description
	env.Color = color
	if err := c.Store.UpdateEnvironment(env); err != nil {
		return nil, err
	}
	return env, nil
}

func (c *Core) DeleteEnvironment(projectID, envID string) error {
	env, err := c.Store.GetEnvironment(envID)
	if err != nil {
		return ErrEnvNotFound
	}
	if env.ProjectID != projectID {
		return ErrEnvNotFound
	}
	envs, err := c.Store.ListEnvironments(projectID)
	if err != nil {
		return err
	}
	if len(envs) <= 1 {
		return ErrEnvLast
	}
	apps, err := c.Store.ListAppsByEnvironment(envID)
	if err != nil {
		return err
	}
	if len(apps) > 0 {
		return ErrEnvHasServices
	}
	return c.Store.DeleteEnvironment(envID)
}

func (c *Core) SetDefaultEnvironment(projectID, envID string) error {
	env, err := c.Store.GetEnvironment(envID)
	if err != nil {
		return ErrEnvNotFound
	}
	if env.ProjectID != projectID {
		return ErrEnvNotFound
	}
	return c.Store.SetDefaultEnvironment(projectID, envID)
}

type EnvSummary struct {
	domain.Environment
	Apps       int    `json:"apps"`
	Status     string `json:"status"`
	LastDeploy string `json:"last_deploy"`
}

func (c *Core) EnvSummaries(projectID string) ([]EnvSummary, error) {
	envs, err := c.Store.ListEnvironments(projectID)
	if err != nil {
		return nil, err
	}
	out := make([]EnvSummary, 0, len(envs))
	for _, e := range envs {
		apps, err := c.Store.ListAppsByEnvironment(e.ID)
		if err != nil {
			return nil, err
		}
		summary := EnvSummary{Environment: e, Apps: len(apps), Status: "idle"}
		var lastDeploy time.Time
		for _, a := range apps {
			deploys, _ := c.Store.ListDeployments(a.ID, 1)
			if len(deploys) == 0 {
				continue
			}
			d := deploys[0]
			switch string(d.Status) {
			case "failed":
				summary.Status = "degraded"
			case "ready":
				if summary.Status != "degraded" {
					summary.Status = "healthy"
				}
			case "building", "queued", "starting", "health_checking":
				if summary.Status != "degraded" && summary.Status != "healthy" {
					summary.Status = "syncing"
				}
			}
			if !d.StartedAt.IsZero() && d.StartedAt.After(lastDeploy) {
				lastDeploy = d.StartedAt
			}
		}
		if !lastDeploy.IsZero() {
			summary.LastDeploy = humanAgo(lastDeploy)
		}
		out = append(out, summary)
	}
	return out, nil
}

func (c *Core) DefaultEnvironment(projectID string) (*domain.Environment, error) {
	envs, err := c.Store.ListEnvironments(projectID)
	if err != nil {
		return nil, err
	}
	for i := range envs {
		if envs[i].IsDefault {
			return &envs[i], nil
		}
	}
	if len(envs) > 0 {
		return &envs[0], nil
	}
	return nil, ErrEnvNotFound
}
