package application

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"aether/internal/modules/snapshots/domain"
	"aether/internal/platform/druntime/locks"
	"aether/internal/platform/druntime/queue"
	"aether/internal/platform/druntime/scheduler"
	"aether/internal/platform/messaging"
	"aether/internal/platform/observability"
)

type ScheduleStore interface {
	ListEnabledSchedules(context.Context) ([]domain.Schedule, error)
	SetScheduleRun(context.Context, uuid.UUID, time.Time, time.Time) error
	CreateSnapshot(context.Context, *domain.Snapshot) (*domain.Snapshot, error)
	ListSnapshotsByOrg(context.Context, uuid.UUID, int) ([]domain.Snapshot, error)
	DeleteSnapshot(context.Context, uuid.UUID, uuid.UUID) error
}

type Executor interface {
	Create(context.Context, string, string) (string, int64, error)
}

type SnapshotWorker struct {
	Store       ScheduleStore
	Executor    Executor
	OutputDir   string
	Queue       queue.Queue
	Scheduler   scheduler.Scheduler
	Locks       locks.LockManager
	Metrics     *observability.Metrics
	Concurrency int
}

func (w *SnapshotWorker) concurrency() int {
	if w.Concurrency < 1 {
		return 1
	}
	return w.Concurrency
}

type snapshotSchedulePayload struct {
	ID    uuid.UUID `json:"id"`
	RunAt time.Time `json:"run_at"`
}

func (w *SnapshotWorker) Run(ctx context.Context) {
	if w.Queue == nil || w.Scheduler == nil || w.Executor == nil {
		return
	}
	consumer, err := w.Queue.NewConsumer(ctx, "snapshots", "snapshot-workers", "aether-snapshot")
	if err != nil {
		return
	}
	defer consumer.Close()
	for range w.concurrency() {
		go w.consumeLoop(ctx, consumer)
	}
	<-ctx.Done()
}

func (w *SnapshotWorker) consumeLoop(ctx context.Context, consumer queue.Consumer) {
	for {
		job, err := consumer.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		if job.Type != "snapshot.create" {
			_ = consumer.Ack(ctx, job)
			continue
		}
		var scheduled snapshotSchedulePayload
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

func (w *SnapshotWorker) Reconcile(ctx context.Context) error {
	schedules, err := w.Store.ListEnabledSchedules(ctx)
	if err != nil {
		return err
	}
	activeKeys := make([]string, 0, len(schedules))
	for _, schedule := range schedules {
		activeKeys = append(activeKeys, "snapshots:"+schedule.ID.String())
		now := time.Now()
		next, err := nextSnapshotRun(schedule.Cron, now)
		if err != nil {
			return fmt.Errorf("snapshot schedule %s: calculate next run: %w", schedule.ID, err)
		}
		if schedule.NextRun != nil && schedule.NextRun.After(now) {
			next = *schedule.NextRun
		}
		if err := w.Store.SetScheduleRun(ctx, schedule.ID, snapshotLastRun(schedule), next); err != nil {
			return fmt.Errorf("snapshot schedule %s: persist next run: %w", schedule.ID, err)
		}
		if err := w.schedule(ctx, schedule.ID, next); err != nil {
			return fmt.Errorf("snapshot schedule %s: publish schedule: %w", schedule.ID, err)
		}
	}
	if recurring, ok := w.Scheduler.(scheduler.RecurringScheduler); ok {
		return recurring.ReconcileRecurring(ctx, "snapshots", activeKeys)
	}
	return nil
}

func (w *SnapshotWorker) execute(ctx context.Context, id uuid.UUID, runAt time.Time) error {
	schedules, err := w.Store.ListEnabledSchedules(ctx)
	if err != nil {
		return err
	}
	var schedule *domain.Schedule
	for i := range schedules {
		if schedules[i].ID == id {
			schedule = &schedules[i]
			break
		}
	}
	if schedule == nil {
		return domain.ErrNotFound
	}
	if schedule.NextRun != nil && !runAt.IsZero() && !schedule.NextRun.Equal(runAt) {
		return nil
	}
	name := schedule.NamePrefix
	if name == "" {
		name = "snapshot"
	}
	path, size, err := w.Executor.Create(ctx, schedule.Volume, name)
	if err != nil {
		return err
	}
	_, err = w.Store.CreateSnapshot(ctx, &domain.Snapshot{ID: uuid.Nil, OrgID: schedule.OrgID, AppID: schedule.AppID, Volume: schedule.Volume, Name: filepath.Base(path), Size: size})
	if err != nil {
		return err
	}
	if err := w.applyRetention(ctx, *schedule); err != nil {
		return err
	}
	next, err := nextSnapshotRun(schedule.Cron, time.Now())
	if err != nil {
		return err
	}
	if err := w.Store.SetScheduleRun(ctx, schedule.ID, time.Now(), next); err != nil {
		return err
	}
	if _, ok := w.Scheduler.(scheduler.RecurringScheduler); ok {
		return nil
	}
	return w.schedule(ctx, schedule.ID, next)
}

func (w *SnapshotWorker) applyRetention(ctx context.Context, schedule domain.Schedule) error {
	if schedule.Retention < 1 {
		return nil
	}
	snapshots, err := w.Store.ListSnapshotsByOrg(ctx, schedule.OrgID, 1000)
	if err != nil {
		return err
	}
	matched := make([]domain.Snapshot, 0)
	for _, snapshot := range snapshots {
		if snapshot.Volume != schedule.Volume || (snapshot.AppID == nil) != (schedule.AppID == nil) || snapshot.AppID != nil && *snapshot.AppID != *schedule.AppID {
			continue
		}
		if schedule.NamePrefix != "" && len(snapshot.Name) >= len(schedule.NamePrefix) && snapshot.Name[:len(schedule.NamePrefix)] != schedule.NamePrefix {
			continue
		}
		matched = append(matched, snapshot)
	}
	for i := schedule.Retention; i < len(matched); i++ {
		if err := w.Store.DeleteSnapshot(ctx, matched[i].ID, schedule.OrgID); err != nil {
			return err
		}
		if remover, ok := w.Executor.(interface {
			Delete(context.Context, string) error
		}); ok {
			_ = remover.Delete(ctx, filepath.Join(w.OutputDir, matched[i].Name))
		}
	}
	return nil
}

func (w *SnapshotWorker) schedule(ctx context.Context, id uuid.UUID, at time.Time) error {
	schedules, err := w.Store.ListEnabledSchedules(ctx)
	if err != nil {
		return err
	}
	var current *domain.Schedule
	for i := range schedules {
		if schedules[i].ID == id {
			current = &schedules[i]
			break
		}
	}
	if current == nil {
		return domain.ErrNotFound
	}
	if recurring, ok := w.Scheduler.(scheduler.RecurringScheduler); ok {
		payload, err := json.Marshal(snapshotSchedulePayload{ID: id})
		if err != nil {
			return err
		}
		return recurring.ScheduleJobCron(ctx, messaging.Jobs("snapshots"), "snapshots:"+id.String(), "snapshot.create", nativeSnapshotCron(current.Cron), "", payload)
	}
	payload, err := json.Marshal(snapshotSchedulePayload{ID: id, RunAt: at})
	if err != nil {
		return err
	}
	return w.Scheduler.ScheduleJobAt(ctx, messaging.Jobs("snapshots"), id.String(), "snapshot.create", at, payload)
}

func nativeSnapshotCron(expression string) string {
	fields := strings.Fields(expression)
	if len(fields) == 5 {
		return "0 " + strings.Join(fields, " ")
	}
	return strings.Join(fields, " ")
}

func nextSnapshotRun(expression string, from time.Time) (time.Time, error) {
	schedule, err := cron.ParseStandard(expression)
	if err != nil {
		return time.Time{}, fmt.Errorf("snapshot schedule: %w", err)
	}
	return schedule.Next(from), nil
}

func snapshotLastRun(schedule domain.Schedule) time.Time {
	if schedule.LastRun != nil {
		return *schedule.LastRun
	}
	return time.Time{}
}
