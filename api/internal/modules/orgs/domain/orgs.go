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

type Org struct {
	ID          uuid.UUID
	Name        string
	Slug        string
	Avatar      *string
	Color       *string
	Description string
	OwnerID     uuid.UUID
	Role        Role
	CreatedAt   time.Time
}

type Member struct {
	OrgID     uuid.UUID
	UserID    uuid.UUID
	Role      Role
	Email     string
	Name      string
	CreatedAt time.Time
}

type Assignment struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	UserID    uuid.UUID
	ProjectID uuid.UUID
	CreatedAt time.Time
}

type AuditEvent struct {
	ID        uuid.UUID
	Action    string
	UserID    uuid.UUID
	Resource  string
	Details   string
	CreatedAt time.Time
}

type Store interface {
	CreateOrg(ctx context.Context, name, slug string, ownerID uuid.UUID) (*Org, error)
	CreateMember(ctx context.Context, orgID, userID uuid.UUID, role Role) error
	GetOrg(ctx context.Context, id uuid.UUID) (*Org, error)
	UpdateOrg(ctx context.Context, id uuid.UUID, name, description string, avatar, color *string) (*Org, error)
	DeleteOrg(ctx context.Context, id, ownerID uuid.UUID) error
	ListOrgsByUser(ctx context.Context, userID uuid.UUID) ([]Org, error)
	GetRole(ctx context.Context, orgID, userID uuid.UUID) (Role, error)

	ListMembers(ctx context.Context, orgID uuid.UUID) ([]Member, error)
	GetMember(ctx context.Context, orgID, userID uuid.UUID) (*Member, error)
	UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role Role) error
	RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error

	SetAssignment(ctx context.Context, orgID, userID, projectID uuid.UUID) error
	RemoveAssignment(ctx context.Context, orgID, userID, projectID uuid.UUID) error
	ListAssignments(ctx context.Context, orgID uuid.UUID) ([]Assignment, error)

	RecordAudit(ctx context.Context, orgID, userID uuid.UUID, action, resource, details string) error
	ListAudit(ctx context.Context, orgID uuid.UUID, limit int) ([]AuditEvent, error)
}
