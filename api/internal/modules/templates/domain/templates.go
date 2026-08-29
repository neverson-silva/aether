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

type Template struct {
	ID            uuid.UUID
	Name          string
	Description   string
	Category      string
	Icon          string
	Version       string
	Definition    string
	Readme        string
	Homepage      string
	GitHub        string
	License       string
	Installs      int
	Featured      bool
	Verified      bool
	EditorsChoice bool
	Tags          []string
	UpdatedAt     time.Time
	ComposeYAML   string
}

type ComposeApp struct {
	ID            uuid.UUID
	OrgID         uuid.UUID
	ProjectID     uuid.UUID
	EnvironmentID *uuid.UUID
	ServiceID     uuid.UUID
	Name          string
	Compose       string
	Status        string
	CreatedAt     time.Time
}

type Filter struct {
	Category      string
	Search        string
	Featured      bool
	Verified      bool
	EditorsChoice bool
}

type Store interface {
	ListTemplates(ctx context.Context, filter Filter) ([]Template, error)
	GetTemplate(ctx context.Context, id uuid.UUID) (*Template, error)
	IncrementInstalls(ctx context.Context, id uuid.UUID) error

	CreateComposeApp(ctx context.Context, app *ComposeApp) (*ComposeApp, error)
	GetComposeApp(ctx context.Context, id uuid.UUID) (*ComposeApp, error)
	ListComposeAppsByOrg(ctx context.Context, orgID uuid.UUID) ([]ComposeApp, error)
	SetComposeStatus(ctx context.Context, id uuid.UUID, status string) error
	DeleteComposeApp(ctx context.Context, id, orgID uuid.UUID) error
}
