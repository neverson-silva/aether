package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserStore interface {
	CreateUser(ctx context.Context, email, name, passwordHash, globalRole string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserWithSecret(ctx context.Context, id uuid.UUID) (*User, error)
	HasUsers(ctx context.Context) (bool, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]Member, error)
	Register(ctx context.Context, email, name, passwordHash, globalRole, orgName, orgSlug string) (*User, *Org, error)
	AddMemberUser(ctx context.Context, orgID uuid.UUID, email, name, passwordHash string, role Role) (*User, error)
	SetTOTP(ctx context.Context, userID uuid.UUID, secret []byte) error
	DisableTOTP(ctx context.Context, userID uuid.UUID) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
}

type OrgStore interface {
	CreateOrg(ctx context.Context, name, slug string, ownerID uuid.UUID) (*Org, error)
	GetOrgByID(ctx context.Context, id uuid.UUID) (*Org, error)
	ListOrgsForUser(ctx context.Context, userID uuid.UUID) ([]Org, error)
}

type MemberStore interface {
	CreateMember(ctx context.Context, orgID, userID uuid.UUID, role Role) error
	GetMember(ctx context.Context, orgID, userID uuid.UUID) (*Member, error)
	UpdateRole(ctx context.Context, orgID, userID uuid.UUID, role Role) error
	DeleteMember(ctx context.Context, orgID, userID uuid.UUID) error
}

type APIKeyStore interface {
	CreateKey(ctx context.Context, orgID uuid.UUID, name, keyHash string, expiresAt *time.Time) (*APIKey, error)
	GetKeyByHash(ctx context.Context, keyHash string) (*APIKey, error)
	ListKeysByOrg(ctx context.Context, orgID uuid.UUID) ([]APIKey, error)
	TouchKey(ctx context.Context, id uuid.UUID) error
	DeleteKey(ctx context.Context, id, orgID uuid.UUID) error
}

type AuditStore interface {
	Record(ctx context.Context, event AuditEvent) error
	List(ctx context.Context, orgID uuid.UUID, limit int32) ([]AuditEvent, error)
}

type SessionStore interface {
	CreateSession(ctx context.Context, userID, orgID uuid.UUID, expiresAt time.Time) (uuid.UUID, error)
	GetSession(ctx context.Context, id uuid.UUID) (*AuthSession, error)
	RevokeSession(ctx context.Context, id uuid.UUID) error
	RevokeUserSessions(ctx context.Context, userID uuid.UUID) error
	RevokeOrgUserSessions(ctx context.Context, orgID, userID uuid.UUID) error
	CreateRefreshToken(ctx context.Context, sessionID, tokenID uuid.UUID, tokenHash string, expiresAt time.Time) error
	ConsumeRefreshToken(ctx context.Context, tokenHash string) (*AuthSession, error)
}

type TokenSigner interface {
	Sign(ctx context.Context, subject, orgID uuid.UUID, role Role, global string, ttl time.Duration) (string, error)
	SignRefresh(ctx context.Context, subject, orgID uuid.UUID, role Role, global string, ttl time.Duration) (string, error)
	SignRefreshUntil(ctx context.Context, subject, orgID uuid.UUID, role Role, global string, expiresAt time.Time) (string, error)
	Verify(ctx context.Context, token string) (*AuthToken, error)
}

type SessionTokenSigner interface {
	SignWithSession(ctx context.Context, sessionID, subject, orgID uuid.UUID, role Role, global string, ttl time.Duration) (string, error)
	SignRefreshWithSession(ctx context.Context, sessionID, subject, orgID uuid.UUID, role Role, global string, ttl time.Duration) (string, error)
	SignRefreshUntilWithSession(ctx context.Context, sessionID, subject, orgID uuid.UUID, role Role, global string, expiresAt time.Time) (string, error)
}

type RefreshTokenIssuer interface {
	IssueRefreshWithSession(ctx context.Context, sessionID, subject, orgID uuid.UUID, role Role, global string, expiresAt time.Time) (string, uuid.UUID, error)
}

type PasswordHasher interface {
	Hash(ctx context.Context, password string) (string, error)
	Verify(ctx context.Context, password, hash string) bool
}
