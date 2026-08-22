package domain

import (
	"time"

	"github.com/google/uuid"
)

type ScheduleType string

const (
	ScheduleHourly   ScheduleType = "hourly"
	ScheduleDaily    ScheduleType = "daily"
	ScheduleWeekly   ScheduleType = "weekly"
	ScheduleBiweekly ScheduleType = "biweekly"
	ScheduleCustom   ScheduleType = "custom"
)

type RetentionType string

const (
	RetentionAll    RetentionType = "all"
	RetentionLatest RetentionType = "latest"
)

type Schedule struct {
	Type      ScheduleType
	Minute    int
	At        string
	DayOfWeek string
	StartDate string
	Cron      string
	Timezone  string
}

type Retention struct {
	Type RetentionType
}

type BackupConfiguration struct {
	ID            uuid.UUID
	DatabaseID    uuid.UUID
	OrgID         uuid.UUID
	Enabled       bool
	DestinationID uuid.UUID
	PathPrefix    string
	Schedule      Schedule
	Retention     Retention
	NextRunAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
