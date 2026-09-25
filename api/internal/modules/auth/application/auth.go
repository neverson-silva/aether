package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/auth/domain"
)

func hashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

const defaultOrgName = "My Organization"

const sessionTTL = 30 * 24 * time.Hour

type Auth struct {
	Users    domain.UserStore
	Orgs     domain.OrgStore
	Members  domain.MemberStore
	Keys     domain.APIKeyStore
	AuditLog domain.AuditStore
	Tokens   domain.TokenSigner
	Hash     domain.PasswordHasher
	SSO      SSOProviderLister
	Sessions domain.SessionStore

	TokenTTL time.Duration
}

type SSOProviderLister interface {
	CountEnabledOIDC(ctx context.Context) (int, error)
}

func (a *Auth) Register(ctx context.Context, email, name, password string) (*domain.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	if err := validateAuth(email, name, password); err != nil {
		return nil, "", err
	}
	hash, err := a.Hash.Hash(ctx, password)
	if err != nil {
		return nil, "", err
	}
	user, org, err := a.Users.Register(ctx, email, name, hash, "", defaultOrgName, "org-"+uuid.NewString()[:8])
	if err != nil {
		return nil, "", err
	}
	token, err := a.sign(ctx, user, org.ID, domain.RoleOwner)
	if err != nil {
		return nil, "", err
	}
	_ = a.AuditLog.Record(ctx, domain.AuditEvent{
		OrgID: org.ID, UserID: user.ID, Action: "user.register",
		ResourceType: "user", ResourceID: user.ID.String(), Details: email,
	})
	return user, token, nil
}

func (a *Auth) Login(ctx context.Context, email, password string) (*domain.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := a.Users.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, "", domain.ErrInvalidCredentials
		}
		return nil, "", err
	}
	if !a.Hash.Verify(ctx, password, user.PasswordHash) {
		return nil, "", domain.ErrInvalidCredentials
	}
	orgs, err := a.Orgs.ListOrgsForUser(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}
	if len(orgs) == 0 {
		return nil, "", domain.ErrNotFound
	}
	org := orgs[0]
	token, err := a.sign(ctx, user, org.ID, org.Role)
	if err != nil {
		return nil, "", err
	}
	_ = a.AuditLog.Record(ctx, domain.AuditEvent{
		OrgID: org.ID, UserID: user.ID, Action: "user.login",
		ResourceType: "user", ResourceID: user.ID.String(),
	})
	return user, token, nil
}

func (a *Auth) Refresh(ctx context.Context, raw string) (string, string, error) {
	token, err := a.Tokens.Verify(ctx, raw)
	if err != nil || token.Kind != "refresh" {
		return "", "", domain.ErrUnauthorized
	}
	if a.Sessions != nil {
		if token.SessionID == uuid.Nil || token.TokenID == uuid.Nil {
			return "", "", domain.ErrUnauthorized
		}
		session, err := a.Sessions.ConsumeRefreshToken(ctx, hashKey(raw))
		if err != nil || session.ID != token.SessionID {
			return "", "", domain.ErrUnauthorized
		}
	}
	user, member, err := a.currentIdentity(ctx, token)
	if err != nil {
		return "", "", err
	}
	access, err := a.signSession(ctx, token.SessionID, user, token.OrgID, member.Role)
	if err != nil {
		return "", "", err
	}
	refresh, err := a.issueRefreshSession(ctx, token.SessionID, user, token.OrgID, member.Role, token.Expires)
	return access, refresh, err
}

func (a *Auth) CreateRefresh(ctx context.Context, raw string) (string, error) {
	token, err := a.Tokens.Verify(ctx, raw)
	if err != nil || token.Kind != "access" {
		return "", domain.ErrUnauthorized
	}
	if _, _, err := a.currentIdentity(ctx, token); err != nil {
		return "", err
	}
	return a.issueRefreshSession(ctx, token.SessionID, &domain.User{ID: token.Subject, GlobalRole: token.Global}, token.OrgID, token.Role, time.Now().Add(sessionTTL))
}

func (a *Auth) ValidateAccessToken(ctx context.Context, token *domain.AuthToken) error {
	if token == nil || token.Kind != "access" {
		return domain.ErrUnauthorized
	}
	_, _, err := a.currentIdentity(ctx, token)
	return err
}

func (a *Auth) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	if a.Sessions == nil || sessionID == uuid.Nil {
		return nil
	}
	return a.Sessions.RevokeSession(ctx, sessionID)
}

func (a *Auth) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	if len(newPassword) < 8 || currentPassword == "" || currentPassword == newPassword {
		return domain.ErrValidation
	}
	user, err := a.Users.GetUserWithSecret(ctx, userID)
	if err != nil || !a.Hash.Verify(ctx, currentPassword, user.PasswordHash) {
		return domain.ErrInvalidCredentials
	}
	hash, err := a.Hash.Hash(ctx, newPassword)
	if err != nil {
		return err
	}
	if err := a.Users.UpdatePassword(ctx, userID, hash); err != nil {
		return err
	}
	if a.Sessions != nil {
		return a.Sessions.RevokeUserSessions(ctx, userID)
	}
	return nil
}

func (a *Auth) SSOLogin(ctx context.Context, email, name string) (*domain.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, "", domain.ErrValidation
	}
	user, err := a.Users.GetUserByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return nil, "", err
		}
		if name == "" {
			name = email
		}
		user, org, err := a.Users.Register(ctx, email, name, "", "", defaultOrgName, "org-"+uuid.NewString()[:8])
		if err != nil {
			return nil, "", err
		}
		token, err := a.sign(ctx, user, org.ID, domain.RoleOwner)
		return user, token, err
	}
	orgs, err := a.Orgs.ListOrgsForUser(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}
	if len(orgs) == 0 {
		return nil, "", domain.ErrNotFound
	}
	token, err := a.sign(ctx, user, orgs[0].ID, orgs[0].Role)
	return user, token, err
}

func (a *Auth) Me(ctx context.Context, userID uuid.UUID) (*domain.User, []domain.Org, error) {
	user, err := a.Users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	orgs, err := a.Orgs.ListOrgsForUser(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	return user, orgs, nil
}

func (a *Auth) AddMember(ctx context.Context, orgID, actorID uuid.UUID, email, name, password string, role domain.Role) error {
	if !role.Valid() {
		return domain.ErrValidation
	}
	member, err := a.Members.GetMember(ctx, orgID, actorID)
	if err != nil {
		return err
	}
	if !canManage(member.Role) {
		return domain.ErrForbidden
	}
	if _, err := a.Users.GetUserByEmail(ctx, strings.ToLower(email)); err == nil {
		return domain.ErrEmailTaken
	}
	hash, err := a.Hash.Hash(ctx, password)
	if err != nil {
		return err
	}
	user, err := a.Users.AddMemberUser(ctx, orgID, strings.ToLower(email), name, hash, role)
	if err != nil {
		return err
	}
	return a.AuditLog.Record(ctx, domain.AuditEvent{
		OrgID: orgID, UserID: actorID, Action: "member.add",
		ResourceType: "member", ResourceID: user.ID.String(), Details: email,
	})
}

func (a *Auth) ListMembers(ctx context.Context, orgID uuid.UUID) ([]domain.Member, error) {
	return a.Users.ListByOrg(ctx, orgID)
}

func (a *Auth) UpdateMemberRole(ctx context.Context, orgID, actorID, targetID uuid.UUID, role domain.Role) error {
	actor, err := a.Members.GetMember(ctx, orgID, actorID)
	if err != nil {
		return err
	}
	if !actor.Role.CanManage() {
		return domain.ErrForbidden
	}
	if !role.Valid() {
		return domain.ErrValidation
	}
	if err := a.Members.UpdateRole(ctx, orgID, targetID, role); err != nil {
		return err
	}
	if a.Sessions != nil {
		return a.Sessions.RevokeOrgUserSessions(ctx, orgID, targetID)
	}
	return nil
}

func (a *Auth) Status(ctx context.Context, userID uuid.UUID) (bool, error) {
	user, err := a.Users.GetUserByID(ctx, userID)
	if err != nil {
		return false, err
	}
	return user.TotpEnabled, nil
}

func (a *Auth) PublicStatus(ctx context.Context) (registered bool, sso bool) {
	if has, err := a.Users.HasUsers(ctx); err == nil {
		registered = has
	}
	if a.SSO != nil {
		if count, err := a.SSO.CountEnabledOIDC(ctx); err == nil && count > 0 {
			sso = true
		}
	}
	return registered, sso
}

func (a *Auth) CreateAPIKey(ctx context.Context, orgID, userID uuid.UUID, name string) (*domain.APIKey, string, error) {
	if strings.TrimSpace(name) == "" {
		return nil, "", domain.ErrValidation
	}
	raw := "aether_" + uuid.NewString() + uuid.NewString()
	hash := hashKey(raw)
	key, err := a.Keys.CreateKey(ctx, orgID, name, hash, nil)
	if err != nil {
		return nil, "", err
	}
	_ = a.AuditLog.Record(ctx, domain.AuditEvent{
		OrgID: orgID, UserID: userID, Action: "apikey.create",
		ResourceType: "apikey", ResourceID: key.ID.String(), Details: name,
	})
	return key, raw, nil
}

func (a *Auth) ListAPIKeys(ctx context.Context, orgID uuid.UUID) ([]domain.APIKey, error) {
	return a.Keys.ListKeysByOrg(ctx, orgID)
}

func (a *Auth) DeleteAPIKey(ctx context.Context, orgID, userID, keyID uuid.UUID) error {
	if err := a.Keys.DeleteKey(ctx, keyID, orgID); err != nil {
		return err
	}
	return a.AuditLog.Record(ctx, domain.AuditEvent{
		OrgID: orgID, UserID: userID, Action: "apikey.delete",
		ResourceType: "apikey", ResourceID: keyID.String(),
	})
}

func (a *Auth) Audit(ctx context.Context, orgID uuid.UUID, limit int32) ([]domain.AuditEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return a.AuditLog.List(ctx, orgID, limit)
}

func (a *Auth) EnrollTOTP(ctx context.Context, userID uuid.UUID, email string) (secret, uri string, err error) {
	secret, err = generateTOTPSecret()
	if err != nil {
		return "", "", err
	}
	if err := a.Users.SetTOTP(ctx, userID, []byte(secret)); err != nil {
		return "", "", err
	}
	return secret, provisioningURI(secret, email, "aether"), nil
}

func (a *Auth) VerifyTOTP(ctx context.Context, userID uuid.UUID, code string) error {
	user, err := a.Users.GetUserWithSecret(ctx, userID)
	if err != nil {
		return err
	}
	if len(user.TotpSecret) == 0 {
		return domain.ErrValidation
	}
	expected, err := totpCode(string(user.TotpSecret), time.Now())
	if err != nil {
		return err
	}
	previous, err := totpCode(string(user.TotpSecret), time.Now().Add(-30*time.Second))
	if err != nil {
		return err
	}
	if code != expected && code != previous {
		return domain.ErrValidation
	}
	return nil
}

func (a *Auth) DisableTOTP(ctx context.Context, userID uuid.UUID) error {
	if err := a.Users.DisableTOTP(ctx, userID); err != nil {
		return err
	}
	if a.Sessions != nil {
		return a.Sessions.RevokeUserSessions(ctx, userID)
	}
	return nil
}

func (a *Auth) sign(ctx context.Context, user *domain.User, orgID uuid.UUID, role domain.Role) (string, error) {
	if a.Sessions == nil {
		return a.Tokens.Sign(ctx, user.ID, orgID, role, user.GlobalRole, a.TokenTTL)
	}
	sessionID, err := a.Sessions.CreateSession(ctx, user.ID, orgID, time.Now().Add(sessionTTL))
	if err != nil {
		return "", err
	}
	return a.signSession(ctx, sessionID, user, orgID, role)
}

func (a *Auth) signSession(ctx context.Context, sessionID uuid.UUID, user *domain.User, orgID uuid.UUID, role domain.Role) (string, error) {
	if signer, ok := a.Tokens.(domain.SessionTokenSigner); ok {
		return signer.SignWithSession(ctx, sessionID, user.ID, orgID, role, user.GlobalRole, a.TokenTTL)
	}
	return a.Tokens.Sign(ctx, user.ID, orgID, role, user.GlobalRole, a.TokenTTL)
}

func (a *Auth) signRefreshSession(ctx context.Context, sessionID, subject, orgID uuid.UUID, role domain.Role, global string, ttl time.Duration) (string, error) {
	if signer, ok := a.Tokens.(domain.SessionTokenSigner); ok {
		return signer.SignRefreshWithSession(ctx, sessionID, subject, orgID, role, global, ttl)
	}
	return a.Tokens.SignRefresh(ctx, subject, orgID, role, global, ttl)
}

func (a *Auth) signRefreshSessionUntil(ctx context.Context, sessionID uuid.UUID, user *domain.User, orgID uuid.UUID, role domain.Role, expiresAt time.Time) (string, error) {
	if signer, ok := a.Tokens.(domain.SessionTokenSigner); ok {
		return signer.SignRefreshUntilWithSession(ctx, sessionID, user.ID, orgID, role, user.GlobalRole, expiresAt)
	}
	return a.Tokens.SignRefreshUntil(ctx, user.ID, orgID, role, user.GlobalRole, expiresAt)
}

func (a *Auth) issueRefreshSession(ctx context.Context, sessionID uuid.UUID, user *domain.User, orgID uuid.UUID, role domain.Role, expiresAt time.Time) (string, error) {
	issuer, ok := a.Tokens.(domain.RefreshTokenIssuer)
	if !ok || a.Sessions == nil || sessionID == uuid.Nil {
		return a.signRefreshSessionUntil(ctx, sessionID, user, orgID, role, expiresAt)
	}
	token, tokenID, err := issuer.IssueRefreshWithSession(ctx, sessionID, user.ID, orgID, role, user.GlobalRole, expiresAt)
	if err != nil {
		return "", err
	}
	if err := a.Sessions.CreateRefreshToken(ctx, sessionID, tokenID, hashKey(token), expiresAt); err != nil {
		_ = a.Sessions.RevokeSession(ctx, sessionID)
		return "", err
	}
	return token, nil
}

func (a *Auth) currentIdentity(ctx context.Context, token *domain.AuthToken) (*domain.User, *domain.Member, error) {
	if a.Users == nil || a.Members == nil {
		return nil, nil, domain.ErrUnauthorized
	}
	user, err := a.Users.GetUserByID(ctx, token.Subject)
	if err != nil {
		return nil, nil, domain.ErrUnauthorized
	}
	member, err := a.Members.GetMember(ctx, token.OrgID, token.Subject)
	if err != nil || member.Role != token.Role || user.GlobalRole != token.Global {
		return nil, nil, domain.ErrUnauthorized
	}
	if a.Sessions != nil {
		if token.SessionID == uuid.Nil {
			return nil, nil, domain.ErrUnauthorized
		}
		session, err := a.Sessions.GetSession(ctx, token.SessionID)
		if err != nil || session.UserID != token.Subject || session.OrgID != token.OrgID || session.RevokedAt != nil || !session.ExpiresAt.After(time.Now()) {
			return nil, nil, domain.ErrUnauthorized
		}
	}
	return user, member, nil
}

func validateAuth(email, name, password string) error {
	if !strings.Contains(email, "@") || len(email) > 255 {
		return domain.ErrValidation
	}
	if len(name) < 1 || len(name) > 120 {
		return domain.ErrValidation
	}
	if len(password) < 8 {
		return domain.ErrValidation
	}
	return nil
}

func slugify(s string) string {
	out := strings.Builder{}
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out.WriteRune(r)
		case r == ' ' || r == '-':
			out.WriteByte('-')
		}
	}
	slug := strings.Trim(out.String(), "-")
	if slug == "" {
		slug = "org-" + uuid.NewString()[:8]
	}
	return slug
}

func canManage(role domain.Role) bool {
	return role == domain.RoleOwner || role == domain.RoleAdmin
}
