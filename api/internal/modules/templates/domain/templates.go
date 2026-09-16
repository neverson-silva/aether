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
	ID             uuid.UUID
	RemoteID       string
	LogoTemplateID uuid.UUID
	Name           string
	Description    string
	Category       string
	Icon           string
	Version        string
	Definition     string
	Readme         string
	Homepage       string
	GitHub         string
	License        string
	Installs       int
	Featured       bool
	Verified       bool
	EditorsChoice  bool
	Tags           []string
	UpdatedAt      time.Time
	ComposeYAML    string
	Environment    []TemplateEnvironmentVariable
	Variables      []TemplateVariable
	Mounts         []TemplateMount
	Domains        []TemplateDomain
}

type TemplateEnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TemplateVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TemplateMount struct {
	ServiceName string `json:"service_name,omitempty"`
	FilePath    string `json:"file_path"`
	Content     string `json:"content"`
}

type TemplateDomain struct {
	ServiceName string `json:"service_name"`
	Port        int    `json:"port"`
	Host        string `json:"host"`
	Path        string `json:"path"`
	Publish     bool   `json:"publish,omitempty"`
}

type ComposeApp struct {
	ID            uuid.UUID
	OrgID         uuid.UUID
	ProjectID     uuid.UUID
	EnvironmentID *uuid.UUID
	ServiceID     uuid.UUID
	Name          string
	Compose       string
	Port          int
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
	CreateComposeApp(ctx context.Context, app *ComposeApp) (*ComposeApp, error)
	UpdateComposeApp(ctx context.Context, id uuid.UUID, compose string, port int) error
	GetComposeApp(ctx context.Context, id uuid.UUID) (*ComposeApp, error)
	ListComposeAppsByOrg(ctx context.Context, orgID uuid.UUID) ([]ComposeApp, error)
	SetComposeStatus(ctx context.Context, id uuid.UUID, status string) error
	DeleteComposeApp(ctx context.Context, id, orgID uuid.UUID) error
}
