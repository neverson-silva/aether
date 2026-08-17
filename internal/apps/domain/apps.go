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

type Project struct {
	ID          uuid.UUID
	OrgID       uuid.UUID
	Name        string
	Slug        string
	Description string
	Color       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Environment struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	Slug        string
	Description string
	Color       string
	IsDefault   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type App struct {
	ID             uuid.UUID
	OrgID          uuid.UUID
	ProjectID      uuid.UUID
	EnvironmentID  *uuid.UUID
	Name           string
	SourceType     string
	Image          string
	GitURL         string
	GitBranch      string
	UploadID       string
	Dockerfile     string
	Port           int
	CPUs           string
	MemMB          int
	HealthCheck    HealthCheck
	WebhookSecret  string
	BuildType      string
	PreviewDomain  string
	ImageRetention int
	StorageMB      int
	InstallCommand string
	BuildCommand   string
	StartCommand   string
	RootFolder     string
	DistFolder     string
	WatchPaths     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type HealthCheck struct {
	Enabled    bool
	Path       string
	IntervalMS int
	TimeoutMS  int
	Retries    int
}

type EnvVar struct {
	Name   string
	Value  string
	Secret bool
}

type SecretCipher interface {
	Encrypt(plain string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type Store interface {
	CreateProject(ctx context.Context, orgID uuid.UUID, name, slug, description, color string) (*Project, error)
	GetProject(ctx context.Context, id, orgID uuid.UUID) (*Project, error)
	ListProjects(ctx context.Context, orgID uuid.UUID) ([]Project, error)
	UpdateProject(ctx context.Context, id, orgID uuid.UUID, name, slug, description, color string) (*Project, error)
	DeleteProject(ctx context.Context, id, orgID uuid.UUID) error

	CreateEnvironment(ctx context.Context, projectID uuid.UUID, name, slug, description, color string, isDefault bool) (*Environment, error)
	GetEnvironment(ctx context.Context, id, projectID uuid.UUID) (*Environment, error)
	DefaultEnvironment(ctx context.Context, projectID uuid.UUID) (uuid.UUID, error)
	ListEnvironments(ctx context.Context, projectID uuid.UUID) ([]Environment, error)
	UpdateEnvironment(ctx context.Context, id, projectID uuid.UUID, name, slug, description, color string) (*Environment, error)
	DeleteEnvironment(ctx context.Context, id, projectID uuid.UUID) error

	CreateApp(ctx context.Context, app *App) (*App, error)
	GetApp(ctx context.Context, id, orgID uuid.UUID) (*App, error)
	GetAppByID(ctx context.Context, id uuid.UUID) (*App, error)
	GetAppByName(ctx context.Context, orgID uuid.UUID, name string) (*App, error)
	ListAppsByProject(ctx context.Context, orgID, projectID uuid.UUID) ([]App, error)
	ListAppsByOrg(ctx context.Context, orgID uuid.UUID) ([]App, error)
	UpdateApp(ctx context.Context, app *App) (*App, error)
	DeleteApp(ctx context.Context, id, orgID uuid.UUID) error
	SetWebhookSecret(ctx context.Context, id, orgID uuid.UUID, secret string) error

	UpsertEnvVar(ctx context.Context, appID uuid.UUID, name, value string, secret bool) error
	ListEnvVars(ctx context.Context, appID uuid.UUID) ([]EnvVar, error)
	DeleteEnvVar(ctx context.Context, appID uuid.UUID, name string) error
}
