package infra

import (
	"database/sql"
	"errors"
	"reflect"
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

func appFromRow(row any) *domain.App {
	value := reflect.ValueOf(row)
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	field := func(name string) reflect.Value {
		return value.FieldByName(name)
	}
	uuidValue := func(name string) uuid.UUID { return field(name).Interface().(uuid.UUID) }
	stringValue := func(name string) string { return field(name).String() }
	int32Value := func(name string) int32 { return int32(field(name).Int()) }
	timeValue := func(name string) time.Time { return field(name).Interface().(time.Time) }
	nullEnvironment := field("EnvironmentID").Interface().(uuid.NullUUID)
	return &domain.App{
		ID: uuidValue("ID"), OrgID: uuidValue("OrgID"), ProjectID: uuidValue("ProjectID"),
		EnvironmentID: uuidPtr(nullEnvironment), Name: stringValue("Name"), SourceType: stringValue("SourceType"),
		Image: stringValue("Image"), GitURL: stringValue("GitUrl"), GitBranch: stringValue("GitBranch"), UploadID: stringValue("UploadID"), Dockerfile: stringValue("Dockerfile"), ComposeFile: stringValue("ComposeFile"),
		Port: int(int32Value("Port")), CPUs: stringValue("Cpus"), MemMB: int(int32Value("MemMb")),
		HealthCheck: domain.HealthCheck{
			Enabled: field("HcEnabled").Bool(), Path: stringValue("HcPath"), IntervalMS: int(int32Value("HcIntervalMs")),
			TimeoutMS: int(int32Value("HcTimeoutMs")), Retries: int(int32Value("HcRetries")),
		},
		WebhookSecret: stringValue("WebhookSecret"), BuildType: stringValue("BuildType"), PreviewDomain: stringValue("PreviewDomain"),
		ImageRetention: int(int32Value("ImageRetention")), StorageMB: int(int32Value("StorageMb")),
		InstallCommand: stringValue("InstallCommand"), BuildCommand: stringValue("BuildCommand"), StartCommand: stringValue("StartCommand"),
		RootFolder: stringValue("RootFolder"), DistFolder: stringValue("DistFolder"), WatchPaths: stringValue("WatchPaths"),
		CreatedAt: timeValue("CreatedAt"), UpdatedAt: timeValue("UpdatedAt"),
	}
}

func appsFromRows(rows any) []domain.App {
	value := reflect.ValueOf(rows)
	out := make([]domain.App, 0, value.Len())
	for index := 0; index < value.Len(); index++ {
		out = append(out, *appFromRow(value.Index(index).Interface()))
	}
	return out
}

func appParams(app *domain.App) gen.CreateAppParams {
	return gen.CreateAppParams{
		OrgID: app.OrgID, ProjectID: app.ProjectID, EnvironmentID: nullUUID(app.EnvironmentID),
		Name: app.Name, SourceType: app.SourceType, Image: app.Image, GitUrl: app.GitURL,
		GitBranch: app.GitBranch, UploadID: app.UploadID, Dockerfile: app.Dockerfile, ComposeFile: app.ComposeFile, Port: int32(app.Port), Cpus: app.CPUs,
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
