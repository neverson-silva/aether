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

type Mirror struct {
	ID            uuid.UUID
	Name          string
	Source        string
	Dest          string
	DestTLSVerify bool
	TagsFilter    string
	Schedule      string
	LastRun       *time.Time
	Status        string
	CreatedAt     time.Time
}

type Store interface {
	CreateMirror(ctx context.Context, mirror *Mirror) (*Mirror, error)
	GetMirror(ctx context.Context, id uuid.UUID) (*Mirror, error)
	ListMirrors(ctx context.Context) ([]Mirror, error)
	SetStatus(ctx context.Context, id uuid.UUID, status string) error
	DeleteMirror(ctx context.Context, id uuid.UUID) error
}
