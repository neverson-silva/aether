package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/backups/domain"
	"aether/internal/platform/druntime/queue"
	platformscheduler "aether/internal/platform/druntime/scheduler"
	"aether/internal/platform/messaging"
)

type BackupScheduler struct {
	Service *DatabaseBackups
}

func (s *DatabaseBackups) Reconcile(ctx context.Context) error {
	configs, err := s.Store.ListEnabledConfigurations(ctx)
	if err != nil {
		return err
	}
	activeKeys := make([]string, 0, len(configs))
	for _, cfg := range configs {
		if cfg.NextRunAt != nil {
			if _, ok := backupScheduleCron(cfg.Schedule); ok {
				activeKeys = append(activeKeys, "backups:"+cfg.ID.String())
			}
			if err := s.scheduleConfiguration(ctx, cfg); err != nil {
				return fmt.Errorf("backup configuration %s: publish schedule: %w", cfg.ID, err)
			}
		}
	}
	if recurring, ok := s.Scheduler.(platformscheduler.RecurringScheduler); ok {
		return recurring.ReconcileRecurring(ctx, "backups", activeKeys)
	}
	return nil
}

func (s *DatabaseBackups) scheduleConfiguration(ctx context.Context, cfg domain.BackupConfiguration) error {
	if s.Scheduler == nil || cfg.NextRunAt == nil {
		return nil
	}
	if recurring, ok := s.Scheduler.(platformscheduler.RecurringScheduler); ok {
		if expression, supported := backupScheduleCron(cfg.Schedule); supported {
			payload, err := json.Marshal(struct {
				ConfigurationID string `json:"configuration_id"`
			}{ConfigurationID: cfg.ID.String()})
			if err != nil {
				return err
			}
			return recurring.ScheduleJobCron(ctx, messaging.Jobs("backups"), "backups:"+cfg.ID.String(), "backup.schedule", expression, cfg.Schedule.Timezone, payload)
		}
	}
	payload, err := json.Marshal(struct {
		ConfigurationID string    `json:"configuration_id"`
		RunAt           time.Time `json:"run_at"`
	}{ConfigurationID: cfg.ID.String(), RunAt: *cfg.NextRunAt})
	if err != nil {
		return err
	}
	return s.Scheduler.ScheduleAt(ctx, cfg.ID.String(), *cfg.NextRunAt, payload)
}

func (s *DatabaseBackups) RunScheduled(ctx context.Context, payload []byte) error {
	var scheduled struct {
		ConfigurationID string    `json:"configuration_id"`
		RunAt           time.Time `json:"run_at"`
	}
	if err := json.Unmarshal(payload, &scheduled); err != nil {
		return err
	}
	id, err := uuid.Parse(scheduled.ConfigurationID)
	if err != nil {
		return err
	}
	configs, err := s.Store.ListEnabledConfigurations(ctx)
	if err != nil {
		return err
	}
	var cfg *domain.BackupConfiguration
	for i := range configs {
		if configs[i].ID == id {
			cfg = &configs[i]
			break
		}
	}
	if cfg == nil {
		return errors.New("backup configuration is not enabled")
	}
	if cfg.NextRunAt == nil || (!scheduled.RunAt.IsZero() && !cfg.NextRunAt.Equal(scheduled.RunAt)) {
		return nil
	}
	active, err := s.Store.ListActiveJobsByDatabase(ctx, cfg.DatabaseID)
	if err != nil {
		return err
	}
	if len(active) == 0 {
		job, err := s.Store.CreateJob(ctx, &domain.BackupJob{DatabaseID: cfg.DatabaseID, ServiceID: cfg.ServiceID, Configuration: &cfg.ID, Trigger: domain.TriggerScheduled, Status: domain.BackupQueued, DestinationID: cfg.DestinationID})
		if err != nil {
			return err
		}
		if err := s.enqueue(ctx, queue.Job{ID: job.ID.String(), Type: "backup", OrgID: cfg.OrgID.String(), Payload: []byte(job.ID.String())}); err != nil {
			return err
		}
	}
	s.advanceNextRun(ctx, cfg)
	if _, recurring := s.Scheduler.(platformscheduler.RecurringScheduler); recurring {
		if _, supported := backupScheduleCron(cfg.Schedule); supported {
			return nil
		}
	}
	return s.scheduleConfiguration(ctx, *cfg)
}

func backupScheduleCron(schedule domain.Schedule) (string, bool) {
	switch schedule.Type {
	case domain.ScheduleHourly:
		return fmt.Sprintf("0 %d * * * *", schedule.Minute), true
	case domain.ScheduleDaily:
		hour, minute, ok := parseBackupHHMM(schedule.At)
		if !ok {
			return "", false
		}
		return fmt.Sprintf("0 %d %d * * *", minute, hour), true
	case domain.ScheduleWeekly:
		hour, minute, ok := parseBackupHHMM(schedule.At)
		if !ok {
			return "", false
		}
		weekday := backupWeekdayIndex(schedule.DayOfWeek)
		if weekday < 0 {
			return "", false
		}
		return fmt.Sprintf("0 %d %d * * %d", minute, hour, weekday), true
	case domain.ScheduleCustom:
		fields := strings.Fields(schedule.Cron)
		if len(fields) == 5 {
			return "0 " + strings.Join(fields, " "), true
		}
		if len(fields) == 6 {
			return strings.Join(fields, " "), true
		}
	}
	return "", false
}

func parseBackupHHMM(value string) (int, int, bool) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	hour, errHour := strconv.Atoi(parts[0])
	minute, errMinute := strconv.Atoi(parts[1])
	if errHour != nil || errMinute != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, false
	}
	return hour, minute, true
}

func backupWeekdayIndex(value string) int {
	for index, weekday := range []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"} {
		if strings.EqualFold(value, weekday) {
			return index
		}
	}
	return -1
}

func (s *DatabaseBackups) advanceNextRun(ctx context.Context, cfg *domain.BackupConfiguration) {
	next, err := NextRun(cfg.Schedule, time.Now())
	if err != nil {
		next = time.Now().Add(24 * time.Hour)
	}
	_ = s.Store.SetConfigurationNextRun(ctx, cfg.ID, &next)
}
