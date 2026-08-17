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

type GitOps struct {
	ID           uuid.UUID
	OrgID        uuid.UUID
	Name         string
	RepoURL      string
	Branch       string
	Path         string
	TargetOrgID  *uuid.UUID
	ApplyMode    string
	LastSHA      string
	LastStatus   string
	DriftAdded   int
	DriftChanged int
	DriftRemoved int
	LastSync     *time.Time
	CreatedAt    time.Time
}

type SyncResult struct {
	SHA      string
	Added    int
	Changed  int
	Removed  int
	SyncedAt time.Time
}

type Store interface {
	CreateGitOps(ctx context.Context, g *GitOps) (*GitOps, error)
	GetGitOps(ctx context.Context, id uuid.UUID) (*GitOps, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]GitOps, error)
	UpdateSync(ctx context.Context, id uuid.UUID, result SyncResult) error
	DeleteGitOps(ctx context.Context, id, orgID uuid.UUID) error
}
