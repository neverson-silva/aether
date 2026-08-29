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

type Volume struct {
	ID        uuid.UUID
	AppID     uuid.UUID
	ServiceID *uuid.UUID
	Name      string
	MountPath string
}

type Backup struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	AppID     *uuid.UUID
	ServiceID *uuid.UUID
	Path      string
	Size      int64
	Kind      string
	Dest      string
	CreatedAt time.Time
}

type Store interface {
	GetVolumeByApp(ctx context.Context, appID uuid.UUID, name string) (*Volume, error)
	GetVolumeByService(ctx context.Context, serviceID uuid.UUID, name string) (*Volume, error)
	ListVolumesByApp(ctx context.Context, appID uuid.UUID) ([]Volume, error)
	ListVolumesByService(ctx context.Context, serviceID uuid.UUID) ([]Volume, error)
	CreateVolume(ctx context.Context, volume *Volume) (*Volume, error)

	CreateBackup(ctx context.Context, backup *Backup) (*Backup, error)
}
