package application

import (
	"context"
	"strings"

	"github.com/google/uuid"

	appsdomain "aether/internal/modules/apps/domain"
	"aether/internal/modules/variables/domain"
)

type Variables struct {
	Store domain.Store
	Apps  AppStore
}

type AppStore interface {
	GetProject(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.Project, error)
	GetEnvironment(ctx context.Context, id, projectID uuid.UUID) (*appsdomain.Environment, error)
	ListEnvironments(ctx context.Context, projectID uuid.UUID) ([]appsdomain.Environment, error)
	CreateEnvironment(ctx context.Context, projectID uuid.UUID, name, slug, description, color string, isDefault bool) (*appsdomain.Environment, error)
}

func (v *Variables) Set(ctx context.Context, projectID, orgID uuid.UUID, environmentID *uuid.UUID, userID uuid.UUID, key, value string, secret bool) (*domain.Variable, error) {
	if _, err := v.Apps.GetProject(ctx, projectID, orgID); err != nil {
		return nil, err
	}
	envID, err := v.scope(ctx, projectID, environmentID)
	if err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if key == "" || strings.Contains(key, "=") {
		return nil, domain.ErrValidation
	}
	variable, err := v.Store.UpsertVariable(ctx, &domain.Variable{
		ProjectID: projectID, EnvironmentID: envID, Key: key, Value: value, IsSecret: secret,
	})
	if err != nil {
		return nil, err
	}
	_ = v.Store.RecordAudit(ctx, projectID, scopePtr(envID), userID, "upsert", key)
	return variable, nil
}

func (v *Variables) List(ctx context.Context, projectID, orgID uuid.UUID, environmentID *uuid.UUID) ([]domain.Variable, error) {
	if _, err := v.Apps.GetProject(ctx, projectID, orgID); err != nil {
		return nil, err
	}
	envID, err := v.scope(ctx, projectID, environmentID)
	if err != nil {
		return nil, err
	}
	variables, err := v.Store.ListVariables(ctx, projectID, envID)
	if err != nil {
		return nil, err
	}
	masked := make([]domain.Variable, 0, len(variables))
	for _, variable := range variables {
		if variable.IsSecret && variable.Value != "" {
			variable.Value = "******"
		}
		masked = append(masked, variable)
	}
	return masked, nil
}

func (v *Variables) Delete(ctx context.Context, projectID, orgID uuid.UUID, environmentID *uuid.UUID, userID uuid.UUID, key string) error {
	if _, err := v.Apps.GetProject(ctx, projectID, orgID); err != nil {
		return err
	}
	envID, err := v.scope(ctx, projectID, environmentID)
	if err != nil {
		return err
	}
	if err := v.Store.DeleteVariable(ctx, projectID, envID, key); err != nil {
		return err
	}
	return v.Store.RecordAudit(ctx, projectID, scopePtr(envID), userID, "delete", key)
}

func (v *Variables) Audit(ctx context.Context, projectID, orgID uuid.UUID, limit int) ([]domain.AuditEvent, error) {
	if _, err := v.Apps.GetProject(ctx, projectID, orgID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return v.Store.ListAudit(ctx, projectID, limit)
}

func (v *Variables) AuditByEnvironment(ctx context.Context, projectID, orgID, environmentID uuid.UUID, limit int) ([]domain.AuditEvent, error) {
	if _, err := v.Apps.GetProject(ctx, projectID, orgID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	events, err := v.Store.ListAudit(ctx, projectID, limit)
	if err != nil {
		return nil, err
	}
	out := events[:0]
	for _, event := range events {
		if event.EnvironmentID != nil && *event.EnvironmentID == environmentID {
			out = append(out, event)
		}
	}
	return out, nil
}

func (v *Variables) Import(ctx context.Context, projectID, orgID uuid.UUID, environmentID *uuid.UUID, userID uuid.UUID, values map[string]string) error {
	if _, err := v.Apps.GetProject(ctx, projectID, orgID); err != nil {
		return err
	}
	envID, err := v.scope(ctx, projectID, environmentID)
	if err != nil {
		return err
	}
	for key, value := range values {
		if _, err := v.Store.UpsertVariable(ctx, &domain.Variable{
			ProjectID: projectID, EnvironmentID: envID, Key: key, Value: value,
		}); err != nil {
			return err
		}
	}
	return v.Store.RecordAudit(ctx, projectID, scopePtr(envID), userID, "import", "bulk")
}

func (v *Variables) Replace(ctx context.Context, projectID, orgID uuid.UUID, environmentID *uuid.UUID, userID uuid.UUID, entries map[string]domain.VariableInput) (int, error) {
	if _, err := v.Apps.GetProject(ctx, projectID, orgID); err != nil {
		return 0, err
	}
	envID, err := v.scope(ctx, projectID, environmentID)
	if err != nil {
		return 0, err
	}
	keys := make(map[string]bool, len(entries))
	for key, entry := range entries {
		key = strings.TrimSpace(key)
		if key == "" || strings.Contains(key, "=") {
			return 0, domain.ErrValidation
		}
		keys[key] = true
		if _, err := v.Store.UpsertVariable(ctx, &domain.Variable{
			ProjectID: projectID, EnvironmentID: envID, Key: key, Value: entry.Value, IsSecret: entry.Secret,
		}); err != nil {
			return 0, err
		}
	}
	existing, err := v.Store.ListVariables(ctx, projectID, envID)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, variable := range existing {
		if !keys[variable.Key] {
			if err := v.Store.DeleteVariable(ctx, projectID, envID, variable.Key); err != nil {
				return 0, err
			}
			deleted++
		}
	}
	if err := v.Store.RecordAudit(ctx, projectID, scopePtr(envID), userID, "replace", "bulk"); err != nil {
		return 0, err
	}
	return len(keys), nil
}

func (v *Variables) SetDefaultEnvironment(ctx context.Context, projectID, orgID, environmentID uuid.UUID) error {
	if _, err := v.Apps.GetProject(ctx, projectID, orgID); err != nil {
		return err
	}
	if _, err := v.Apps.GetEnvironment(ctx, environmentID, projectID); err != nil {
		return err
	}
	return v.Store.SetDefaultEnvironment(ctx, projectID, environmentID)
}

func (v *Variables) scope(ctx context.Context, projectID uuid.UUID, environmentID *uuid.UUID) (uuid.UUID, error) {
	if environmentID == nil {
		return uuid.Nil, nil
	}
	if _, err := v.Apps.GetEnvironment(ctx, *environmentID, projectID); err != nil {
		return uuid.Nil, err
	}
	return *environmentID, nil
}

func scopePtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}
