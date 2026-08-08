package core

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"aether/internal/domain"
)

// nextCron computes the next occurrence of a cron expression (5 fields: min hour dom mon dow)
// from a given time, with a 1-minute resolution and a horizon of 60 days.
func nextCron(cron string, from time.Time) (time.Time, error) {
	fields := strings.Fields(cron)
	if len(fields) != 5 {
		if cron == "@daily" {
			fields = []string{"0", "0", "*", "*", "*"}
		} else if cron == "@weekly" {
			fields = []string{"0", "0", "*", "*", "0"}
		} else if cron == "@hourly" {
			fields = []string{"0", "*", "*", "*", "*"}
		} else {
			return time.Time{}, fmt.Errorf("cron inválido: %q (use 'min hour dom mon dow' ou @daily/@weekly/@hourly)", cron)
		}
	}
	parse := func(f string) (map[int]bool, bool, error) {
		set := map[int]bool{}
		if f == "*" {
			return nil, true, nil
		}
		for _, part := range strings.Split(f, ",") {
			if strings.HasPrefix(part, "*/") {
				n, err := strconv.Atoi(strings.TrimPrefix(part, "*/"))
				if err != nil || n < 1 {
					return nil, false, fmt.Errorf("cron: passo inválido %q", part)
				}
				set[-n] = true
				continue
			}
			n, err := strconv.Atoi(part)
			if err != nil {
				return nil, false, fmt.Errorf("cron: campo inválido %q", part)
			}
			set[n] = true
		}
		return set, false, nil
	}
	minF, minAny, err := parse(fields[0])
	if err != nil {
		return time.Time{}, err
	}
	hourF, hourAny, err := parse(fields[1])
	if err != nil {
		return time.Time{}, err
	}
	domF, domAny, err := parse(fields[2])
	if err != nil {
		return time.Time{}, err
	}
	monF, monAny, err := parse(fields[3])
	if err != nil {
		return time.Time{}, err
	}
	dowF, dowAny, err := parse(fields[4])
	if err != nil {
		return time.Time{}, err
	}
	match := func(v int, set map[int]bool, any bool) bool {
		if any {
			return true
		}
		for k := range set {
			if k < 0 && v%(-k) == 0 {
				return true
			}
		}
		return set[v]
	}
	t := from.Add(1 * time.Minute).Truncate(time.Minute)
	horizon := from.Add(60 * 24 * time.Hour)
	for t.Before(horizon) {
		if match(int(t.Month()), monF, monAny) && match(t.Day(), domF, domAny) &&
			match(int(t.Weekday()), dowF, dowAny) && match(t.Hour(), hourF, hourAny) && match(t.Minute(), minF, minAny) {
			return t, nil
		}
		t = t.Add(1 * time.Minute)
	}
	return time.Time{}, fmt.Errorf("cron sem próxima ocorrência em 60 dias: %q", cron)
}

func (c *Core) CreateSnapshotSchedule(s *domain.SnapshotSchedule) error {
	if s.Retention <= 0 {
		s.Retention = 7
	}
	if _, err := nextCron(s.Cron, time.Now()); err != nil {
		return err
	}
	if s.ID == "" {
		s.ID = domain.NewID()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	next, _ := nextCron(s.Cron, s.CreatedAt)
	enabled := 0
	if s.Enabled {
		enabled = 1
	}
	_, err := c.DB.Exec(`INSERT INTO snapshot_schedules(id,org_id,app_id,volume,name_prefix,cron,retention,enabled,last_run,next_run,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		s.ID, s.OrgID, s.AppID, s.Volume, s.NamePrefix, s.Cron, s.Retention, enabled, "",
		next.UTC().Format(time.RFC3339), s.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (c *Core) ListSnapshotSchedules(orgID string) ([]domain.SnapshotSchedule, error) {
	rows, err := c.DB.Query(`SELECT id,org_id,app_id,volume,name_prefix,cron,retention,enabled,last_run,next_run,created_at FROM snapshot_schedules WHERE org_id=? ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.SnapshotSchedule{}
	for rows.Next() {
		var s domain.SnapshotSchedule
		var enabled int
		var lastRun, nextRun, created string
		if err := rows.Scan(&s.ID, &s.OrgID, &s.AppID, &s.Volume, &s.NamePrefix, &s.Cron, &s.Retention, &enabled, &lastRun, &nextRun, &created); err != nil {
			return nil, err
		}
		s.Enabled = enabled == 1
		s.LastRun, _ = time.Parse(time.RFC3339, lastRun)
		s.NextRun, _ = time.Parse(time.RFC3339, nextRun)
		s.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, s)
	}
	return out, rows.Err()
}

func (c *Core) DeleteSnapshotSchedule(id string) error {
	_, err := c.DB.Exec(`DELETE FROM snapshot_schedules WHERE id=?`, id)
	return err
}

func (c *Core) RunSnapshotScheduler(ctx context.Context) error {
	now := time.Now().UTC()
	orgs, err := c.Store.ListOrgs()
	if err != nil {
		return err
	}
	for _, org := range orgs {
		schedules, err := c.ListSnapshotSchedules(org.ID)
		if err != nil {
			continue
		}
		for _, s := range schedules {
			if !s.Enabled || s.NextRun.IsZero() || s.NextRun.After(now) {
				continue
			}
			name := s.NamePrefix
			if name == "" {
				name = "scheduled"
			}
			name += "-" + now.Format("20060102-150405")
			snap, err := c.CreateVolumeSnapshot(ctx, org.ID, s.AppID, s.Volume, name)
			if err != nil {
				continue
			}
			next, err := nextCron(s.Cron, now)
			if err != nil {
				continue
			}
			_, _ = c.DB.Exec(`UPDATE snapshot_schedules SET last_run=?, next_run=? WHERE id=?`,
				now.Format(time.RFC3339), next.UTC().Format(time.RFC3339), s.ID)
			c.notify.emit(org.ID, "backup.finished", "Scheduled snapshot created · "+s.AppID+" · "+snap.Name, map[string]any{
				"app_id": s.AppID, "snapshot": snap.Name, "volume": s.Volume,
			})
			c.enforceSnapshotRetention(ctx, s)
		}
	}
	return nil
}

func (c *Core) enforceSnapshotRetention(ctx context.Context, s domain.SnapshotSchedule) {
	rows, err := c.DB.Query(`SELECT id FROM snapshots WHERE app_id=? AND volume=? ORDER BY created_at DESC OFFSET ?`, s.AppID, s.Volume, s.Retention)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			_ = c.Store.DeleteSnapshot(id)
		}
	}
}

func (c *Core) StartSnapshotScheduler(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = c.RunSnapshotScheduler(context.Background())
			}
		}
	}()
}
