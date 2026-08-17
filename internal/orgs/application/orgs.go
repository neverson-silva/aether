package application

import (
	"context"
	"strings"

	"github.com/google/uuid"

	appsdomain "aether/internal/apps/domain"
	"aether/internal/orgs/domain"
)

type Organizations struct {
	Store     domain.Store
	Apps      AppStore
	Databases ExportDatabaseStore
	Domains   DomainLister
}

type AppStore interface {
	GetProject(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.Project, error)
}

func (o *Organizations) Create(ctx context.Context, userID uuid.UUID, name string) (*domain.Org, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrValidation
	}
	org, err := o.Store.CreateOrg(ctx, name, slugify(name), userID)
	if err != nil {
		return nil, err
	}
	if err := o.Store.CreateMember(ctx, org.ID, userID, domain.RoleOwner); err != nil {
		return nil, err
	}
	_ = o.Store.RecordAudit(ctx, org.ID, userID, "org.create", "org", name)
	return org, nil
}

func (o *Organizations) List(ctx context.Context, userID uuid.UUID) ([]domain.Org, error) {
	return o.Store.ListOrgsByUser(ctx, userID)
}

func (o *Organizations) Get(ctx context.Context, orgID, userID uuid.UUID) (*domain.Org, error) {
	org, err := o.Store.GetOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	role, err := o.Store.GetRole(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}
	org.Role = role
	return org, nil
}

func (o *Organizations) Update(ctx context.Context, orgID, userID uuid.UUID, name, description string, avatar, color *string) (*domain.Org, error) {
	if err := o.requireManager(ctx, orgID, userID); err != nil {
		return nil, err
	}
	org, err := o.Store.UpdateOrg(ctx, orgID, strings.TrimSpace(name), description, avatar, color)
	if err != nil {
		return nil, err
	}
	_ = o.Store.RecordAudit(ctx, orgID, userID, "org.update", "org", org.Name)
	return org, nil
}

func (o *Organizations) Delete(ctx context.Context, orgID, userID uuid.UUID) error {
	if err := o.Store.DeleteOrg(ctx, orgID, userID); err != nil {
		return err
	}
	_ = o.Store.RecordAudit(ctx, orgID, userID, "org.delete", "org", orgID.String())
	return nil
}

func (o *Organizations) Members(ctx context.Context, orgID, userID uuid.UUID) ([]domain.Member, error) {
	if _, err := o.requireRole(ctx, orgID, userID); err != nil {
		return nil, err
	}
	return o.Store.ListMembers(ctx, orgID)
}

func (o *Organizations) UpdateMember(ctx context.Context, orgID, actorID, targetID uuid.UUID, role domain.Role) error {
	if err := o.requireManager(ctx, orgID, actorID); err != nil {
		return err
	}
	if !role.Valid() {
		return domain.ErrValidation
	}
	if err := o.Store.UpdateMemberRole(ctx, orgID, targetID, role); err != nil {
		return err
	}
	return o.Store.RecordAudit(ctx, orgID, actorID, "member.update", "member", targetID.String())
}

func (o *Organizations) RemoveMember(ctx context.Context, orgID, actorID, targetID uuid.UUID) error {
	if err := o.requireManager(ctx, orgID, actorID); err != nil {
		return err
	}
	if err := o.Store.RemoveMember(ctx, orgID, targetID); err != nil {
		return err
	}
	return o.Store.RecordAudit(ctx, orgID, actorID, "member.remove", "member", targetID.String())
}

func (o *Organizations) AssignProject(ctx context.Context, orgID, actorID, userID, projectID uuid.UUID) error {
	if err := o.requireManager(ctx, orgID, actorID); err != nil {
		return err
	}
	if _, err := o.Apps.GetProject(ctx, projectID, orgID); err != nil {
		return err
	}
	if err := o.Store.SetAssignment(ctx, orgID, userID, projectID); err != nil {
		return err
	}
	return o.Store.RecordAudit(ctx, orgID, actorID, "assignment.set", "project", projectID.String())
}

func (o *Organizations) RemoveAssignment(ctx context.Context, orgID, actorID, userID, projectID uuid.UUID) error {
	if err := o.requireManager(ctx, orgID, actorID); err != nil {
		return err
	}
	if err := o.Store.RemoveAssignment(ctx, orgID, userID, projectID); err != nil {
		return err
	}
	return o.Store.RecordAudit(ctx, orgID, actorID, "assignment.remove", "project", projectID.String())
}

func (o *Organizations) Assignments(ctx context.Context, orgID, userID uuid.UUID) ([]domain.Assignment, error) {
	if _, err := o.requireRole(ctx, orgID, userID); err != nil {
		return nil, err
	}
	return o.Store.ListAssignments(ctx, orgID)
}

func (o *Organizations) Audit(ctx context.Context, orgID, userID uuid.UUID, limit int) ([]domain.AuditEvent, error) {
	if _, err := o.requireRole(ctx, orgID, userID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return o.Store.ListAudit(ctx, orgID, limit)
}

func (o *Organizations) requireManager(ctx context.Context, orgID, userID uuid.UUID) error {
	role, err := o.Store.GetRole(ctx, orgID, userID)
	if err != nil {
		return err
	}
	if !role.CanManage() {
		return domain.ErrForbidden
	}
	return nil
}

func (o *Organizations) requireRole(ctx context.Context, orgID, userID uuid.UUID) (domain.Role, error) {
	role, err := o.Store.GetRole(ctx, orgID, userID)
	if err != nil {
		return "", err
	}
	return role, nil
}

func slugify(s string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			out.WriteByte('-')
		}
	}
	slug := strings.Trim(out.String(), "-")
	if slug == "" {
		slug = "org-" + uuid.NewString()[:8]
	}
	return slug
}
