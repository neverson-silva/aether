package infra

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"aether/internal/auth/domain"
	gen "aether/internal/infrastructure/pg/gen"
)

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return domain.ErrConflict
		case "23503":
			return domain.ErrConflict
		case "23502", "22P02":
			return domain.ErrValidation
		case "42P01", "42703":
			return domain.ErrValidation
		}
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

func userFromCreateRow(row gen.CreateUserRow) *domain.User {
	return userFromFields(row.ID, row.Email, row.Name, row.PasswordHash, row.GlobalRole, row.TotpEnabled, row.CreatedAt, row.UpdatedAt)
}

func userFromEmailRow(row gen.GetUserByEmailRow) *domain.User {
	return userFromFields(row.ID, row.Email, row.Name, row.PasswordHash, row.GlobalRole, row.TotpEnabled, row.CreatedAt, row.UpdatedAt)
}

func userFromIDRow(row gen.GetUserByIDRow) *domain.User {
	return userFromFields(row.ID, row.Email, row.Name, row.PasswordHash, row.GlobalRole, row.TotpEnabled, row.CreatedAt, row.UpdatedAt)
}

func userFromSecretRow(row gen.GetUserWithSecretRow) *domain.User {
	return &domain.User{
		ID: row.ID, Email: row.Email, Name: row.Name, PasswordHash: row.PasswordHash,
		GlobalRole: row.GlobalRole, TotpEnabled: row.TotpEnabled, TotpSecret: row.TotpSecret,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func userFromFields(id uuid.UUID, email, name, passwordHash, globalRole string, totpEnabled bool, createdAt, updatedAt time.Time) *domain.User {
	return &domain.User{
		ID: id, Email: email, Name: name, PasswordHash: passwordHash,
		GlobalRole: globalRole, TotpEnabled: totpEnabled, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

func orgFromRow(org gen.CreateOrgRow, role domain.Role) *domain.Org {
	return orgFromFields(org.ID, org.Name, org.Slug, org.Avatar, org.Color, org.OwnerUserID, role)
}

func orgFromGetRow(org gen.GetOrgByIDRow, role domain.Role) *domain.Org {
	return orgFromFields(org.ID, org.Name, org.Slug, org.Avatar, org.Color, org.OwnerUserID, role)
}

func orgFromFields(id uuid.UUID, name, slug string, avatar, color sql.NullString, owner uuid.NullUUID, role domain.Role) *domain.Org {
	avatarPtr, colorPtr := (*string)(nil), (*string)(nil)
	if avatar.Valid {
		avatarPtr = &avatar.String
	}
	if color.Valid {
		colorPtr = &color.String
	}
	return &domain.Org{ID: id, Name: name, Slug: slug, Avatar: avatarPtr, Color: colorPtr, OwnerID: owner.UUID, Role: role}
}

func orgFromListRow(org gen.ListOrgsForUserRow, role domain.Role) *domain.Org {
	avatar, color := (*string)(nil), (*string)(nil)
	if org.Avatar.Valid {
		avatar = &org.Avatar.String
	}
	if org.Color.Valid {
		color = &org.Color.String
	}
	return &domain.Org{ID: org.ID, Name: org.Name, Slug: org.Slug, Avatar: avatar, Color: color, OwnerID: org.OwnerUserID.UUID, Role: role}
}

func apiKeyFromRow(row gen.ApiKey) *domain.APIKey {
	var exp, last *time.Time
	if row.ExpiresAt.Valid {
		t := row.ExpiresAt.Time
		exp = &t
	}
	if row.LastUsedAt.Valid {
		t := row.LastUsedAt.Time
		last = &t
	}
	return &domain.APIKey{ID: row.ID, OrgID: row.OrgID, Name: row.Name, KeyHash: row.KeyHash, ExpiresAt: exp, LastUsed: last, CreatedAt: row.CreatedAt}
}

func apiKeyFromListRow(row gen.ListAPIKeysByOrgRow) *domain.APIKey {
	var exp, last *time.Time
	if row.ExpiresAt.Valid {
		t := row.ExpiresAt.Time
		exp = &t
	}
	if row.LastUsedAt.Valid {
		t := row.LastUsedAt.Time
		last = &t
	}
	return &domain.APIKey{ID: row.ID, OrgID: row.OrgID, Name: row.Name, ExpiresAt: exp, LastUsed: last, CreatedAt: row.CreatedAt}
}
