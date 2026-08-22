package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email already registered")
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrForbidden          = errors.New("access denied")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrValidation         = errors.New("invalid input")
)

type Role string

const (
	RoleOwner     Role = "owner"
	RoleAdmin     Role = "admin"
	RoleDeveloper Role = "developer"
	RoleMember    Role = "member"
	RoleViewer    Role = "viewer"
)

func (r Role) Valid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleDeveloper, RoleMember, RoleViewer:
		return true
	}
	return false
}

func (r Role) CanManage() bool {
	return r == RoleOwner || r == RoleAdmin
}

type User struct {
	ID           uuid.UUID
	Email        string
	Name         string
	PasswordHash string
	GlobalRole   string
	TotpEnabled  bool
	TotpSecret   []byte
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Org struct {
	ID      uuid.UUID
	Name    string
	Slug    string
	Avatar  *string
	Color   *string
	OwnerID uuid.UUID
	Role    Role
}

type Member struct {
	OrgID     uuid.UUID
	UserID    uuid.UUID
	Role      Role
	Email     string
	Name      string
	CreatedAt time.Time
}

type APIKey struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	Name      string
	KeyHash   string
	ExpiresAt *time.Time
	LastUsed  *time.Time
	CreatedAt time.Time
}

type Token struct {
	Token      string
	User       *User
	OrgID      uuid.UUID
	OrgRole    Role
	GlobalRole string
}

type AuthToken struct {
	Subject uuid.UUID
	OrgID   uuid.UUID
	Role    Role
	Global  string
	Expires time.Time
}

type AuditEvent struct {
	OrgID        uuid.UUID
	UserID       uuid.UUID
	Action       string
	ResourceType string
	ResourceID   string
	Details      string
	CreatedAt    time.Time
}
