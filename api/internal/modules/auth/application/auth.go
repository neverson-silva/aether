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

type Auth struct {
	Users    domain.UserStore
	Orgs     domain.OrgStore
	Members  domain.MemberStore
	Keys     domain.APIKeyStore
	AuditLog domain.AuditStore
	Tokens   domain.TokenSigner
	Hash     domain.PasswordHasher
	SSO      SSOProviderLister

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
	if _, err := a.Users.GetUserByEmail(ctx, email); err == nil {
		return nil, "", domain.ErrEmailTaken
	}
	hash, err := a.Hash.Hash(ctx, password)
	if err != nil {
		return nil, "", err
	}
	user, org, err := a.Users.Register(ctx, email, name, hash, "", defaultOrgName, "my-organization")
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
		println("DEBUG_LOGIN_GETUSER:", err.Error())
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
		println("DEBUG_LOGIN_ORGS:", err.Error())
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
		user, org, err := a.Users.Register(ctx, email, name, "", "", defaultOrgName, "my-organization")
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
	return a.Members.UpdateRole(ctx, orgID, targetID, role)
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
	return a.Users.DisableTOTP(ctx, userID)
}

func (a *Auth) sign(ctx context.Context, user *domain.User, orgID uuid.UUID, role domain.Role) (string, error) {
	return a.Tokens.Sign(ctx, user.ID, orgID, role, user.GlobalRole, a.TokenTTL)
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
