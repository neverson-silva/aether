package application

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/backups/domain"
	"aether/internal/platform/druntime/queue"
)

func TestBackupWorkerProcessesScheduledJobWithoutOrganization(t *testing.T) {
	service, store, _, queued, _ := newService()
	configurationID := uuid.New()
	databaseID := uuid.New()
	nextRun := time.Now().Add(time.Hour)
	store.config = &domain.BackupConfiguration{
		ID:            configurationID,
		DatabaseID:    databaseID,
		OrgID:         uuid.New(),
		Enabled:       true,
		DestinationID: uuid.New(),
		NextRunAt:     &nextRun,
		Schedule: domain.Schedule{
			Type:     domain.ScheduleCustom,
			Cron:     "*/15 * * * *",
			Timezone: "America/Sao_Paulo",
		},
		Retention: domain.Retention{Type: domain.RetentionAll},
	}
	payload, err := json.Marshal(struct {
		ConfigurationID string `json:"configuration_id"`
	}{ConfigurationID: configurationID.String()})
	if err != nil {
		t.Fatalf("marshal scheduled payload: %v", err)
	}

	worker := &BackupWorker{Service: service}
	err = worker.process(context.Background(), &queue.Job{
		ID:      "schedule:backups:" + configurationID.String(),
		Type:    "backup.schedule",
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("process scheduled backup: %v", err)
	}
	if len(store.jobs) != 1 {
		t.Fatalf("expected one scheduled backup job, got %d", len(store.jobs))
	}
	if len(queued.jobs) != 1 || queued.jobs[0].Type != "backup" {
		t.Fatalf("expected one backup job in the queue, got %+v", queued.jobs)
	}
	if store.nextRun == nil || !store.nextRun.After(time.Now()) {
		t.Fatal("expected the next scheduled run to advance")
	}
}
