package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrLifecycleConflict = errors.New("service lifecycle transition conflicts with its current state")
	ErrNotFound          = errors.New("service lifecycle operation not found")
)

type LifecycleAction string

const (
	LifecycleStart LifecycleAction = "start"
	LifecycleStop  LifecycleAction = "stop"
)

type LifecycleOperation struct {
	ID         uuid.UUID
	ServiceID  uuid.UUID
	OrgID      uuid.UUID
	SpecID     uuid.UUID
	Kind       Kind
	Action     LifecycleAction
	Status     string
	Attempts   int
	LeaseUntil *time.Time
}

type LifecycleRequest struct {
	ServiceID uuid.UUID
	OrgID     uuid.UUID
	SpecID    uuid.UUID
	Kind      Kind
	Action    LifecycleAction
}
