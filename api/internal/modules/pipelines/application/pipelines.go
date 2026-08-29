package application

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/uuid"

	appsdomain "aether/internal/modules/apps/domain"
	"aether/internal/modules/pipelines/domain"
)

type Pipelines struct {
	Store    domain.Store
	Apps     AppStore
	Services ServiceStore
	StageRunner
}

type AppStore interface {
	GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error)
}

type ServiceStore interface {
	GetService(ctx context.Context, id, orgID uuid.UUID) error
}

type ServiceStoreFunc func(context.Context, uuid.UUID, uuid.UUID) error

func (f ServiceStoreFunc) GetService(ctx context.Context, id, orgID uuid.UUID) error {
	return f(ctx, id, orgID)
}

type StageRunner interface {
	RunStage(ctx context.Context, image string, commands []string) (output string, err error)
}

type PodmanStageRunner struct{}

func (PodmanStageRunner) RunStage(ctx context.Context, image string, commands []string) (string, error) {
	script := strings.Join(commands, "\n")
	args := []string{"run", "--rm", image, "sh", "-c", script}
	out, err := exec.CommandContext(ctx, "podman", args...).CombinedOutput()
	return string(out), err
}

func (p *Pipelines) Create(ctx context.Context, orgID uuid.UUID, appID *uuid.UUID, name, trigger string, stages []domain.Stage) (*domain.Pipeline, error) {
	return p.create(ctx, orgID, appID, nil, name, trigger, stages)
}

func (p *Pipelines) CreateForService(ctx context.Context, orgID, serviceID uuid.UUID, name, trigger string, stages []domain.Stage) (*domain.Pipeline, error) {
	if p.Services == nil {
		return nil, domain.ErrValidation
	}
	if err := p.Services.GetService(ctx, serviceID, orgID); err != nil {
		return nil, err
	}
	return p.create(ctx, orgID, nil, &serviceID, name, trigger, stages)
}

func (p *Pipelines) create(ctx context.Context, orgID uuid.UUID, appID, serviceID *uuid.UUID, name, trigger string, stages []domain.Stage) (*domain.Pipeline, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(stages) == 0 {
		return nil, domain.ErrValidation
	}
	if appID != nil {
		if _, err := p.Apps.GetApp(ctx, *appID, orgID); err != nil {
			return nil, err
		}
	}
	if trigger == "" {
		trigger = "manual"
	}
	if trigger != "manual" && trigger != "auto" && trigger != "webhook" {
		return nil, domain.ErrValidation
	}
	return p.Store.CreatePipeline(ctx, &domain.Pipeline{
		OrgID: orgID, AppID: appID, ServiceID: serviceID, Name: name, Trigger: trigger, Stages: stages, Enabled: true,
	})
}

func (p *Pipelines) List(ctx context.Context, orgID uuid.UUID) ([]domain.Pipeline, error) {
	return p.Store.ListPipelinesByOrg(ctx, orgID)
}

func (p *Pipelines) Delete(ctx context.Context, pipelineID, orgID uuid.UUID) error {
	return p.Store.DeletePipeline(ctx, pipelineID, orgID)
}

func (p *Pipelines) Run(ctx context.Context, pipelineID, orgID uuid.UUID, trigger string) (*domain.Run, error) {
	pipeline, err := p.Store.GetPipeline(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	if pipeline.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	if trigger == "" {
		trigger = "manual"
	}
	run, err := p.Store.CreateRun(ctx, &domain.Run{PipelineID: pipelineID, Status: "running", Trigger: trigger})
	if err != nil {
		return nil, err
	}
	var log strings.Builder
	status := "success"
	for _, stage := range pipeline.Stages {
		fmt.Fprintf(&log, "== stage %s (%s) ==\n", stage.Name, stage.Image)
		out, stageErr := p.RunStage(ctx, stage.Image, stage.Commands)
		log.WriteString(out)
		if !strings.HasSuffix(out, "\n") {
			log.WriteString("\n")
		}
		if stageErr != nil {
			fmt.Fprintf(&log, "stage %s failed: %v\n", stage.Name, stageErr)
			status = "failed"
			break
		}
	}
	if err := p.Store.FinishRun(ctx, run.ID, status, log.String()); err != nil {
		return nil, err
	}
	run.Status = status
	run.Log = log.String()
	return run, nil
}

func (p *Pipelines) ListRuns(ctx context.Context, pipelineID, orgID uuid.UUID, limit int) ([]domain.Run, error) {
	pipeline, err := p.Store.GetPipeline(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	if pipeline.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	return p.Store.ListRuns(ctx, pipelineID, limit)
}
