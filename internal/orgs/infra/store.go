package infra

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	gen "aether/internal/infrastructure/pg/gen"
	"aether/internal/orgs/domain"
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

func (s *Store) CreateOrg(ctx context.Context, name, slug string, ownerID uuid.UUID) (*domain.Org, error) {
	row, err := s.q.CreateOrg(ctx, gen.CreateOrgParams{
		Name: name, Slug: slug, OwnerUserID: uuid.NullUUID{UUID: ownerID, Valid: true},
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return &domain.Org{
		ID: row.ID, Name: row.Name, Slug: row.Slug,
		Avatar: nullStringPtr(row.Avatar), Color: nullStringPtr(row.Color),
		OwnerID: row.OwnerUserID.UUID, Role: domain.RoleOwner, CreatedAt: row.CreatedAt,
	}, nil
}

func (s *Store) CreateMember(ctx context.Context, orgID, userID uuid.UUID, role domain.Role) error {
	return mapErr(s.q.CreateMember(ctx, gen.CreateMemberParams{OrgID: orgID, UserID: userID, Role: string(role)}))
}

func (s *Store) GetOrg(ctx context.Context, id uuid.UUID) (*domain.Org, error) {
	row, err := s.q.GetOrg(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return &domain.Org{
		ID: row.ID, Name: row.Name, Slug: row.Slug,
		Avatar: nullStringPtr(row.Avatar), Color: nullStringPtr(row.Color),
		Description: row.Description, OwnerID: row.OwnerUserID.UUID, CreatedAt: row.CreatedAt,
	}, nil
}

func (s *Store) UpdateOrg(ctx context.Context, id uuid.UUID, name, description string, avatar, color *string) (*domain.Org, error) {
	row, err := s.q.UpdateOrg(ctx, gen.UpdateOrgParams{
		ID: id, Name: name, Description: description,
		Avatar: stringPtr(avatar), Color: stringPtr(color),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return &domain.Org{
		ID: row.ID, Name: row.Name, Slug: row.Slug,
		Avatar: nullStringPtr(row.Avatar), Color: nullStringPtr(row.Color),
		Description: row.Description, OwnerID: row.OwnerUserID.UUID, CreatedAt: row.CreatedAt,
	}, nil
}

func (s *Store) DeleteOrg(ctx context.Context, id, ownerID uuid.UUID) error {
	return mapErr(s.q.DeleteOrg(ctx, gen.DeleteOrgParams{ID: id, OwnerUserID: uuid.NullUUID{UUID: ownerID, Valid: true}}))
}

func (s *Store) ListOrgsByUser(ctx context.Context, userID uuid.UUID) ([]domain.Org, error) {
	rows, err := s.q.ListOrgsForUser(ctx, userID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Org, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Org{
			ID: r.ID, Name: r.Name, Slug: r.Slug, Avatar: nullStringPtr(r.Avatar),
			Color: nullStringPtr(r.Color), OwnerID: r.OwnerUserID.UUID,
			Role: domain.Role(r.MemberRole), CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

func (s *Store) GetRole(ctx context.Context, orgID, userID uuid.UUID) (domain.Role, error) {
	role, err := s.q.GetOrgRole(ctx, gen.GetOrgRoleParams{OrgID: orgID, UserID: userID})
	if err != nil {
		return "", mapErr(err)
	}
	return domain.Role(role), nil
}

func (s *Store) ListMembers(ctx context.Context, orgID uuid.UUID) ([]domain.Member, error) {
	rows, err := s.q.ListOrgMembers(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Member, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Member{
			OrgID: r.OrgID, UserID: r.UserID, Role: domain.Role(r.Role),
			Email: r.Email, Name: r.Name, CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

func (s *Store) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*domain.Member, error) {
	row, err := s.q.GetMember(ctx, gen.GetMemberParams{OrgID: orgID, UserID: userID})
	if err != nil {
		return nil, mapErr(err)
	}
	return &domain.Member{OrgID: row.OrgID, UserID: row.UserID, Role: domain.Role(row.Role), CreatedAt: row.CreatedAt}, nil
}

func (s *Store) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role domain.Role) error {
	return mapErr(s.q.UpdateMemberRole(ctx, gen.UpdateMemberRoleParams{OrgID: orgID, UserID: userID, Role: string(role)}))
}

func (s *Store) RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error {
	return mapErr(s.q.DeleteMember(ctx, gen.DeleteMemberParams{OrgID: orgID, UserID: userID}))
}

func (s *Store) SetAssignment(ctx context.Context, orgID, userID, projectID uuid.UUID) error {
	return mapErr(s.q.SetProjectAssignment(ctx, gen.SetProjectAssignmentParams{OrgID: orgID, UserID: userID, ProjectID: projectID}))
}

func (s *Store) RemoveAssignment(ctx context.Context, orgID, userID, projectID uuid.UUID) error {
	return mapErr(s.q.RemoveProjectAssignment(ctx, gen.RemoveProjectAssignmentParams{OrgID: orgID, UserID: userID, ProjectID: projectID}))
}

func (s *Store) ListAssignments(ctx context.Context, orgID uuid.UUID) ([]domain.Assignment, error) {
	rows, err := s.q.ListProjectAssignments(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Assignment, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Assignment{
			ID: r.ID, OrgID: r.OrgID, UserID: r.UserID, ProjectID: r.ProjectID, CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

func (s *Store) RecordAudit(ctx context.Context, orgID, userID uuid.UUID, action, resource, details string) error {
	return mapErr(s.q.CreateAuditLog(ctx, gen.CreateAuditLogParams{
		OrgID: orgID, UserID: uuid.NullUUID{UUID: userID, Valid: true},
		Action: action, ResourceType: resource, ResourceID: "", Details: details,
	}))
}

func (s *Store) ListAudit(ctx context.Context, orgID uuid.UUID, limit int) ([]domain.AuditEvent, error) {
	rows, err := s.q.ListAuditLogsByOrg(ctx, gen.ListAuditLogsByOrgParams{OrgID: orgID, Limit: int32(limit)})
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.AuditEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.AuditEvent{
			ID: r.ID, Action: r.Action, UserID: r.UserID.UUID,
			Resource: r.ResourceType, Details: r.Details, CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func stringPtr(v *string) sql.NullString {
	if v == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *v, Valid: true}
}

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
		case "23502", "22P02", "23514":
			return domain.ErrValidation
		}
	}
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}
