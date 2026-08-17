package application

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	appsdomain "aether/internal/apps/domain"
	deploydomain "aether/internal/deployments/domain"
	"aether/internal/worker"
)

type fakeSummaryAppStore struct {
	orgID    uuid.UUID
	projects []appsdomain.Project
	appsBy   map[uuid.UUID][]appsdomain.App
}

func (f fakeSummaryAppStore) GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error) {
	return nil, nil
}

func (f fakeSummaryAppStore) ListAppsByOrg(ctx context.Context, orgID uuid.UUID) ([]appsdomain.App, error) {
	return nil, nil
}

func (f fakeSummaryAppStore) ListAppsByProject(ctx context.Context, orgID, projectID uuid.UUID) ([]appsdomain.App, error) {
	return f.appsBy[projectID], nil
}

func (f fakeSummaryAppStore) GetProject(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.Project, error) {
	return nil, nil
}

func (f fakeSummaryAppStore) ListProjects(ctx context.Context, orgID uuid.UUID) ([]appsdomain.Project, error) {
	return f.projects, nil
}

func (f fakeSummaryAppStore) ListEnvironments(ctx context.Context, projectID uuid.UUID) ([]appsdomain.Environment, error) {
	return nil, nil
}

func (f fakeSummaryAppStore) ListEnvVars(ctx context.Context, appID uuid.UUID) ([]appsdomain.EnvVar, error) {
	return nil, nil
}

type fakeSummaryDeployStore struct {
	deps map[uuid.UUID][]deploydomain.Deployment
}

func (f fakeSummaryDeployStore) ListByApp(ctx context.Context, appID uuid.UUID, limit int) ([]deploydomain.Deployment, error) {
	return f.deps[appID], nil
}

func (f fakeSummaryDeployStore) GetDeployment(ctx context.Context, id uuid.UUID) (*deploydomain.Deployment, error) {
	return nil, nil
}

type fakeSummaryRuntime struct{}

func (fakeSummaryRuntime) ContainerState(ctx context.Context, containerID string) (string, error) {
	return "running", nil
}

func (fakeSummaryRuntime) Stats(ctx context.Context, containerID string) (worker.ContainerStats, error) {
	return worker.ContainerStats{CPUPercent: 1.5, MemUsage: 1024, MemLimit: 2048, NetInput: 100, NetOutput: 200}, nil
}

func TestSystemSummary(t *testing.T) {
	orgID := uuid.New()
	projID := uuid.New()
	appID := uuid.New()
	started := time.Now().Add(-time.Minute)
	s := &Specs{
		Apps: fakeSummaryAppStore{
			orgID:    orgID,
			projects: []appsdomain.Project{{ID: projID, OrgID: orgID, Name: "Proj"}},
			appsBy: map[uuid.UUID][]appsdomain.App{projID: {{
				ID: appID, OrgID: orgID, ProjectID: projID, Name: "web",
			}}},
		},
		Deployments: fakeSummaryDeployStore{deps: map[uuid.UUID][]deploydomain.Deployment{appID: {{
			ID: uuid.New(), AppID: appID, Number: 1, Status: deploydomain.StatusReady,
			ContainerID: "cid", StartedAt: &started,
		}}}},
		Runtime: fakeSummaryRuntime{},
	}
	out, err := s.SystemSummary(context.Background(), orgID)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if len(out.Projects) != 1 || out.Projects[0].Status != "healthy" {
		t.Fatalf("projects: %+v", out.Projects)
	}
	if out.Deployments != 1 {
		t.Fatalf("deployments: %d", out.Deployments)
	}
	if len(out.Apps) != 1 || out.Apps[0].Name != "web" {
		t.Fatalf("apps: %+v", out.Apps)
	}
	if out.TrafficBytes != 300 {
		t.Fatalf("traffic: %d", out.TrafficBytes)
	}
}

func TestSystemSummaryEmpty(t *testing.T) {
	orgID := uuid.New()
	s := &Specs{
		Apps:        fakeSummaryAppStore{orgID: orgID, projects: []appsdomain.Project{}, appsBy: map[uuid.UUID][]appsdomain.App{}},
		Deployments: fakeSummaryDeployStore{deps: map[uuid.UUID][]deploydomain.Deployment{}},
		Runtime:     fakeSummaryRuntime{},
	}
	out, err := s.SystemSummary(context.Background(), orgID)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if out.Projects == nil || out.Apps == nil {
		t.Fatalf("slices deveriam ser vazios não nil: %+v", out)
	}
}
