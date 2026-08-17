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

type Domain struct {
	ID            uuid.UUID
	AppID         uuid.UUID
	ServerID      uuid.UUID
	Host          string
	HTTPS         bool
	Path          string
	InternalPath  string
	StripPath     bool
	ContainerPort int
	Status        string
	CertStatus    string
	RetryCount    int
	LastError     string
	NextRetryAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type DomainStatus string

const (
	DomainPending      DomainStatus = "PENDING"
	DomainProvisioning DomainStatus = "PROVISIONING"
	DomainActive       DomainStatus = "ACTIVE"
	DomainError        DomainStatus = "ERROR"
	DomainRemoving     DomainStatus = "REMOVING"
	DomainRemoved      DomainStatus = "REMOVED"
)

type Preview struct {
	ID           uuid.UUID
	AppID        uuid.UUID
	Branch       string
	DeploymentID *uuid.UUID
	ContainerID  string
	Domain       string
	Status       string
	CreatedAt    time.Time
}

type Certificate struct {
	DomainID  uuid.UUID
	AppID     uuid.UUID
	AppName   string
	Host      string
	HTTPS     bool
	CertState string
	CreatedAt time.Time
}

type Store interface {
	CreateDomain(ctx context.Context, d *Domain) (*Domain, error)
	ListDomains(ctx context.Context, appID uuid.UUID) ([]Domain, error)
	GetDomainByHost(ctx context.Context, appID uuid.UUID, host string) (*Domain, error)
	GetDomainByID(ctx context.Context, id uuid.UUID) (*Domain, error)
	UpdateDomainStatus(ctx context.Context, id, appID uuid.UUID, status, certStatus string) error
	UpdateDomainFields(ctx context.Context, id, appID uuid.UUID, host string, https bool, path, internalPath string, stripPath bool, containerPort int) error
	UpdateDomainProvision(ctx context.Context, id, appID uuid.UUID, status, certStatus, lastError string, nextRetryAt *time.Time, retryCount int) error
	ListProvisioningDomains(ctx context.Context, now time.Time, maxRetries int) ([]Domain, error)
	DeleteDomain(ctx context.Context, id, appID uuid.UUID) error

	CreatePreview(ctx context.Context, preview *Preview) (*Preview, error)
	ListPreviews(ctx context.Context, appID uuid.UUID) ([]Preview, error)
	GetPreview(ctx context.Context, id, appID uuid.UUID) (*Preview, error)
	GetPreviewByID(ctx context.Context, id uuid.UUID) (*Preview, error)
	UpdatePreviewResult(ctx context.Context, id, appID uuid.UUID, deploymentID *uuid.UUID, containerID, status string) error
	DeletePreview(ctx context.Context, id, appID uuid.UUID) error

	ListCertificatesByOrg(ctx context.Context, orgID uuid.UUID) ([]Certificate, error)
}
