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

type CronJob struct {
	ID          uuid.UUID
	OrgID       uuid.UUID
	AppID       uuid.UUID
	ServiceID   uuid.UUID
	ServiceName string
	Name        string
	Schedule    string
	Command     string
	Enabled     bool
	LastRun     *time.Time
	NextRun     *time.Time
	CreatedAt   time.Time
}

type Worker struct {
	ID          uuid.UUID
	AppID       uuid.UUID
	ServiceID   uuid.UUID
	Name        string
	Command     string
	Replicas    int
	Enabled     bool
	Status      string
	ContainerID string
	CreatedAt   time.Time
}

type Policy struct {
	AppID        uuid.UUID
	ServiceID    uuid.UUID
	Enabled      bool
	CPUMin       float64
	CPUMax       float64
	MemMinMB     int
	MemMaxMB     int
	ScaleUpPct   int
	ScaleDownPct int
	CooldownMin  int
	UpdatedAt    time.Time
}

type AutopilotEvent struct {
	ID        uuid.UUID
	AppID     uuid.UUID
	ServiceID uuid.UUID
	Action    string
	Detail    string
	CreatedAt time.Time
}

type Store interface {
	CreateCronJob(ctx context.Context, job *CronJob) (*CronJob, error)
	GetCronJob(ctx context.Context, id uuid.UUID) (*CronJob, error)
	ListCronJobsByApp(ctx context.Context, appID uuid.UUID) ([]CronJob, error)
	ListCronJobsByOrg(ctx context.Context, orgID uuid.UUID) ([]CronJob, error)
	UpdateCronJob(ctx context.Context, job *CronJob) (*CronJob, error)
	DeleteCronJob(ctx context.Context, id uuid.UUID) error

	CreateWorker(ctx context.Context, worker *Worker) (*Worker, error)
	GetWorker(ctx context.Context, id uuid.UUID) (*Worker, error)
	ListWorkersByApp(ctx context.Context, appID uuid.UUID) ([]Worker, error)
	SetWorkerState(ctx context.Context, id, appID uuid.UUID, status, containerID string) error
	DeleteWorker(ctx context.Context, id uuid.UUID) error

	GetPolicy(ctx context.Context, appID uuid.UUID) (*Policy, error)
	SavePolicy(ctx context.Context, policy *Policy) (*Policy, error)
	CreateAutopilotEvent(ctx context.Context, appID uuid.UUID, action, detail string) error
	ListAutopilotEvents(ctx context.Context, appID uuid.UUID, limit int) ([]AutopilotEvent, error)
}
