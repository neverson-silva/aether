package application

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	appsdomain "aether/internal/modules/apps/domain"
	deploydomain "aether/internal/modules/deployments/domain"
	variablesApp "aether/internal/modules/variables/application"
	"aether/internal/platform/druntime/queue"
)

type Deployments struct {
	Store    deploydomain.Store
	Apps     AppStore
	Resolver *variablesApp.Resolver
	Queue    queue.Queue
	Notifier DeployNotifier
}

type DeployNotifier interface {
	NotifyDeploy(ctx context.Context, event deploydomain.DeployEvent)
}

type AppStore interface {
	GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error)
	ListEnvVars(ctx context.Context, appID uuid.UUID) ([]appsdomain.EnvVar, error)
	ListAppsByOrg(ctx context.Context, orgID uuid.UUID) ([]appsdomain.App, error)
}

type DeployOpts struct {
	Trigger     string
	TriggeredBy string
	CommitSHA   string
	ImageRef    string
}

func (d *Deployments) Deploy(ctx context.Context, appID, orgID uuid.UUID, opts DeployOpts) (*deploydomain.Deployment, error) {
	app, err := d.Apps.GetApp(ctx, appID, orgID)
	if err != nil {
		return nil, err
	}
	if opts.Trigger == "" {
		opts.Trigger = "api"
	}
	number, err := d.Store.NextNumber(ctx, appID)
	if err != nil {
		return nil, err
	}
	snapshot, err := d.envSnapshot(ctx, appID, orgID)
	if err != nil {
		return nil, err
	}
	spec, _ := json.Marshal(map[string]any{
		"name": app.Name, "port": app.Port, "mem_mb": app.MemMB, "cpus": app.CPUs,
		"image": app.Image, "git_url": app.GitURL, "git_branch": app.GitBranch, "upload_id": app.UploadID,
		"build_type": app.BuildType, "dockerfile": app.Dockerfile, "build_command": app.BuildCommand,
		"install_command": app.InstallCommand, "start_command": app.StartCommand,
		"root_folder": app.RootFolder, "dist_folder": app.DistFolder,
		"health_check": map[string]any{
			"enabled": app.HealthCheck.Enabled, "path": app.HealthCheck.Path,
			"interval_ms": app.HealthCheck.IntervalMS, "timeout_ms": app.HealthCheck.TimeoutMS,
			"retries": app.HealthCheck.Retries,
		},
	})
	dep, err := d.Store.CreateDeployment(ctx, &deploydomain.Deployment{
		AppID: appID, Number: number, Status: deploydomain.StatusQueued,
		Trigger: opts.Trigger, TriggeredBy: opts.TriggeredBy, CommitSHA: opts.CommitSHA,
		ImageRef: opts.ImageRef, EnvSnapshot: snapshot, DeploySpec: spec,
	})
	if err != nil {
		return nil, err
	}
	d.enqueue(ctx, dep, orgID)
	return dep, nil
}

func (d *Deployments) enqueue(ctx context.Context, dep *deploydomain.Deployment, orgID uuid.UUID) {
	if d.Queue != nil {
		_ = d.Queue.Enqueue(ctx, "deployments", queue.Job{
			DeploymentID: dep.ID.String(), AppID: dep.AppID.String(), OrgID: orgID.String(),
		})
	}
	if d.Notifier != nil {
		d.Notifier.NotifyDeploy(ctx, deploydomain.DeployEvent{
			AppID: dep.AppID, DepID: dep.ID, Status: string(deploydomain.StatusQueued),
		})
	}
}

func (d *Deployments) Rollback(ctx context.Context, appID, orgID uuid.UUID, by string) (*deploydomain.Deployment, error) {
	if _, err := d.Apps.GetApp(ctx, appID, orgID); err != nil {
		return nil, err
	}
	last, err := d.Store.LastReady(ctx, appID)
	if err != nil {
		return nil, err
	}
	if last.ImageRef == "" {
		return nil, deploydomain.ErrValidation
	}
	number, err := d.Store.NextNumber(ctx, appID)
	if err != nil {
		return nil, err
	}
	next, err := d.Store.CreateRollback(ctx, &deploydomain.Deployment{
		AppID: appID, Number: number, Status: deploydomain.StatusQueued,
		Trigger: "rollback", TriggeredBy: by, CommitSHA: last.CommitSHA, ImageRef: last.ImageRef,
		EnvSnapshot: last.EnvSnapshot, DeploySpec: last.DeploySpec, ComposeYAML: last.ComposeYAML,
		ComposeHash: last.ComposeHash,
	}, last.ID)
	if err != nil {
		return nil, err
	}
	d.enqueue(ctx, next, orgID)
	return next, nil
}

func (d *Deployments) Transition(ctx context.Context, appID, orgID, depID uuid.UUID, to deploydomain.Status) (*deploydomain.Deployment, error) {
	dep, err := d.Get(ctx, depID, orgID)
	if err != nil {
		return nil, err
	}
	if dep.AppID != appID {
		return nil, deploydomain.ErrForbidden
	}
	if err := dep.Transition(to); err != nil {
		return nil, err
	}
	if err := d.Store.UpdateStatus(ctx, dep.ID, dep.Status, dep.Error, dep.ImageRef, dep.ContainerID, dep.StartedAt, dep.FinishedAt); err != nil {
		return nil, err
	}
	return dep, nil
}

func (d *Deployments) Get(ctx context.Context, depID, orgID uuid.UUID) (*deploydomain.Deployment, error) {
	dep, err := d.Store.GetDeployment(ctx, depID)
	if err != nil {
		return nil, err
	}
	if _, err := d.Apps.GetApp(ctx, dep.AppID, orgID); err != nil {
		return nil, err
	}
	return dep, nil
}

func (d *Deployments) List(ctx context.Context, appID, orgID uuid.UUID, limit int) ([]deploydomain.Deployment, error) {
	if _, err := d.Apps.GetApp(ctx, appID, orgID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 25
	}
	return d.Store.ListByApp(ctx, appID, limit)
}

func (d *Deployments) envSnapshot(ctx context.Context, appID, orgID uuid.UUID) ([]byte, error) {
	var kv map[string]string
	var err error
	if d.Resolver != nil {
		kv, err = d.Resolver.Effective(ctx, appID, orgID)
	} else {
		var vars []appsdomain.EnvVar
		vars, err = d.Apps.ListEnvVars(ctx, appID)
		kv = make(map[string]string, len(vars))
		for _, v := range vars {
			kv[v.Name] = v.Value
		}
	}
	if err != nil {
		return nil, err
	}
	return json.Marshal(kv)
}
