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

type OutWebhook struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	Name      string
	URL       string
	SecretEnc string
	Events    []string
	Enabled   bool
	CreatedAt time.Time
}

type PasswordCipher interface {
	Encrypt(plain string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type Store interface {
	CreateOutWebhook(ctx context.Context, hook *OutWebhook) (*OutWebhook, error)
	GetOutWebhook(ctx context.Context, id uuid.UUID) (*OutWebhook, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]OutWebhook, error)
	ListByEvent(ctx context.Context, event string) ([]OutWebhook, error)
	DeleteOutWebhook(ctx context.Context, id, orgID uuid.UUID) error
}
