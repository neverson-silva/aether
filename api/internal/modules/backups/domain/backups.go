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

type Backup struct {
	ID         uuid.UUID
	OrgID      uuid.UUID
	DatabaseID *uuid.UUID
	AppID      *uuid.UUID
	Path       string
	Size       int64
	Kind       string
	Dest       string
	CreatedAt  time.Time
}

type Store interface {
	CreateBackup(ctx context.Context, backup *Backup) (*Backup, error)
	GetBackup(ctx context.Context, id uuid.UUID) (*Backup, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]Backup, error)
	ListByDatabase(ctx context.Context, databaseID uuid.UUID, limit int) ([]Backup, error)
}
