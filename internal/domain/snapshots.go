package domain

import "time"

type SnapshotSchedule struct {
	ID         string    `json:"id"`
	OrgID      string    `json:"org_id"`
	AppID      string    `json:"app_id"`
	Volume     string    `json:"volume"`
	NamePrefix string    `json:"name_prefix"`
	Cron       string    `json:"cron"`
	Retention  int       `json:"retention"`
	Enabled    bool      `json:"enabled"`
	LastRun    time.Time `json:"last_run"`
	NextRun    time.Time `json:"next_run"`
	CreatedAt  time.Time `json:"created_at"`
}
