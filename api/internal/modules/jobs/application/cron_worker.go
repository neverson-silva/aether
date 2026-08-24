package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"aether/internal/modules/jobs/domain"
	variablesApp "aether/internal/modules/variables/application"
	"aether/internal/platform/druntime/locks"
	"aether/internal/platform/druntime/queue"
	"aether/internal/platform/druntime/scheduler"
	"aether/internal/platform/messaging"
	"aether/internal/platform/observability"
)

type cronScheduleStore interface {
	ListEnabledCronJobs(context.Context) ([]domain.CronJob, error)
	GetCronJob(context.Context, uuid.UUID) (*domain.CronJob, error)
	SetCronRun(context.Context, uuid.UUID, time.Time, time.Time) error
}

type CronWorker struct {
	Store       cronScheduleStore
	Apps        AppStore
	Runtime     Runtime
	Resolver    *variablesApp.Resolver
	Queue       queue.Queue
	Scheduler   scheduler.Scheduler
	Locks       locks.LockManager
	Metrics     *observability.Metrics
	Concurrency int
}

func (w *CronWorker) concurrency() int {
	if w.Concurrency < 1 {
		return 1
	}
	return w.Concurrency
}

type cronSchedulePayload struct {
	ID    uuid.UUID `json:"id"`
	RunAt time.Time `json:"run_at"`
}

func (w *CronWorker) Run(ctx context.Context) {
	if w.Scheduler == nil || w.Queue == nil {
		return
	}
	consumer, err := w.Queue.NewConsumer(ctx, "cron", "cron-workers", "aether-cron")
	if err != nil {
		return
	}
	defer consumer.Close()
	for range w.concurrency() {
		go w.consumeLoop(ctx, consumer)
	}
	<-ctx.Done()
}

func (w *CronWorker) consumeLoop(ctx context.Context, consumer queue.Consumer) {
	for {
		job, err := consumer.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		if job.Type != "cron.execute" {
			_ = consumer.Ack(ctx, job)
			continue
		}
		var scheduled cronSchedulePayload
		if json.Unmarshal(job.Payload, &scheduled) != nil || scheduled.ID == uuid.Nil {
			_ = consumer.Ack(ctx, job)
			continue
		}
		stopProgress := queue.StartProgress(ctx, consumer, job)
		finish := func(bool) {}
		if w.Metrics != nil {
			finish = w.Metrics.StartJob(job.Type)
		}
		err = w.execute(ctx, scheduled.ID, scheduled.RunAt)
		stopProgress()
		if w.Metrics != nil {
			finish(err != nil)
		}
		if err != nil {
			_ = consumer.Nack(ctx, job)
			continue
		}
		_ = consumer.Ack(ctx, job)
	}
}

func (w *CronWorker) Reconcile(ctx context.Context) error {
	jobs, err := w.Store.ListEnabledCronJobs(ctx)
	if err != nil {
		return err
	}
	activeKeys := make([]string, 0, len(jobs))
	for _, job := range jobs {
		activeKeys = append(activeKeys, "cron:"+job.ID.String())
		now := time.Now()
		next, err := nextCron(job.Schedule, now)
		if err != nil {
			return fmt.Errorf("cron %s: calculate next run: %w", job.ID, err)
		}
		if job.NextRun != nil && job.NextRun.After(now) {
			next = *job.NextRun
		}
		if err := w.Store.SetCronRun(ctx, job.ID, lastRunValue(job), next); err != nil {
			return fmt.Errorf("cron %s: persist next run: %w", job.ID, err)
		}
		if err := w.schedule(ctx, job.ID, next); err != nil {
			return fmt.Errorf("cron %s: publish schedule: %w", job.ID, err)
		}
	}
	if recurring, ok := w.Scheduler.(scheduler.RecurringScheduler); ok {
		return recurring.ReconcileRecurring(ctx, "cron", activeKeys)
	}
	return nil
}

func (w *CronWorker) execute(ctx context.Context, id uuid.UUID, runAt time.Time) error {
	jobs, err := w.Store.ListEnabledCronJobs(ctx)
	if err != nil {
		return err
	}
	var job *domain.CronJob
	for i := range jobs {
		if jobs[i].ID == id {
			job = &jobs[i]
			break
		}
	}
	if job == nil {
		return domain.ErrNotFound
	}
	if job.NextRun != nil && !runAt.IsZero() && !job.NextRun.Equal(runAt) {
		return nil
	}
	app, err := w.Apps.GetApp(ctx, job.AppID, job.OrgID)
	if err != nil {
		return err
	}
	runner, ok := w.Runtime.(OneShotRuntime)
	if !ok {
		return fmt.Errorf("one-shot runtime is unavailable")
	}
	var env []string
	if w.Resolver != nil {
		values, err := w.Resolver.Effective(ctx, job.AppID, job.OrgID)
		if err != nil {
			return err
		}
		env = make([]string, 0, len(values))
		for key, value := range values {
			env = append(env, key+"="+value)
		}
	}
	if _, err := runner.RunOnce(ctx, "cron-"+job.ID.String()[:8], app.Image, job.Command, env); err != nil {
		return err
	}
	next, err := nextCron(job.Schedule, time.Now())
	if err != nil {
		return err
	}
	if err := w.Store.SetCronRun(ctx, job.ID, time.Now(), next); err != nil {
		return err
	}
	if _, ok := w.Scheduler.(scheduler.RecurringScheduler); ok {
		return nil
	}
	return w.schedule(ctx, job.ID, next)
}

func (w *CronWorker) schedule(ctx context.Context, id uuid.UUID, at time.Time) error {
	job, err := w.Store.GetCronJob(ctx, id)
	if err != nil {
		return err
	}
	if recurring, ok := w.Scheduler.(scheduler.RecurringScheduler); ok {
		payload, err := json.Marshal(cronSchedulePayload{ID: id})
		if err != nil {
			return err
		}
		return recurring.ScheduleJobCron(ctx, messaging.Jobs("cron"), "cron:"+id.String(), "cron.execute", nativeCron(job.Schedule), "", payload)
	}
	payload, err := json.Marshal(cronSchedulePayload{ID: id, RunAt: at})
	if err != nil {
		return err
	}
	return w.Scheduler.ScheduleJobAt(ctx, messaging.Jobs("cron"), id.String(), "cron.execute", at, payload)
}

func nativeCron(expression string) string {
	fields := strings.Fields(expression)
	if len(fields) == 5 {
		return "0 " + strings.Join(fields, " ")
	}
	return strings.Join(fields, " ")
}

func nextCron(expression string, from time.Time) (time.Time, error) {
	schedule, err := cron.ParseStandard(expression)
	if err != nil {
		return time.Time{}, err
	}
	return schedule.Next(from), nil
}

func lastRunValue(j domain.CronJob) time.Time {
	if j.LastRun != nil {
		return *j.LastRun
	}
	return time.Time{}
}
