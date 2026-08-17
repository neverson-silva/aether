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

type Stage struct {
	Name     string   `json:"name"`
	Image    string   `json:"image"`
	Commands []string `json:"commands"`
}

type Pipeline struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	AppID     *uuid.UUID
	Name      string
	Trigger   string
	Stages    []Stage
	Enabled   bool
	CreatedAt time.Time
}

type Run struct {
	ID         uuid.UUID
	PipelineID uuid.UUID
	Status     string
	Trigger    string
	Log        string
	StartedAt  time.Time
	FinishedAt *time.Time
}

type Store interface {
	CreatePipeline(ctx context.Context, pipeline *Pipeline) (*Pipeline, error)
	GetPipeline(ctx context.Context, id uuid.UUID) (*Pipeline, error)
	ListPipelinesByOrg(ctx context.Context, orgID uuid.UUID) ([]Pipeline, error)
	DeletePipeline(ctx context.Context, id, orgID uuid.UUID) error

	CreateRun(ctx context.Context, run *Run) (*Run, error)
	FinishRun(ctx context.Context, id uuid.UUID, status, log string) error
	ListRuns(ctx context.Context, pipelineID uuid.UUID, limit int) ([]Run, error)
}
