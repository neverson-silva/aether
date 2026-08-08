package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"aether/internal/domain"
	"aether/internal/git"
	"aether/internal/planner"
)

// analyzeSource materializes the app source (git clone or zip upload) into a
// temporary directory and returns it together with a cleanup func.
func (c *Core) analyzeSource(ctx context.Context, app *domain.App) (string, func(), error) {
	dir, err := os.MkdirTemp("", "aether-analyze-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(dir) }
	if app.UploadID != "" {
		uploadDir := filepath.Join(c.Cfg.BuildsDir, "uploads", app.UploadID)
		if err := copyDir(uploadDir, dir); err != nil {
			cleanup()
			return "", nil, err
		}
		flattenSingleRoot(dir)
		return dir, cleanup, nil
	}
	if err := git.Clone(ctx, app.GitURL, app.GitBranch, dir); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("git clone: %w", err)
	}
	return dir, cleanup, nil
}

// AnalyzeApp runs the planner over the app source and persists the plan.
func (c *Core) AnalyzeApp(ctx context.Context, appID string) (*domain.DeploymentPlan, error) {
	app, err := c.Store.GetApp(appID)
	if err != nil {
		return nil, err
	}
	srcDir, cleanup, err := c.analyzeSource(ctx, app)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	plan, err := planner.Detect(srcDir)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	dp := &domain.DeploymentPlan{
		ID:             domain.NewID(),
		AppID:          appID,
		Framework:      plan.Framework,
		Library:        plan.Library,
		PackageManager: plan.PackageManager,
		Runtime:        plan.Runtime,
		BuildCommand:   plan.BuildCommand,
		InstallCommand: plan.InstallCommand,
		OutputDir:      plan.OutputDir,
		AppType:        string(plan.AppType),
		WebServer:      plan.WebServer,
		ContainerPort:  plan.ContainerPort,
		SPAFallback:    plan.SPAFallback,
		IndexFile:      plan.IndexFile,
		NginxConf:      plan.NginxConf,
		Dockerfile:     plan.Dockerfile,
		Warnings:       plan.Warnings,
		DetectedAt:     now,
		CreatedAt:      now,
	}
	if err := c.Store.SaveDeploymentPlan(dp); err != nil {
		return nil, err
	}
	return dp, nil
}

func (c *Core) GetDeploymentPlan(appID string) (*domain.DeploymentPlan, error) {
	return c.Store.GetDeploymentPlan(appID)
}

// AnalyzeRepo runs the planner over an arbitrary git repo or zip upload (pre-create).
func (c *Core) AnalyzeRepo(ctx context.Context, gitURL, branch, uploadID string) (*planner.Plan, error) {
	if uploadID != "" {
		srcDir := filepath.Join(c.Cfg.BuildsDir, "uploads", uploadID)
		if _, err := os.Stat(srcDir); err != nil {
			return nil, fmt.Errorf("upload não encontrado")
		}
		return planner.Detect(srcDir)
	}
	if branch == "" {
		branch = "main"
	}
	dir, err := os.MkdirTemp("", "aether-analyze-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if err := git.Clone(ctx, gitURL, branch, dir); err != nil {
		return nil, fmt.Errorf("git clone: %w", err)
	}
	return planner.Detect(dir)
}
