package application

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"aether/internal/backups/domain"
)

func TestValidateSchedule(t *testing.T) {
	valid := []domain.Schedule{
		{Type: domain.ScheduleHourly, Minute: 0, Timezone: "UTC"},
		{Type: domain.ScheduleDaily, At: "03:00", Timezone: "America/Sao_Paulo"},
		{Type: domain.ScheduleWeekly, At: "03:00", DayOfWeek: "sunday", Timezone: "UTC"},
		{Type: domain.ScheduleBiweekly, At: "03:00", StartDate: "2026-08-20", Timezone: "UTC"},
		{Type: domain.ScheduleCustom, Cron: "0 3 * * *", Timezone: "UTC"},
	}
	for _, s := range valid {
		if err := ValidateSchedule(s); err != nil {
			t.Errorf("expected valid schedule %+v, got %v", s, err)
		}
	}

	invalid := []domain.Schedule{
		{Type: domain.ScheduleHourly, Minute: 60, Timezone: "UTC"},
		{Type: domain.ScheduleDaily, At: "25:00", Timezone: "UTC"},
		{Type: domain.ScheduleWeekly, At: "03:00", DayOfWeek: "funday", Timezone: "UTC"},
		{Type: domain.ScheduleBiweekly, At: "03:00", StartDate: "20-08-2026", Timezone: "UTC"},
		{Type: domain.ScheduleCustom, Cron: "0 3 * * nonsense", Timezone: "UTC"},
		{Type: domain.ScheduleDaily, At: "03:00", Timezone: "Not/AZone"},
		{Type: "bogus", Timezone: "UTC"},
	}
	for _, s := range invalid {
		if err := ValidateSchedule(s); err == nil {
			t.Errorf("expected invalid schedule %+v", s)
		}
	}
}

func TestNextRunDailyRespectsTimezone(t *testing.T) {
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	from := time.Date(2026, 8, 20, 12, 0, 0, 0, loc)
	next, err := NextRun(domain.Schedule{Type: domain.ScheduleDaily, At: "03:00", Timezone: "America/Sao_Paulo"}, from)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 8, 21, 3, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("next = %v, want %v", next, want)
	}
	_, nextOffset := next.Zone()
	_, wantOffset := want.Zone()
	if nextOffset != wantOffset {
		t.Fatalf("zone offset mismatch: %v vs %v", nextOffset, wantOffset)
	}
}

func TestNextRunDailyNextDay(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	from := time.Date(2026, 8, 20, 2, 0, 0, 0, loc)
	next, _ := NextRun(domain.Schedule{Type: domain.ScheduleDaily, At: "03:00", Timezone: "UTC"}, from)
	want := time.Date(2026, 8, 20, 3, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("next = %v, want %v", next, want)
	}
}

func TestNextRunHourly(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	from := time.Date(2026, 8, 20, 12, 30, 0, 0, loc)
	next, _ := NextRun(domain.Schedule{Type: domain.ScheduleHourly, Minute: 15, Timezone: "UTC"}, from)
	want := time.Date(2026, 8, 20, 13, 15, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("next = %v, want %v", next, want)
	}
}

func TestNextRunWeekly(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	// 2026-08-20 is a Thursday
	from := time.Date(2026, 8, 20, 12, 0, 0, 0, loc)
	next, _ := NextRun(domain.Schedule{Type: domain.ScheduleWeekly, At: "03:00", DayOfWeek: "sunday", Timezone: "UTC"}, from)
	want := time.Date(2026, 8, 23, 3, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("next = %v, want %v", next, want)
	}
}

func TestNextRunBiweeklyDeterministic(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	from := time.Date(2026, 8, 21, 12, 0, 0, 0, loc)
	next, _ := NextRun(domain.Schedule{Type: domain.ScheduleBiweekly, At: "03:00", StartDate: "2026-08-20", Timezone: "UTC"}, from)
	// start 08-20 03:00, step 15d: 09-04
	want := time.Date(2026, 9, 4, 3, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("next = %v, want %v", next, want)
	}
	// stepping is deterministic: next again from the result
	next2, _ := NextRun(domain.Schedule{Type: domain.ScheduleBiweekly, At: "03:00", StartDate: "2026-08-20", Timezone: "UTC"}, next)
	want2 := time.Date(2026, 9, 19, 3, 0, 0, 0, loc)
	if !next2.Equal(want2) {
		t.Fatalf("next2 = %v, want %v", next2, want2)
	}
}

func TestNextRunCustom(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	from := time.Date(2026, 8, 20, 12, 0, 0, 0, loc)
	next, _ := NextRun(domain.Schedule{Type: domain.ScheduleCustom, Cron: "0 3 * * *", Timezone: "UTC"}, from)
	want := time.Date(2026, 8, 21, 3, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("next = %v, want %v", next, want)
	}
}

func TestStorageKeyDeterministicAndSafe(t *testing.T) {
	dbID := uuid.New()
	bkpID := uuid.New()
	ts := time.Date(2026, 8, 20, 3, 0, 0, 0, time.UTC)

	key, err := StorageKey("databases/production", dbID, "postgres", bkpID, ts, "dump")
	if err != nil {
		t.Fatal(err)
	}
	expected := "databases/production/postgres/" + dbID.String() + "/backup-20260820T030000Z-" + bkpID.String() + ".dump"
	if key != expected {
		t.Fatalf("key = %q, want %q", key, expected)
	}

	key2, _ := StorageKey("databases/production", dbID, "postgres", bkpID, ts, "dump")
	if key != key2 {
		t.Fatalf("key not deterministic")
	}
}

func TestStorageKeyRejectsTraversal(t *testing.T) {
	dbID := uuid.New()
	bkpID := uuid.New()
	ts := time.Date(2026, 8, 20, 3, 0, 0, 0, time.UTC)
	for _, bad := range []string{"../etc", "a/../../b", "..", "a//b"} {
		if _, err := StorageKey(bad, dbID, "postgres", bkpID, ts, "dump"); err == nil {
			t.Errorf("expected rejection for prefix %q", bad)
		}
	}
	key, err := StorageKey("/abs", dbID, "postgres", bkpID, ts, "dump")
	if err != nil {
		t.Fatalf("leading slash should be normalized, got %v", err)
	}
	if !strings.HasPrefix(key, "abs/") {
		t.Fatalf("expected normalized prefix, got %q", key)
	}
}

func TestBackupJobStateMachine(t *testing.T) {
	j := &domain.BackupJob{Status: domain.BackupQueued}
	if err := j.Transition(domain.BackupPreparing); err != nil {
		t.Fatal(err)
	}
	if err := j.Transition(domain.BackupRunning); err != nil {
		t.Fatal(err)
	}
	if err := j.Transition(domain.BackupUploading); err != nil {
		t.Fatal(err)
	}
	if err := j.Transition(domain.BackupVerifying); err != nil {
		t.Fatal(err)
	}
	if err := j.Transition(domain.BackupCompleted); err != nil {
		t.Fatal(err)
	}
	if !j.Terminal() {
		t.Fatal("completed should be terminal")
	}
	if err := j.Transition(domain.BackupRunning); err == nil {
		t.Fatal("invalid transition should fail")
	}
}

func TestBackupJobFailureFromAny(t *testing.T) {
	for _, from := range []domain.BackupStatus{
		domain.BackupQueued, domain.BackupPreparing, domain.BackupRunning,
		domain.BackupUploading, domain.BackupVerifying,
	} {
		j := &domain.BackupJob{Status: from}
		if err := j.Transition(domain.BackupFailed); err != nil {
			t.Errorf("expected %s -> failed allowed, got %v", from, err)
		}
	}
}