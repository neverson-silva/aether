package application

import (
	"testing"

	"aether/internal/modules/backups/domain"
)

func TestBackupScheduleCronHourly(t *testing.T) {
	got, ok := backupScheduleCron(domain.Schedule{Type: domain.ScheduleHourly, Minute: 15})
	if !ok || got != "0 15 * * * *" {
		t.Fatalf("unexpected hourly schedule: %q supported=%v", got, ok)
	}
}

func TestBackupScheduleCronWeekly(t *testing.T) {
	got, ok := backupScheduleCron(domain.Schedule{Type: domain.ScheduleWeekly, At: "03:20", DayOfWeek: "Monday"})
	if !ok || got != "0 20 3 * * 1" {
		t.Fatalf("unexpected weekly schedule: %q supported=%v", got, ok)
	}
}

func TestBackupScheduleCronBiweeklyUsesFallback(t *testing.T) {
	if _, ok := backupScheduleCron(domain.Schedule{Type: domain.ScheduleBiweekly, At: "03:20", StartDate: "2026-08-24"}); ok {
		t.Fatal("biweekly schedule should use one-shot fallback")
	}
}
