package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict")
	ErrValidation = errors.New("invalid input")
	ErrForbidden  = errors.New("access denied")
)

type Snapshot struct {
	ID         uuid.UUID
	OrgID      uuid.UUID
	AppID      *uuid.UUID
	Volume     string
	Name       string
	Size       int64
	Chunks     int
	DedupSaved int64
	CreatedAt  time.Time
}

type Schedule struct {
	ID         uuid.UUID
	OrgID      uuid.UUID
	AppID      *uuid.UUID
	Volume     string
	NamePrefix string
	Cron       string
	Retention  int
	Enabled    bool
	LastRun    *time.Time
	NextRun    *time.Time
	CreatedAt  time.Time
}

type Store interface {
	CreateSnapshot(ctx context.Context, snapshot *Snapshot) (*Snapshot, error)
	GetSnapshot(ctx context.Context, id uuid.UUID) (*Snapshot, error)
	ListSnapshotsByOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]Snapshot, error)
	DeleteSnapshot(ctx context.Context, id, orgID uuid.UUID) error

	CreateSchedule(ctx context.Context, schedule *Schedule) (*Schedule, error)
	GetSchedule(ctx context.Context, id uuid.UUID) (*Schedule, error)
	ListSchedulesByOrg(ctx context.Context, orgID uuid.UUID) ([]Schedule, error)
	DeleteSchedule(ctx context.Context, id, orgID uuid.UUID) error
}
