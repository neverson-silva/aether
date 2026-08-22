package application

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	appsdomain "aether/internal/modules/apps/domain"
	deploydomain "aether/internal/modules/deployments/domain"
	"aether/internal/modules/specs/domain"
	"aether/internal/platform/worker"
)

type Specs struct {
	Apps        AppStore
	Deployments DeploymentStore
	Runtime     RuntimeReader
}

type AppStore interface {
	GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error)
	ListAppsByOrg(ctx context.Context, orgID uuid.UUID) ([]appsdomain.App, error)
	ListAppsByProject(ctx context.Context, orgID, projectID uuid.UUID) ([]appsdomain.App, error)
	GetProject(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.Project, error)
	ListProjects(ctx context.Context, orgID uuid.UUID) ([]appsdomain.Project, error)
	ListEnvironments(ctx context.Context, projectID uuid.UUID) ([]appsdomain.Environment, error)
	ListEnvVars(ctx context.Context, appID uuid.UUID) ([]appsdomain.EnvVar, error)
}

type DeploymentStore interface {
	ListByApp(ctx context.Context, appID uuid.UUID, limit int) ([]deploydomain.Deployment, error)
	GetDeployment(ctx context.Context, id uuid.UUID) (*deploydomain.Deployment, error)
}

type RuntimeReader interface {
	ContainerState(ctx context.Context, containerID string) (string, error)
	Stats(ctx context.Context, containerID string) (worker.ContainerStats, error)
}

func (s *Specs) AppSpec(ctx context.Context, appID, orgID uuid.UUID) (domain.Spec, error) {
	app, err := s.Apps.GetApp(ctx, appID, orgID)
	if err != nil {
		return domain.Spec{}, err
	}
	spec := domain.Spec{
		Name: app.Name, Image: app.Image, Port: app.Port, MemMB: app.MemMB,
		CPUs: app.CPUs, Replicas: 1,
		HealthCheck: domain.HealthCheck{
			Enabled: app.HealthCheck.Enabled, Path: app.HealthCheck.Path,
			IntervalMS: app.HealthCheck.IntervalMS, TimeoutMS: app.HealthCheck.TimeoutMS,
			Retries: app.HealthCheck.Retries,
		},
	}
	env, err := s.Apps.ListEnvVars(ctx, appID)
	if err == nil {
		spec.Env = make(map[string]string, len(env))
		for _, v := range env {
			spec.Env[v.Name] = v.Value
		}
	}
	return spec, nil
}

func (s *Specs) ExportCompose(spec domain.Spec) (string, error) {
	service := map[string]any{
		"image":   spec.Image,
		"restart": "unless-stopped",
	}
	if spec.Port > 0 {
		service["ports"] = []string{fmt.Sprintf("%d:%d", spec.Port, spec.Port)}
	}
	if spec.MemMB > 0 {
		service["mem_limit"] = fmt.Sprintf("%dm", spec.MemMB)
	}
	if len(spec.Env) > 0 {
		service["environment"] = spec.Env
	}
	if spec.HealthCheck.Enabled {
		service["healthcheck"] = map[string]any{
			"test":     []string{"CMD-SHELL", "curl -f http://localhost:" + itoa(spec.Port) + spec.HealthCheck.Path + " || exit 1"},
			"interval": fmt.Sprintf("%dms", spec.HealthCheck.IntervalMS),
			"timeout":  fmt.Sprintf("%dms", spec.HealthCheck.TimeoutMS),
			"retries":  spec.HealthCheck.Retries,
		}
	}
	compose := map[string]any{
		"version":  "3.8",
		"services": map[string]any{spec.Name: service},
	}
	raw, err := yaml.Marshal(compose)
	return string(raw), err
}

func (s *Specs) ExportKubernetes(spec domain.Spec) (string, error) {
	replicas := spec.Replicas
	if replicas <= 0 {
		replicas = 1
	}
	manifest := map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]any{"name": spec.Name},
		"spec": map[string]any{
			"replicas": replicas,
			"selector": map[string]any{"matchLabels": map[string]any{"app": spec.Name}},
			"template": map[string]any{
				"metadata": map[string]any{"labels": map[string]any{"app": spec.Name}},
				"spec": map[string]any{
					"containers": []map[string]any{{
						"name":  spec.Name,
						"image": spec.Image,
						"env":   envToK8s(spec.Env),
						"ports": []map[string]any{{"containerPort": spec.Port}},
					}},
				},
			},
		},
	}
	raw, err := yaml.Marshal(manifest)
	return string(raw), err
}

func (s *Specs) ExportNomad(spec domain.Spec) (string, error) {
	var env []string
	keys := make([]string, 0, len(spec.Env))
	for k := range spec.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		env = append(env, fmt.Sprintf("    %s = %q", k, spec.Env[k]))
	}
	job := fmt.Sprintf(`job "%s" {
  datacenters = ["dc1"]
  group "app" {
    network {
      port "http" {
        static = %d
      }
    }
    task "%s" {
      driver = "docker"
      config {
        image = %q
        ports = ["http"]
      }
      env {
%s
      }
    }
  }
}
`, spec.Name, spec.Port, spec.Name, spec.Image, strings.Join(env, "\n"))
	return job, nil
}

func (s *Specs) Compare(ctx context.Context, appID, orgID, depAID, depBID uuid.UUID) (domain.DeploymentDiff, error) {
	if _, err := s.Apps.GetApp(ctx, appID, orgID); err != nil {
		return domain.DeploymentDiff{}, err
	}
	depA, err := s.Deployments.GetDeployment(ctx, depAID)
	if err != nil {
		return domain.DeploymentDiff{}, err
	}
	depB, err := s.Deployments.GetDeployment(ctx, depBID)
	if err != nil {
		return domain.DeploymentDiff{}, err
	}
	if depA.AppID != appID || depB.AppID != appID {
		return domain.DeploymentDiff{}, domain.ErrValidation
	}
	envA := envMap(depA.EnvSnapshot)
	envB := envMap(depB.EnvSnapshot)
	diff := domain.DeploymentDiff{
		ImageA: depA.ImageRef, ImageB: depB.ImageRef,
		StatusA: string(depA.Status), StatusB: string(depB.Status),
		NumberA: depA.Number, NumberB: depB.Number,
		EnvChanged: map[string][2]string{},
	}
	for k := range envA {
		if _, ok := envB[k]; !ok {
			diff.EnvRemoved = append(diff.EnvRemoved, k)
		} else if envA[k] != envB[k] {
			diff.EnvChanged[k] = [2]string{envA[k], envB[k]}
		}
	}
	for k := range envB {
		if _, ok := envA[k]; !ok {
			diff.EnvAdded = append(diff.EnvAdded, k)
		}
	}
	sort.Strings(diff.EnvAdded)
	sort.Strings(diff.EnvRemoved)
	return diff, nil
}

func (s *Specs) SystemSummary(ctx context.Context, orgID uuid.UUID) (domain.SystemSummary, error) {
	out := domain.SystemSummary{Apps: []domain.AppSummary{}, Projects: []domain.ProjectRow{}}
	projects, err := s.Apps.ListProjects(ctx, orgID)
	if err != nil {
		return out, err
	}
	totalDeployments := 0
	totalApps := 0
	readyApps := 0
	var totalNet, totalIO uint64
	var cpuSum, memSum, memMax float64
	for _, p := range projects {
		apps, err := s.Apps.ListAppsByProject(ctx, orgID, p.ID)
		if err != nil {
			continue
		}
		row := domain.ProjectRow{ID: p.ID.String(), Name: p.Name, Apps: len(apps), Env: "—", Status: "idle", LastDeploy: "—"}
		var lastDeploy time.Time
		anyReady, anyBuilding, anyFailed := false, false, false
		for i := range apps {
			app := &apps[i]
			deps, err := s.Deployments.ListByApp(ctx, app.ID, 1)
			if err != nil || len(deps) == 0 {
				continue
			}
			dep := deps[0]
			totalDeployments++
			totalApps++
			switch dep.Status {
			case deploydomain.StatusReady:
				anyReady = true
				readyApps++
			case deploydomain.StatusQueued, deploydomain.StatusBuilding, deploydomain.StatusStarting, deploydomain.StatusHealthChecking:
				anyBuilding = true
			case deploydomain.StatusFailed:
				anyFailed = true
			}
			if dep.StartedAt != nil && dep.StartedAt.After(lastDeploy) {
				lastDeploy = *dep.StartedAt
			}
			if dep.ContainerID == "" {
				continue
			}
			if stats, err := s.Runtime.Stats(ctx, dep.ContainerID); err == nil && stats.MemUsage > 0 {
				cpuSum += stats.CPUPercent
				memSum += float64(stats.MemUsage)
				memMax += float64(stats.MemLimit)
				totalNet += stats.NetInput + stats.NetOutput
				totalIO += stats.BlockInput + stats.BlockOutput
				out.Apps = append(out.Apps, domain.AppSummary{
					ID: app.ID.String(), Name: app.Name, CPU: stats.CPUPercent,
					MemPct: pctOf(stats.MemUsage, stats.MemLimit),
					NetRx:  stats.NetInput, NetTx: stats.NetOutput,
				})
			}
		}
		switch {
		case anyFailed:
			row.Status, row.Env = "degraded", "Production"
		case anyBuilding:
			row.Status, row.Env = "syncing", "Staging"
		case anyReady:
			row.Status, row.Env = "healthy", "Production"
		}
		if !lastDeploy.IsZero() {
			row.LastDeploy = humanAgo(lastDeploy)
		}
		out.Projects = append(out.Projects, row)
	}
	appCount := len(out.Apps)
	if appCount > 0 {
		out.CPUPct = cpuSum / float64(appCount)
	}
	if memMax > 0 {
		out.MemPct = memSum / memMax * 100
	}
	if totalIO > 0 {
		out.IOPct = float64(totalIO) / float64(appCount) / (1024 * 1024 * 1024) * 100
		if out.IOPct > 100 {
			out.IOPct = 100
		}
	}
	out.Deployments = totalDeployments
	out.TrafficBytes = totalNet
	out.IOBytes = totalIO
	if totalApps > 0 {
		out.HealthPct = float64(readyApps) / float64(totalApps) * 100
	}
	sort.Slice(out.Apps, func(i, j int) bool { return out.Apps[i].Name < out.Apps[j].Name })
	return out, nil
}

func pctOf(used, limit uint64) float64 {
	if limit == 0 {
		return 0
	}
	return float64(used) / float64(limit) * 100
}

func humanAgo(t time.Time) string {
	secs := int(time.Since(t).Seconds())
	switch {
	case secs < 60:
		return "agora"
	case secs < 3600:
		return fmt.Sprintf("%dm", secs/60)
	case secs < 86400:
		return fmt.Sprintf("%dh", secs/3600)
	default:
		return fmt.Sprintf("%dd", secs/86400)
	}
}

func envMap(raw []byte) map[string]string {
	out := map[string]string{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

func envToK8s(env map[string]string) []map[string]any {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(env))
	for _, k := range keys {
		out = append(out, map[string]any{"name": k, "value": env[k]})
	}
	return out
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
