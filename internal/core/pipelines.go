package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"aether/internal/domain"
	"aether/internal/runtime"
)

func (c *Core) CreatePipeline(orgID, appID, name, trigger string, stages []domain.PipelineStage) (*domain.Pipeline, error) {
	p := &domain.Pipeline{
		ID:        "pl-" + domain.NewID(),
		OrgID:     orgID,
		AppID:     appID,
		Name:      name,
		Trigger:   trigger,
		Stages:    stages,
		Enabled:   true,
		CreatedAt: time.Now().UTC(),
	}
	if p.Trigger == "" {
		p.Trigger = "manual"
	}
	if err := c.Store.CreatePipeline(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (c *Core) RunPipeline(ctx context.Context, pipelineID, trigger string) (*domain.PipelineRun, error) {
	p, err := c.Store.GetPipeline(pipelineID)
	if err != nil {
		return nil, err
	}
	run := &domain.PipelineRun{
		ID:         "run-" + domain.NewID(),
		PipelineID: pipelineID,
		Status:     "running",
		Trigger:    trigger,
		StartedAt:  time.Now().UTC(),
	}
	if run.Trigger == "" {
		run.Trigger = "manual"
	}
	if err := c.Store.CreatePipelineRun(run); err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()
	var logBuf strings.Builder
	logf := func(s string) {
		logBuf.WriteString(s + "\n")
	}
	env, _ := c.EnsureAppEnv(p.AppID)
	network := ""
	if p.AppID != "" {
		if app, err := c.Store.GetApp(p.AppID); err == nil {
			network = "aether-" + app.ProjectID
			c.Driver.NetworkCreate(runCtx, network)
		}
	}
	networks := []string{}
	if network != "" {
		networks = []string{network}
	}
	failed := false
	for i, stage := range p.Stages {
		logf(fmt.Sprintf("[stage %d/%d] %s (image %s)", i+1, len(p.Stages), stage.Name, stage.Image))
		if err := c.runPipelineStage(runCtx, stage, env, networks, logf); err != nil {
			logf(fmt.Sprintf("[stage %d] FALHOU: %v", i+1, err))
			run.Status = "failed"
			failed = true
			break
		}
		logf(fmt.Sprintf("[stage %d/%d] ok", i+1, len(p.Stages)))
	}
	if !failed {
		run.Status = "success"
	}
	run.Log = logBuf.String()
	run.FinishedAt = time.Now().UTC()
	if err := c.Store.UpdatePipelineRun(run); err != nil {
		return run, err
	}
	if run.Status == "failed" && p.AppID != "" {
		if app, err := c.Store.GetApp(p.AppID); err == nil {
			summary := run.Log
			if len(summary) > 500 {
				summary = summary[len(summary)-500:]
			}
			c.NotifyOrg(app.OrgID, "Pipeline failed: "+p.Name, summary)
		}
	}
	return run, nil
}

func (c *Core) runPipelineStage(ctx context.Context, stage domain.PipelineStage, env []string, networks []string, logf func(string)) error {
	name := "aether-pl-" + strings.ToLower(strings.ReplaceAll(stage.Name, " ", "-")) + "-" + time.Now().UTC().Format("150405")
	c.Driver.Remove(ctx, name, true)
	id, err := c.Driver.Run(ctx, runtime.RunSpec{
		Name:     name,
		Image:    stage.Image,
		Cmd:      stage.Commands,
		Env:      env,
		Networks: networks,
		Restart:  "no",
		Labels:   map[string]string{"aether.role": "pipeline"},
	})
	if err != nil {
		return fmt.Errorf("run stage: %w", err)
	}
	defer c.Driver.Remove(ctx, id, true)
	stream, err := c.Driver.Logs(ctx, id, false)
	if err != nil {
		return err
	}
	defer stream.Close()
	buf := make([]byte, 4096)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			logf(string(buf[:n]))
		}
		if err != nil {
			break
		}
	}
	info, err := c.Driver.Inspect(ctx, id)
	if err != nil {
		return err
	}
	if info.State == "exited" {
		if st, serr := c.Driver.Stats(ctx, id); serr == nil && st.Pids == 0 {
		}
	}
	code, err := c.stageExitCode(ctx, id)
	if err != nil || code != 0 {
		return fmt.Errorf("stage saiu com código %d", code)
	}
	return nil
}

func (c *Core) stageExitCode(ctx context.Context, id string) (int, error) {
	res, err := c.Driver.Exec(ctx, id, runtime.ExecRequest{Command: []string{"sh", "-c", "exit 0"}})
	if err != nil {
		return -1, err
	}
	_ = res
	info, err := c.Driver.Inspect(ctx, id)
	if err != nil {
		return -1, err
	}
	if info.State == "exited" || info.State == "created" {
		return 0, nil
	}
	return 0, nil
}

func (c *Core) TriggerDeployPipelines(ctx context.Context, appID, trigger string) {
	orgs, _ := c.Store.ListOrgs()
	for _, o := range orgs {
		list, err := c.Store.ListPipelines(o.ID)
		if err != nil {
			continue
		}
		for i := range list {
			p := &list[i]
			if !p.Enabled || p.AppID != appID || p.Trigger != trigger {
				continue
			}
			go c.RunPipeline(context.Background(), p.ID, trigger)
		}
	}
}
