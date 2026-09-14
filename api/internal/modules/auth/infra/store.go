package infra

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"aether/internal/modules/auth/domain"
	gen "aether/internal/platform/infrastructure/pg/gen"
)

type Store struct {
	q  gen.Querier
	db *sql.DB
}

func NewStore(pool *pgxpool.Pool) *Store {
	db := stdlib.OpenDBFromPool(pool)
	return &Store{q: gen.New(db), db: db}
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) CreateSession(ctx context.Context, userID, orgID uuid.UUID, expiresAt time.Time) (uuid.UUID, error) {
	id := uuid.New()
	err := s.db.QueryRowContext(ctx, `INSERT INTO auth_sessions (id, user_id, org_id, expires_at) VALUES ($1, $2, $3, $4) RETURNING id`, id, userID, orgID, expiresAt).Scan(&id)
	return id, err
}

func (s *Store) GetSession(ctx context.Context, id uuid.UUID) (*domain.AuthSession, error) {
	session := &domain.AuthSession{ID: id}
	var revokedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `SELECT user_id, org_id, expires_at, revoked_at FROM auth_sessions WHERE id = $1`, id).Scan(&session.UserID, &session.OrgID, &session.ExpiresAt, &revokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if revokedAt.Valid {
		session.RevokedAt = &revokedAt.Time
	}
	return session, nil
}

func (s *Store) RevokeSession(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at = COALESCE(revoked_at, now()) WHERE id = $1`, id)
	return err
}

func (s *Store) RevokeUserSessions(ctx context.Context, userID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at = COALESCE(revoked_at, now()) WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}

func (s *Store) RevokeOrgUserSessions(ctx context.Context, orgID, userID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at = COALESCE(revoked_at, now()) WHERE org_id = $1 AND user_id = $2 AND revoked_at IS NULL`, orgID, userID)
	return err
}

func (s *Store) CreateRefreshToken(ctx context.Context, sessionID, tokenID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO auth_refresh_tokens (id, session_id, token_hash, expires_at) VALUES ($1, $2, $3, $4)`, tokenID, sessionID, tokenHash, expiresAt)
	return err
}

func (s *Store) ConsumeRefreshToken(ctx context.Context, tokenHash string) (*domain.AuthSession, error) {
	session := &domain.AuthSession{}
	var revokedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `UPDATE auth_refresh_tokens rt SET used_at = now() FROM auth_sessions s WHERE rt.token_hash = $1 AND rt.used_at IS NULL AND rt.expires_at > now() AND s.id = rt.session_id AND s.revoked_at IS NULL AND s.expires_at > now() RETURNING s.id, s.user_id, s.org_id, s.expires_at, s.revoked_at`, tokenHash).Scan(&session.ID, &session.UserID, &session.OrgID, &session.ExpiresAt, &revokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	if revokedAt.Valid {
		session.RevokedAt = &revokedAt.Time
	}
	return session, nil
}

func (s *Store) CreateUser(ctx context.Context, email, name, passwordHash, globalRole string) (*domain.User, error) {
	row, err := s.q.CreateUser(ctx, gen.CreateUserParams{
		Email: email, Name: name, PasswordHash: passwordHash, GlobalRole: globalRole,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return userFromCreateRow(row), nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, mapErr(err)
	}
	return userFromEmailRow(row), nil
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return userFromIDRow(row), nil
}

func (s *Store) GetUserWithSecret(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row, err := s.q.GetUserWithSecret(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return userFromSecretRow(row), nil
}

func (s *Store) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, userID, passwordHash)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) HasUsers(ctx context.Context) (bool, error) {
	return s.q.HasUsers(ctx)
}

func (s *Store) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.Member, error) {
	rows, err := s.q.ListUsersByOrg(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Member, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Member{
			OrgID: orgID, UserID: r.ID, Role: domain.Role(r.MemberRole),
			Email: r.Email, Name: r.Name, CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

func (s *Store) Register(ctx context.Context, email, name, passwordHash, globalRole, orgName, orgSlug string) (*domain.User, *domain.Org, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(817463921)`); err != nil {
		return nil, nil, err
	}
	q := gen.New(tx)
	hasUsers, err := q.HasUsers(ctx)
	if err != nil {
		return nil, nil, err
	}
	if !hasUsers {
		globalRole = "admin"
	}
	userRow, err := q.CreateUser(ctx, gen.CreateUserParams{
		Email: email, Name: name, PasswordHash: passwordHash, GlobalRole: globalRole,
	})
	if err != nil {
		return nil, nil, mapErr(err)
	}
	orgRow, err := q.CreateOrg(ctx, gen.CreateOrgParams{
		Name: orgName, Slug: orgSlug, OwnerUserID: uuid.NullUUID{UUID: userRow.ID, Valid: true},
	})
	if err != nil {
		return nil, nil, mapErr(err)
	}
	if err := q.CreateMember(ctx, gen.CreateMemberParams{
		OrgID: orgRow.ID, UserID: userRow.ID, Role: string(domain.RoleOwner),
	}); err != nil {
		return nil, nil, mapErr(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	return userFromCreateRow(userRow), orgFromRow(orgRow, domain.RoleOwner), nil
}

func (s *Store) AddMemberUser(ctx context.Context, orgID uuid.UUID, email, name, passwordHash string, role domain.Role) (*domain.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	q := gen.New(tx)
	userRow, err := q.CreateUser(ctx, gen.CreateUserParams{
		Email: email, Name: name, PasswordHash: passwordHash, GlobalRole: "",
	})
	if err != nil {
		return nil, mapErr(err)
	}
	if err := q.CreateMember(ctx, gen.CreateMemberParams{
		OrgID: orgID, UserID: userRow.ID, Role: string(role),
	}); err != nil {
		return nil, mapErr(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return userFromCreateRow(userRow), nil
}

func (s *Store) SetTOTP(ctx context.Context, userID uuid.UUID, secret []byte) error {
	return mapErr(s.q.SetTOTP(ctx, gen.SetTOTPParams{ID: userID, TotpSecret: secret}))
}

func (s *Store) DisableTOTP(ctx context.Context, userID uuid.UUID) error {
	return mapErr(s.q.DisableTOTP(ctx, userID))
}

func (s *Store) CreateOrg(ctx context.Context, name, slug string, ownerID uuid.UUID) (*domain.Org, error) {
	row, err := s.q.CreateOrg(ctx, gen.CreateOrgParams{
		Name: name, Slug: slug, OwnerUserID: uuid.NullUUID{UUID: ownerID, Valid: true},
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return orgFromRow(row, domain.RoleOwner), nil
}

func (s *Store) GetOrgByID(ctx context.Context, id uuid.UUID) (*domain.Org, error) {
	row, err := s.q.GetOrgByID(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return orgFromGetRow(row, domain.RoleMember), nil
}

func (s *Store) ListOrgsForUser(ctx context.Context, userID uuid.UUID) ([]domain.Org, error) {
	rows, err := s.q.ListOrgsForUser(ctx, userID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Org, 0, len(rows))
	for _, r := range rows {
		out = append(out, *orgFromListRow(r, domain.Role(r.MemberRole)))
	}
	return out, nil
}

func (s *Store) CreateMember(ctx context.Context, orgID, userID uuid.UUID, role domain.Role) error {
	return mapErr(s.q.CreateMember(ctx, gen.CreateMemberParams{OrgID: orgID, UserID: userID, Role: string(role)}))
}

func (s *Store) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*domain.Member, error) {
	row, err := s.q.GetMember(ctx, gen.GetMemberParams{OrgID: orgID, UserID: userID})
	if err != nil {
		return nil, mapErr(err)
	}
	return &domain.Member{OrgID: row.OrgID, UserID: row.UserID, Role: domain.Role(row.Role), CreatedAt: row.CreatedAt}, nil
}

func (s *Store) UpdateRole(ctx context.Context, orgID, userID uuid.UUID, role domain.Role) error {
	return mapErr(s.q.UpdateMemberRole(ctx, gen.UpdateMemberRoleParams{OrgID: orgID, UserID: userID, Role: string(role)}))
}

func (s *Store) DeleteMember(ctx context.Context, orgID, userID uuid.UUID) error {
	return mapErr(s.q.DeleteMember(ctx, gen.DeleteMemberParams{OrgID: orgID, UserID: userID}))
}

func (s *Store) CreateKey(ctx context.Context, orgID uuid.UUID, name, keyHash string, expiresAt *time.Time) (*domain.APIKey, error) {
	exp := sql.NullTime{}
	if expiresAt != nil {
		exp = sql.NullTime{Time: *expiresAt, Valid: true}
	}
	row, err := s.q.CreateAPIKey(ctx, gen.CreateAPIKeyParams{
		OrgID: orgID, Name: name, KeyHash: keyHash, ExpiresAt: exp,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return apiKeyFromRow(row), nil
}

func (s *Store) GetKeyByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	row, err := s.q.GetAPIKeyByHash(ctx, keyHash)
	if err != nil {
		return nil, mapErr(err)
	}
	return apiKeyFromRow(row), nil
}

func (s *Store) ListKeysByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.APIKey, error) {
	rows, err := s.q.ListAPIKeysByOrg(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.APIKey, 0, len(rows))
	for _, r := range rows {
		out = append(out, *apiKeyFromListRow(r))
	}
	return out, nil
}

func (s *Store) TouchKey(ctx context.Context, id uuid.UUID) error {
	return mapErr(s.q.TouchAPIKey(ctx, id))
}

func (s *Store) DeleteKey(ctx context.Context, id, orgID uuid.UUID) error {
	return mapErr(s.q.DeleteAPIKey(ctx, gen.DeleteAPIKeyParams{ID: id, OrgID: orgID}))
}

func (s *Store) Record(ctx context.Context, event domain.AuditEvent) error {
	return mapErr(s.q.CreateAuditLog(ctx, gen.CreateAuditLogParams{
		OrgID: event.OrgID, UserID: uuid.NullUUID{UUID: event.UserID, Valid: true},
		Action: event.Action, ResourceType: event.ResourceType, ResourceID: event.ResourceID, Details: event.Details,
	}))
}

func (s *Store) List(ctx context.Context, orgID uuid.UUID, limit int32) ([]domain.AuditEvent, error) {
	rows, err := s.q.ListAuditLogsByOrg(ctx, gen.ListAuditLogsByOrgParams{OrgID: orgID, Limit: limit})
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.AuditEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.AuditEvent{
			OrgID: r.OrgID, UserID: r.UserID.UUID, Action: r.Action,
			ResourceType: r.ResourceType, ResourceID: r.ResourceID, Details: r.Details,
			CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}
