package infra

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"aether/internal/modules/apps/domain"
	gen "aether/internal/platform/infrastructure/pg/gen"
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
		}
	}
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

func projectFromRow(row gen.Project) *domain.Project {
	return projectFromFields(row.ID, row.OrgID, row.Name, row.Slug, row.Description, row.Color, row.CreatedAt, row.UpdatedAt)
}

func projectFromCreateRow(row gen.CreateProjectRow) *domain.Project {
	return projectFromFields(row.ID, row.OrgID, row.Name, row.Slug, row.Description, row.Color, row.CreatedAt, row.UpdatedAt)
}

func projectFromGetRow(row gen.GetProjectRow) *domain.Project {
	return projectFromFields(row.ID, row.OrgID, row.Name, row.Slug, row.Description, row.Color, row.CreatedAt, row.UpdatedAt)
}

func projectFromListRow(row gen.ListProjectsRow) *domain.Project {
	return projectFromFields(row.ID, row.OrgID, row.Name, row.Slug, row.Description, row.Color, row.CreatedAt, row.UpdatedAt)
}

func projectFromUpdateRow(row gen.UpdateProjectRow) *domain.Project {
	return projectFromFields(row.ID, row.OrgID, row.Name, row.Slug, row.Description, row.Color, row.CreatedAt, row.UpdatedAt)
}

func projectFromFields(id, orgID uuid.UUID, name, slug, description, color string, createdAt, updatedAt time.Time) *domain.Project {
	return &domain.Project{
		ID: id, OrgID: orgID, Name: name, Slug: slug,
		Description: description, Color: color, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

func envFromRow(row gen.Environment) *domain.Environment {
	return &domain.Environment{
		ID: row.ID, ProjectID: row.ProjectID, Name: row.Name, Slug: row.Slug,
		Description: row.Description, Color: row.Color, IsDefault: row.IsDefault,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func appFromRow(row gen.App) *domain.App {
	return &domain.App{
		ID: row.ID, OrgID: row.OrgID, ProjectID: row.ProjectID,
		EnvironmentID: uuidPtr(row.EnvironmentID), Name: row.Name, SourceType: row.SourceType,
		Image: row.Image, GitURL: row.GitUrl, GitBranch: row.GitBranch, UploadID: row.UploadID, Dockerfile: row.Dockerfile,
		Port: int(row.Port), CPUs: row.Cpus, MemMB: int(row.MemMb),
		HealthCheck: domain.HealthCheck{
			Enabled: row.HcEnabled, Path: row.HcPath, IntervalMS: int(row.HcIntervalMs),
			TimeoutMS: int(row.HcTimeoutMs), Retries: int(row.HcRetries),
		},
		WebhookSecret: row.WebhookSecret, BuildType: row.BuildType, PreviewDomain: row.PreviewDomain,
		ImageRetention: int(row.ImageRetention), StorageMB: int(row.StorageMb),
		InstallCommand: row.InstallCommand, BuildCommand: row.BuildCommand, StartCommand: row.StartCommand,
		RootFolder: row.RootFolder, DistFolder: row.DistFolder, WatchPaths: row.WatchPaths,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func appsFromRows(rows []gen.App) []domain.App {
	out := make([]domain.App, 0, len(rows))
	for _, r := range rows {
		out = append(out, *appFromRow(r))
	}
	return out
}

func appParams(app *domain.App) gen.CreateAppParams {
	return gen.CreateAppParams{
		OrgID: app.OrgID, ProjectID: app.ProjectID, EnvironmentID: nullUUID(app.EnvironmentID),
		Name: app.Name, SourceType: app.SourceType, Image: app.Image, GitUrl: app.GitURL,
		GitBranch: app.GitBranch, UploadID: app.UploadID, Dockerfile: app.Dockerfile, Port: int32(app.Port), Cpus: app.CPUs,
		MemMb: int32(app.MemMB), HcEnabled: app.HealthCheck.Enabled, HcPath: app.HealthCheck.Path,
		HcIntervalMs: int32(app.HealthCheck.IntervalMS), HcTimeoutMs: int32(app.HealthCheck.TimeoutMS),
		HcRetries: int32(app.HealthCheck.Retries), BuildType: app.BuildType,
		PreviewDomain: app.PreviewDomain, ImageRetention: int32(app.ImageRetention),
		StorageMb: int32(app.StorageMB), InstallCommand: app.InstallCommand,
		BuildCommand: app.BuildCommand, StartCommand: app.StartCommand,
		RootFolder: app.RootFolder, DistFolder: app.DistFolder, WatchPaths: app.WatchPaths,
	}
}

func nullUUID(v *uuid.UUID) uuid.NullUUID {
	if v == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *v, Valid: true}
}

func uuidPtr(v uuid.NullUUID) *uuid.UUID {
	if !v.Valid {
		return nil
	}
	return &v.UUID
}
