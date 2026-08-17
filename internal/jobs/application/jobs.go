package application

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	appsdomain "aether/internal/apps/domain"
	"aether/internal/jobs/domain"
	variablesApp "aether/internal/variables/application"
)

type Jobs struct {
	Store    domain.Store
	Apps     AppStore
	Resolver *variablesApp.Resolver
	Runtime  Runtime
}

type AppStore interface {
	GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error)
}

// Runtime orquestra containers de workers. Executa podman no host.
type Runtime interface {
	Run(ctx context.Context, name, image, command string, env []string) (string, error)
	Stop(ctx context.Context, containerID string) error
	Remove(ctx context.Context, containerID string) error
}

type podmanRuntime struct{}

func NewRuntime() Runtime { return podmanRuntime{} }

func (podmanRuntime) Run(ctx context.Context, name, image, command string, env []string) (string, error) {
	args := []string{"run", "-d", "--name", name}
	for _, e := range env {
		args = append(args, "-e", e)
	}
	args = append(args, image, "sh", "-c", command)
	out, err := exec.CommandContext(ctx, "podman", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (podmanRuntime) Stop(ctx context.Context, containerID string) error {
	_, err := exec.CommandContext(ctx, "podman", "stop", containerID).CombinedOutput()
	return err
}

func (podmanRuntime) Remove(ctx context.Context, containerID string) error {
	_, err := exec.CommandContext(ctx, "podman", "rm", "-f", containerID).CombinedOutput()
	return err
}

func (j *Jobs) CreateCronJob(ctx context.Context, appID, orgID uuid.UUID, name, schedule, command string) (*domain.CronJob, error) {
	if _, err := j.Apps.GetApp(ctx, appID, orgID); err != nil {
		return nil, err
	}
	if err := validateCron(name, schedule, command); err != nil {
		return nil, err
	}
	return j.Store.CreateCronJob(ctx, &domain.CronJob{
		AppID: appID, Name: strings.TrimSpace(name), Schedule: schedule,
		Command: strings.TrimSpace(command), Enabled: true,
	})
}

func (j *Jobs) ListCronJobs(ctx context.Context, appID, orgID uuid.UUID) ([]domain.CronJob, error) {
	if _, err := j.Apps.GetApp(ctx, appID, orgID); err != nil {
		return nil, err
	}
	return j.Store.ListCronJobsByApp(ctx, appID)
}

func (j *Jobs) ListAllCronJobs(ctx context.Context, orgID uuid.UUID) ([]domain.CronJob, error) {
	return j.Store.ListCronJobsByOrg(ctx, orgID)
}

func (j *Jobs) UpdateCronJob(ctx context.Context, id, orgID uuid.UUID, schedule, command *string, enabled *bool) (*domain.CronJob, error) {
	job, err := j.storeForOrg(ctx, id, orgID)
	if err != nil {
		return nil, err
	}
	if schedule != nil {
		if _, err := cron.ParseStandard(*schedule); err != nil {
			return nil, domain.ErrValidation
		}
		job.Schedule = *schedule
	}
	if command != nil {
		job.Command = *command
	}
	if enabled != nil {
		job.Enabled = *enabled
	}
	return j.Store.UpdateCronJob(ctx, job)
}

func (j *Jobs) DeleteCronJob(ctx context.Context, id, orgID uuid.UUID) error {
	if _, err := j.storeForOrg(ctx, id, orgID); err != nil {
		return err
	}
	return j.Store.DeleteCronJob(ctx, id)
}

func (j *Jobs) CreateWorker(ctx context.Context, appID, orgID uuid.UUID, name, command string, replicas int) (*domain.Worker, error) {
	if _, err := j.Apps.GetApp(ctx, appID, orgID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(command) == "" {
		return nil, domain.ErrValidation
	}
	if replicas < 1 || replicas > 20 {
		return nil, domain.ErrValidation
	}
	return j.Store.CreateWorker(ctx, &domain.Worker{
		AppID: appID, Name: strings.TrimSpace(name), Command: strings.TrimSpace(command),
		Replicas: replicas, Enabled: true, Status: "stopped",
	})
}

func (j *Jobs) ListWorkers(ctx context.Context, appID, orgID uuid.UUID) ([]domain.Worker, error) {
	if _, err := j.Apps.GetApp(ctx, appID, orgID); err != nil {
		return nil, err
	}
	return j.Store.ListWorkersByApp(ctx, appID)
}

func (j *Jobs) StartWorker(ctx context.Context, id, orgID uuid.UUID) error {
	worker, err := j.workerForOrg(ctx, id, orgID)
	if err != nil {
		return err
	}
	app, err := j.Apps.GetApp(ctx, worker.AppID, orgID)
	if err != nil {
		return err
	}
	command := strings.TrimSpace(worker.Command)
	if command == "" {
		return domain.ErrValidation
	}
	var env []string
	if j.Resolver != nil {
		effective, err := j.Resolver.Effective(ctx, worker.AppID, orgID)
		if err != nil {
			return err
		}
		env = make([]string, 0, len(effective))
		for k, v := range effective {
			env = append(env, k+"="+v)
		}
	}
	if j.Runtime == nil {
		return domain.ErrValidation
	}
	name := "worker-" + worker.ID.String()[:8]
	containerID, err := j.Runtime.Run(ctx, name, app.Image, command, env)
	if err != nil {
		return err
	}
	return j.Store.SetWorkerState(ctx, worker.ID, worker.AppID, "running", containerID)
}

func (j *Jobs) StopWorker(ctx context.Context, id, orgID uuid.UUID) error {
	worker, err := j.workerForOrg(ctx, id, orgID)
	if err != nil {
		return err
	}
	if worker.ContainerID != "" && j.Runtime != nil {
		_ = j.Runtime.Remove(ctx, worker.ContainerID)
	}
	return j.Store.SetWorkerState(ctx, worker.ID, worker.AppID, "stopped", "")
}

func (j *Jobs) DeleteWorker(ctx context.Context, id, orgID uuid.UUID) error {
	worker, err := j.workerForOrg(ctx, id, orgID)
	if err != nil {
		return err
	}
	if worker.ContainerID != "" && j.Runtime != nil {
		_ = j.Runtime.Remove(ctx, worker.ContainerID)
	}
	return j.Store.DeleteWorker(ctx, id)
}

func (j *Jobs) GetPolicy(ctx context.Context, appID, orgID uuid.UUID) (*domain.Policy, error) {
	if _, err := j.Apps.GetApp(ctx, appID, orgID); err != nil {
		return nil, err
	}
	policy, err := j.Store.GetPolicy(ctx, appID)
	if errors.Is(err, domain.ErrNotFound) {
		return &domain.Policy{
			AppID: appID, Enabled: false, CPUMin: 0.25, CPUMax: 4,
			MemMinMB: 128, MemMaxMB: 2048, ScaleUpPct: 80, ScaleDownPct: 15, CooldownMin: 15,
		}, nil
	}
	return policy, err
}

func (j *Jobs) SavePolicy(ctx context.Context, appID, orgID uuid.UUID, policy *domain.Policy) (*domain.Policy, error) {
	if _, err := j.Apps.GetApp(ctx, appID, orgID); err != nil {
		return nil, err
	}
	policy.AppID = appID
	if err := validatePolicy(policy); err != nil {
		return nil, err
	}
	return j.Store.SavePolicy(ctx, policy)
}

func (j *Jobs) PolicyEvents(ctx context.Context, appID, orgID uuid.UUID, limit int) ([]domain.AutopilotEvent, error) {
	if _, err := j.Apps.GetApp(ctx, appID, orgID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return j.Store.ListAutopilotEvents(ctx, appID, limit)
}

func (j *Jobs) storeForOrg(ctx context.Context, jobID, orgID uuid.UUID) (*domain.CronJob, error) {
	job, err := j.Store.GetCronJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if _, err := j.Apps.GetApp(ctx, job.AppID, orgID); err != nil {
		return nil, err
	}
	return job, nil
}

func (j *Jobs) workerForOrg(ctx context.Context, workerID, orgID uuid.UUID) (*domain.Worker, error) {
	worker, err := j.Store.GetWorker(ctx, workerID)
	if err != nil {
		return nil, err
	}
	if _, err := j.Apps.GetApp(ctx, worker.AppID, orgID); err != nil {
		return nil, err
	}
	return worker, nil
}

func validateCron(name, schedule, command string) error {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(command) == "" {
		return domain.ErrValidation
	}
	if _, err := cron.ParseStandard(schedule); err != nil {
		return domain.ErrValidation
	}
	return nil
}

func validatePolicy(p *domain.Policy) error {
	switch {
	case p.CPUMin <= 0, p.CPUMax < p.CPUMin:
		return domain.ErrValidation
	case p.MemMinMB < 0, p.MemMaxMB < p.MemMinMB:
		return domain.ErrValidation
	case p.ScaleUpPct < 1 || p.ScaleUpPct > 100, p.ScaleDownPct < 1 || p.ScaleDownPct > 100:
		return domain.ErrValidation
	case p.CooldownMin < 0:
		return domain.ErrValidation
	}
	return nil
}
