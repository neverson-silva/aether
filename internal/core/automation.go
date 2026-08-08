package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"aether/internal/domain"
	"aether/internal/runtime"
)

func (c *Core) CreateCronJob(appID, name, schedule, command string) (*domain.CronJob, error) {
	app, err := c.Store.GetApp(appID)
	if err != nil {
		return nil, err
	}
	if _, err := cron.ParseStandard(schedule); err != nil {
		return nil, fmt.Errorf("schedule inválida: %w", err)
	}
	now := time.Now().UTC()
	j := &domain.CronJob{
		ID:        domain.NewID(),
		AppID:     appID,
		Name:      name,
		Schedule:  schedule,
		Command:   command,
		Enabled:   true,
		NextRun:   c.nextCron(schedule, now),
		CreatedAt: now,
	}
	if err := c.Store.CreateCronJob(j); err != nil {
		return nil, err
	}
	_ = app
	return j, nil
}

func (c *Core) nextCron(schedule string, from time.Time) time.Time {
	sch, err := cron.ParseStandard(schedule)
	if err != nil {
		return from.Add(time.Minute)
	}
	return sch.Next(from).UTC()
}

func (c *Core) UpdateCronJob(id, schedule, command string, enabled bool) (*domain.CronJob, error) {
	j, err := c.Store.GetCronJob(id)
	if err != nil {
		return nil, err
	}
	if schedule != "" {
		if _, err := cron.ParseStandard(schedule); err != nil {
			return nil, fmt.Errorf("schedule inválida: %w", err)
		}
		j.Schedule = schedule
	}
	if command != "" {
		j.Command = command
	}
	j.Enabled = enabled
	now := time.Now().UTC()
	if j.Enabled && (j.NextRun.IsZero() || j.NextRun.Before(now)) {
		j.NextRun = c.nextCron(j.Schedule, now)
	}
	if err := c.Store.UpdateCronJob(j); err != nil {
		return nil, err
	}
	return j, nil
}

func (c *Core) StartScheduler(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(15 * time.Second):
				c.schedulerTick(ctx)
			}
		}
	}()
}

func (c *Core) schedulerTick(ctx context.Context) {
	apps, err := c.Store.ListAllApps()
	if err != nil {
		return
	}
	now := time.Now().UTC()
	for _, app := range apps {
		jobs, err := c.Store.ListCronJobs(app.ID)
		if err != nil {
			continue
		}
		for i := range jobs {
			j := &jobs[i]
			if !j.Enabled {
				continue
			}
			if j.NextRun.IsZero() || !now.After(j.NextRun) {
				continue
			}
			j.LastRun = now
			j.NextRun = c.nextCron(j.Schedule, now)
			c.Store.UpdateCronJob(j)
			go c.runCronJob(&app, j)
		}
	}
}

func (c *Core) runCronJob(app *domain.App, j *domain.CronJob) {
	c.withLockSkip("lock:cron:"+j.ID, lockCronTTL, func() {
		c.executeCronJob(app, j)
	})
}

func (c *Core) executeCronJob(app *domain.App, j *domain.CronJob) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	image, err := c.appImageFor(app)
	if err != nil {
		log.Printf("[cron] %s/%s: %v", app.Name, j.Name, err)
		return
	}
	env, err := c.EnsureAppEnv(app.ID)
	if err != nil {
		log.Printf("[cron] %s/%s env: %v", app.Name, j.Name, err)
		return
	}
	name := fmt.Sprintf("aether-%s-cron-%s", app.Name, j.Name)
	c.Driver.Remove(ctx, name, true)
	network := "aether-" + app.ProjectID
	c.Driver.NetworkCreate(ctx, network)
	spec := runtime.RunSpec{
		Name:     name,
		Image:    image,
		Cmd:      strings.Fields(j.Command),
		Env:      env,
		Networks: []string{network},
		Restart:  "no",
		Labels:   map[string]string{"aether.app": app.ID, "aether.cron": j.ID},
	}
	id, err := c.Driver.Run(ctx, spec)
	if err != nil {
		log.Printf("[cron] %s/%s run: %v", app.Name, j.Name, err)
		return
	}
	defer c.Driver.Remove(ctx, id, true)
	stream, err := c.Driver.Logs(ctx, id, false)
	if err != nil {
		return
	}
	defer stream.Close()
	dir := filepath.Join(c.Cfg.LogsDir, "apps", app.Name)
	os.MkdirAll(dir, 0o750)
	logPath := filepath.Join(dir, "cron-"+j.Name+".log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "\n=== cron %s %s ===\n", j.Name, time.Now().UTC().Format(time.RFC3339))
	ioCopy(f, stream)
}

func (c *Core) appImageFor(app *domain.App) (string, error) {
	if app.SourceType == domain.SourceImage {
		return app.Image, nil
	}
	deploys, err := c.Store.ListDeployments(app.ID, 1)
	if err != nil || len(deploys) == 0 {
		return "", fmt.Errorf("sem deployment para %s", app.Name)
	}
	if deploys[0].ImageRef == "" {
		return "", fmt.Errorf("sem imagem para %s", app.Name)
	}
	return deploys[0].ImageRef, nil
}

func (c *Core) CreateWorker(appID, name, command string, replicas int) (*domain.Worker, error) {
	if replicas < 1 {
		replicas = 1
	}
	w := &domain.Worker{
		ID:        domain.NewID(),
		AppID:     appID,
		Name:      name,
		Command:   command,
		Replicas:  replicas,
		Enabled:   true,
		Status:    "stopped",
		CreatedAt: time.Now().UTC(),
	}
	if err := c.Store.CreateWorker(w); err != nil {
		return nil, err
	}
	if err := c.StartWorker(w.ID); err != nil {
		return w, err
	}
	return w, nil
}

func (c *Core) StartWorker(id string) error {
	w, err := c.Store.GetWorker(id)
	if err != nil {
		return err
	}
	app, err := c.Store.GetApp(w.AppID)
	if err != nil {
		return err
	}
	image, err := c.appImageFor(app)
	if err != nil {
		return err
	}
	env, err := c.EnsureAppEnv(app.ID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	network := "aether-" + app.ProjectID
	c.Driver.NetworkCreate(ctx, network)
	name := fmt.Sprintf("aether-%s-worker-%s", app.Name, w.Name)
	c.Driver.Remove(ctx, name, true)
	spec := runtime.RunSpec{
		Name:     name,
		Image:    image,
		Cmd:      strings.Fields(w.Command),
		Env:      env,
		Networks: []string{network},
		Restart:  "always",
		Labels:   map[string]string{"aether.app": app.ID, "aether.worker": w.ID},
	}
	cid, err := c.Driver.Run(ctx, spec)
	if err != nil {
		return err
	}
	w.Status = "running"
	w.ContainerID = cid
	return c.Store.UpdateWorker(w)
}

func (c *Core) StopWorker(id string) error {
	w, err := c.Store.GetWorker(id)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if w.ContainerID != "" {
		c.Driver.Remove(ctx, w.ContainerID, true)
	}
	w.Status = "stopped"
	w.ContainerID = ""
	return c.Store.UpdateWorker(w)
}

func (c *Core) GetWorker(id string) (*domain.Worker, error) {
	return c.Store.GetWorker(id)
}

func (c *Core) ListWorkers(appID string) ([]domain.Worker, error) {
	return c.Store.ListWorkers(appID)
}

func (c *Core) DeleteWorker(id string) error {
	c.StopWorker(id)
	return c.Store.DeleteWorker(id)
}
