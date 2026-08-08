package core

import (
	"aether/internal/domain"
	"testing"
	"time"
)

func TestNextCron(t *testing.T) {
	base := time.Date(2026, 8, 5, 10, 30, 0, 0, time.UTC)
	cases := []struct {
		cron string
		want string
	}{
		{"@daily", "2026-08-06T00:00:00Z"},
		{"@hourly", "2026-08-05T11:00:00Z"},
		{"0 3 * * *", "2026-08-06T03:00:00Z"},
		{"*/15 * * * *", "2026-08-05T10:45:00Z"},
		{"30 10 * * 1", "2026-08-10T10:30:00Z"},
	}
	for _, tc := range cases {
		got, err := nextCron(tc.cron, base)
		if err != nil {
			t.Fatalf("%s: %v", tc.cron, err)
		}
		if got.Format(time.RFC3339) != tc.want {
			t.Fatalf("%s: esperado %s, obtido %s", tc.cron, tc.want, got.Format(time.RFC3339))
		}
	}
	if _, err := nextCron("banana", base); err == nil {
		t.Fatal("cron inválido deveria falhar")
	}
}

func TestSnapshotScheduleCRUD(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	user, org, err := c.CreateUserAndOrg("snap@aether.local", "snap", "senha-snap")
	if err != nil {
		t.Fatal(err)
	}
	_ = user
	proj, _ := c.CreateProject(org.ID, "snap")
	envs, _ := c.ListEnvironments(proj.ID)
	app := &domain.App{ID: domain.NewID(), OrgID: org.ID, ProjectID: proj.ID, EnvironmentID: envs[0].ID, Name: "snapapp", SourceType: domain.SourceImage, Image: "nginx:alpine", Port: 80}
	if err := c.CreateApp(org.ID, app); err != nil {
		t.Fatal(err)
	}
	sched := &domain.SnapshotSchedule{
		OrgID: org.ID, AppID: app.ID, Volume: "aether-snapapp-data", NamePrefix: "nightly",
		Cron: "@daily", Retention: 7, Enabled: true,
	}
	if err := c.CreateSnapshotSchedule(sched); err != nil {
		t.Fatal(err)
	}
	all, err := c.ListSnapshotSchedules(org.ID)
	if err != nil || len(all) != 1 {
		t.Fatalf("schedules: %v %v", all, err)
	}
	if all[0].NextRun.IsZero() {
		t.Fatal("next_run não calculado")
	}
	if err := c.DeleteSnapshotSchedule(sched.ID); err != nil {
		t.Fatal(err)
	}
	all, _ = c.ListSnapshotSchedules(org.ID)
	if len(all) != 0 {
		t.Fatal("schedule não deletado")
	}
}
