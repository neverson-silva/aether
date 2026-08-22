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

type Variable struct {
	ID            uuid.UUID
	ProjectID     uuid.UUID
	EnvironmentID uuid.UUID
	Key           string
	Value         string
	IsSecret      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type VariableInput struct {
	Value  string
	Secret bool
}

type SecretCipher interface {
	Encrypt(plain string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type ResolvedVariable struct {
	Key    string
	Value  string
	Source string
	Secret bool
}

type AuditEvent struct {
	ID            uuid.UUID
	ProjectID     uuid.UUID
	EnvironmentID *uuid.UUID
	UserID        uuid.UUID
	Action        string
	Key           string
	CreatedAt     time.Time
}

type Store interface {
	UpsertVariable(ctx context.Context, variable *Variable) (*Variable, error)
	ListVariables(ctx context.Context, projectID, environmentID uuid.UUID) ([]Variable, error)
	DeleteVariable(ctx context.Context, projectID, environmentID uuid.UUID, key string) error

	RecordAudit(ctx context.Context, projectID uuid.UUID, environmentID *uuid.UUID, userID uuid.UUID, action, key string) error
	ListAudit(ctx context.Context, projectID uuid.UUID, limit int) ([]AuditEvent, error)

	SetDefaultEnvironment(ctx context.Context, projectID, environmentID uuid.UUID) error
}
