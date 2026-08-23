package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrConflict          = errors.New("conflict")
	ErrValidation        = errors.New("invalid input")
	ErrForbidden         = errors.New("access denied")
	ErrInvalidTransition = errors.New("invalid status transition")
)

type Status string

const (
	StatusQueued         Status = "queued"
	StatusBuilding       Status = "building"
	StatusStarting       Status = "starting"
	StatusHealthChecking Status = "health_checking"
	StatusReady          Status = "ready"
	StatusFailed         Status = "failed"
	StatusRolledBack     Status = "rolled_back"
	StatusCancelled      Status = "cancelled"
)

func (s Status) Valid() bool {
	switch s {
	case StatusQueued, StatusBuilding, StatusStarting, StatusHealthChecking,
		StatusReady, StatusFailed, StatusRolledBack, StatusCancelled:
		return true
	}
	return false
}

var terminal = map[Status]bool{
	StatusReady:      true,
	StatusFailed:     true,
	StatusRolledBack: true,
	StatusCancelled:  true,
}
var transitions = map[Status][]Status{
	StatusQueued:         {StatusBuilding, StatusFailed, StatusCancelled},
	StatusBuilding:       {StatusStarting, StatusFailed, StatusCancelled},
	StatusStarting:       {StatusHealthChecking, StatusFailed},
	StatusHealthChecking: {StatusReady, StatusFailed},
}

func (s Status) CanTransition(to Status) bool {
	if s == to {
		return false
	}
	if terminal[s] {
		return false
	}
	for _, next := range transitions[s] {
		if next == to {
			return true
		}
	}
	return false
}

func (s Status) Terminal() bool {
	return terminal[s]
}

type Deployment struct {
	ID          uuid.UUID
	AppID       uuid.UUID
	Number      int
	Status      Status
	Trigger     string
	TriggeredBy string
	CommitSHA   string
	ImageRef    string
	ContainerID string
	ServerID    string
	Error       string
	EnvSnapshot []byte
	ComposeYAML string
	DeploySpec  []byte
	ComposeHash string
	CreatedAt   time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
}

type DeployEvent struct {
	AppID  uuid.UUID `json:"app_id"`
	DepID  uuid.UUID `json:"deployment_id"`
	Status string    `json:"status"`
	Detail string    `json:"detail"`
}

func (d *Deployment) Transition(to Status) error {
	if !to.Valid() {
		return ErrValidation
	}
	if !d.Status.CanTransition(to) {
		return ErrInvalidTransition
	}
	d.Status = to
	now := time.Now().UTC()
	if to == StatusBuilding && d.StartedAt == nil {
		d.StartedAt = &now
	}
	if to.Terminal() || to == StatusReady {
		d.FinishedAt = &now
	}
	return nil
}

type Store interface {
	CreateDeployment(ctx context.Context, dep *Deployment) (*Deployment, error)
	GetDeployment(ctx context.Context, id uuid.UUID) (*Deployment, error)
	GetByApp(ctx context.Context, appID uuid.UUID, number int) (*Deployment, error)
	ListByApp(ctx context.Context, appID uuid.UUID, limit int) ([]Deployment, error)
	ListQueued(ctx context.Context) ([]Deployment, error)
	ListReady(ctx context.Context) ([]Deployment, error)
	NextNumber(ctx context.Context, appID uuid.UUID) (int, error)
	LastReady(ctx context.Context, appID uuid.UUID) (*Deployment, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status, errMsg, imageRef, containerID string, startedAt, finishedAt *time.Time) error
	MarkRolledBack(ctx context.Context, id uuid.UUID) error
	CreateRollback(ctx context.Context, newDep *Deployment, rolledBackID uuid.UUID) (*Deployment, error)
}
