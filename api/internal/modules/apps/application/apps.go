package application

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"aether/internal/modules/apps/domain"
)

type Apps struct {
	Store           domain.Store
	Secrets         domain.SecretCipher
	Containers      ContainerRemover
	Databases       DatabaseNameStore
	ServiceIdentity func(context.Context, uuid.UUID) (uuid.UUID, error)
	// LatestDeployments returns the status of the most recent deployment per
	// app. When nil, the apps list is served without deployment status.
	LatestDeployments func(ctx context.Context, appIDs []uuid.UUID) (map[uuid.UUID]string, error)
}

func (a *Apps) GetServiceID(ctx context.Context, appID uuid.UUID) (uuid.UUID, error) {
	if a.ServiceIdentity == nil {
		return appID, nil
	}
	return a.ServiceIdentity(ctx, appID)
}

type ContainerRemover interface {
	Delete(ctx context.Context, appID, orgID uuid.UUID) error
}

type DatabaseNameStore interface {
	HasName(ctx context.Context, orgID uuid.UUID, name string) (bool, error)
}

var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func (a *Apps) CreateProject(ctx context.Context, orgID uuid.UUID, name, description, color string) (*domain.Project, error) {
	name = strings.TrimSpace(name)
	if err := validateName(name); err != nil {
		return nil, err
	}
	return a.Store.CreateProject(ctx, orgID, name, slugify(name), strings.TrimSpace(description), strings.TrimSpace(color))
}

func (a *Apps) GetProject(ctx context.Context, id, orgID uuid.UUID) (*domain.Project, error) {
	return a.Store.GetProject(ctx, id, orgID)
}

func (a *Apps) ListProjects(ctx context.Context, orgID uuid.UUID) ([]domain.Project, error) {
	return a.Store.ListProjects(ctx, orgID)
}

func (a *Apps) UpdateProject(ctx context.Context, id, orgID uuid.UUID, name, description, color string) (*domain.Project, error) {
	name = strings.TrimSpace(name)
	if err := validateName(name); err != nil {
		return nil, err
	}
	return a.Store.UpdateProject(ctx, id, orgID, name, slugify(name), strings.TrimSpace(description), strings.TrimSpace(color))
}

func (a *Apps) DeleteProject(ctx context.Context, id, orgID uuid.UUID) error {
	return a.Store.DeleteProject(ctx, id, orgID)
}

func (a *Apps) CreateEnvironment(ctx context.Context, projectID uuid.UUID, name, description, color string, isDefault bool) (*domain.Environment, error) {
	name = strings.TrimSpace(name)
	if err := validateName(name); err != nil {
		return nil, err
	}
	env, err := a.Store.CreateEnvironment(ctx, projectID, name, slugify(name), strings.TrimSpace(description), strings.TrimSpace(color), isDefault)
	if errors.Is(err, domain.ErrConflict) {
		return nil, domain.ErrConflict
	}
	return env, err
}

func (a *Apps) GetEnvironment(ctx context.Context, id, projectID uuid.UUID) (*domain.Environment, error) {
	return a.Store.GetEnvironment(ctx, id, projectID)
}

func (a *Apps) ListEnvironments(ctx context.Context, projectID uuid.UUID) ([]domain.Environment, error) {
	return a.Store.ListEnvironments(ctx, projectID)
}

func (a *Apps) DefaultEnvironment(ctx context.Context, projectID uuid.UUID) (uuid.UUID, error) {
	return a.Store.DefaultEnvironment(ctx, projectID)
}

func (a *Apps) UpdateEnvironment(ctx context.Context, id, projectID uuid.UUID, name, description, color string) (*domain.Environment, error) {
	name = strings.TrimSpace(name)
	if err := validateName(name); err != nil {
		return nil, err
	}
	return a.Store.UpdateEnvironment(ctx, id, projectID, name, slugify(name), strings.TrimSpace(description), strings.TrimSpace(color))
}

func (a *Apps) DeleteEnvironment(ctx context.Context, id, projectID uuid.UUID) error {
	return a.Store.DeleteEnvironment(ctx, id, projectID)
}

func (a *Apps) CreateApp(ctx context.Context, orgID, projectID uuid.UUID, app *domain.App) (*domain.App, error) {
	app.OrgID = orgID
	app.ProjectID = projectID
	app.Name = strings.TrimSpace(app.Name)
	if app.EnvironmentID != nil {
		if _, err := a.Store.GetEnvironment(ctx, *app.EnvironmentID, projectID); err != nil {
			return nil, err
		}
	} else if envID, err := a.Store.DefaultEnvironment(ctx, projectID); err == nil {
		app.EnvironmentID = &envID
	}
	if err := validateApp(app); err != nil {
		return nil, err
	}
	if a.Databases != nil {
		exists, err := a.Databases.HasName(ctx, orgID, app.Name)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, domain.ErrConflict
		}
	}
	applyAppDefaults(app)
	created, err := a.Store.CreateApp(ctx, app)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (a *Apps) GetApp(ctx context.Context, id, orgID uuid.UUID) (*domain.App, error) {
	return a.Store.GetApp(ctx, id, orgID)
}

func (a *Apps) ListApps(ctx context.Context, orgID uuid.UUID, projectID *uuid.UUID) ([]domain.App, error) {
	if projectID != nil {
		return a.Store.ListAppsByProject(ctx, orgID, *projectID)
	}
	return a.Store.ListAppsByOrg(ctx, orgID)
}

func (a *Apps) UpdateApp(ctx context.Context, id, orgID uuid.UUID, app *domain.App) (*domain.App, error) {
	existing, err := a.Store.GetApp(ctx, id, orgID)
	if err != nil {
		return nil, err
	}
	app.ID = id
	app.OrgID = orgID
	app.ProjectID = existing.ProjectID
	if app.Name == "" {
		app.Name = existing.Name
	}
	if app.SourceType == "" {
		app.SourceType = existing.SourceType
	}
	if app.Port == 0 {
		app.Port = existing.Port
	}
	if app.Dockerfile == "" {
		app.Dockerfile = existing.Dockerfile
	}
	if app.Image == "" {
		app.Image = existing.Image
	}
	if app.GitURL == "" {
		app.GitURL = existing.GitURL
	}
	if app.UploadID == "" {
		app.UploadID = existing.UploadID
	}
	if app.BuildType == "" {
		app.BuildType = existing.BuildType
	}
	if app.EnvironmentID == nil {
		app.EnvironmentID = existing.EnvironmentID
	}
	if err := validateApp(app); err != nil {
		return nil, err
	}
	applyAppDefaults(app)
	return a.Store.UpdateApp(ctx, app)
}

func (a *Apps) DeleteApp(ctx context.Context, id, orgID uuid.UUID) error {
	if a.Containers != nil {
		_ = a.Containers.Delete(ctx, id, orgID)
	}
	return a.Store.DeleteApp(ctx, id, orgID)
}

func (a *Apps) SetEnv(ctx context.Context, appID, orgID uuid.UUID, name, value string, secret bool) error {
	if _, err := a.Store.GetApp(ctx, appID, orgID); err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" || strings.Contains(name, "=") {
		return domain.ErrValidation
	}
	return a.Store.UpsertEnvVar(ctx, appID, name, value, secret)
}

func (a *Apps) ImportMissingEnvVars(ctx context.Context, appID, orgID uuid.UUID, names []string) (int, error) {
	if _, err := a.Store.GetApp(ctx, appID, orgID); err != nil {
		return 0, err
	}
	unique := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || strings.Contains(name, "=") {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		unique = append(unique, name)
	}
	return a.Store.InsertMissingEnvVars(ctx, appID, unique)
}

func (a *Apps) ListEnv(ctx context.Context, appID, orgID uuid.UUID) ([]domain.EnvVar, error) {
	if _, err := a.Store.GetApp(ctx, appID, orgID); err != nil {
		return nil, err
	}
	vars, err := a.Store.ListEnvVars(ctx, appID)
	if err != nil {
		return nil, err
	}
	masked := make([]domain.EnvVar, 0, len(vars))
	for _, v := range vars {
		if v.Secret && v.Value != "" {
			v.Value = "******"
		}
		masked = append(masked, v)
	}
	return masked, nil
}

func (a *Apps) DeleteEnv(ctx context.Context, appID, orgID uuid.UUID, name string) error {
	if _, err := a.Store.GetApp(ctx, appID, orgID); err != nil {
		return err
	}
	return a.Store.DeleteEnvVar(ctx, appID, name)
}

func (a *Apps) SetWebhook(ctx context.Context, appID, orgID uuid.UUID, secret string) error {
	if _, err := a.Store.GetApp(ctx, appID, orgID); err != nil {
		return err
	}
	if secret == "" {
		return a.Store.SetWebhookSecret(ctx, appID, orgID, "")
	}
	if a.Secrets == nil {
		return domain.ErrValidation
	}
	enc, err := a.Secrets.Encrypt(secret)
	if err != nil {
		return err
	}
	return a.Store.SetWebhookSecret(ctx, appID, orgID, enc)
}

func validateName(name string) error {
	if len(name) < 1 || len(name) > 64 {
		return domain.ErrValidation
	}
	return nil
}

func validateApp(app *domain.App) error {
	if len(app.Name) < 1 || len(app.Name) > 64 || !namePattern.MatchString(app.Name) {
		return domain.ErrValidation
	}
	switch app.SourceType {
	case "image", "git", "upload":
	default:
		return domain.ErrValidation
	}
	if app.SourceType == "image" && app.Image == "" {
		return domain.ErrValidation
	}
	if app.Port < 0 || app.Port > 65535 {
		return domain.ErrValidation
	}
	if app.MemMB < 0 || app.StorageMB < 0 || app.ImageRetention < 0 {
		return domain.ErrValidation
	}
	if app.HealthCheck.IntervalMS < 0 || app.HealthCheck.TimeoutMS < 0 || app.HealthCheck.Retries < 0 {
		return domain.ErrValidation
	}
	return nil
}

func applyAppDefaults(app *domain.App) {
	if app.SourceType == "" {
		app.SourceType = "image"
	}
	if app.BuildType == "" {
		app.BuildType = "buildpacks"
	}
	if app.Dockerfile == "" {
		app.Dockerfile = "Dockerfile"
	}
	if app.Port == 0 {
		app.Port = 80
	}
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
		slug = "item-" + uuid.NewString()[:8]
	}
	return slug
}
