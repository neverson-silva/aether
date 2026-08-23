package infra

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"aether/internal/modules/apps/domain"
	gen "aether/internal/platform/infrastructure/pg/gen"
)

type Store struct {
	q      gen.Querier
	db     *sql.DB
	Cipher domain.SecretCipher
}

func NewStore(pool *pgxpool.Pool) *Store {
	db := stdlib.OpenDBFromPool(pool)
	return &Store{q: gen.New(db), db: db}
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) CreateProject(ctx context.Context, orgID uuid.UUID, name, slug, description, color string) (*domain.Project, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	q := gen.New(tx)
	row, err := q.CreateProject(ctx, gen.CreateProjectParams{
		OrgID: orgID, Name: name, Slug: slug, Description: description, Color: color,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	_, err = q.CreateEnvironment(ctx, gen.CreateEnvironmentParams{
		ProjectID: row.ID, Name: "Production", Slug: "production",
		Description: "Default production environment", IsDefault: true,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return projectFromCreateRow(row), nil
}

func (s *Store) GetProject(ctx context.Context, id, orgID uuid.UUID) (*domain.Project, error) {
	row, err := s.q.GetProject(ctx, gen.GetProjectParams{ID: id, OrgID: orgID})
	if err != nil {
		return nil, mapErr(err)
	}
	return projectFromGetRow(row), nil
}

func (s *Store) ListProjects(ctx context.Context, orgID uuid.UUID) ([]domain.Project, error) {
	rows, err := s.q.ListProjects(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Project, 0, len(rows))
	for _, r := range rows {
		out = append(out, *projectFromListRow(r))
	}
	return out, nil
}

func (s *Store) UpdateProject(ctx context.Context, id, orgID uuid.UUID, name, slug, description, color string) (*domain.Project, error) {
	row, err := s.q.UpdateProject(ctx, gen.UpdateProjectParams{
		ID: id, OrgID: orgID, Name: name, Slug: slug, Description: description, Color: color,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return projectFromUpdateRow(row), nil
}

func (s *Store) DeleteProject(ctx context.Context, id, orgID uuid.UUID) error {
	return mapErr(s.q.DeleteProject(ctx, gen.DeleteProjectParams{ID: id, OrgID: orgID}))
}

func (s *Store) CreateEnvironment(ctx context.Context, projectID uuid.UUID, name, slug, description, color string, isDefault bool) (*domain.Environment, error) {
	row, err := s.q.CreateEnvironment(ctx, gen.CreateEnvironmentParams{
		ProjectID: projectID, Name: name, Slug: slug, Description: description, Color: color, IsDefault: isDefault,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return envFromRow(row), nil
}

func (s *Store) GetEnvironment(ctx context.Context, id, projectID uuid.UUID) (*domain.Environment, error) {
	row, err := s.q.GetEnvironment(ctx, gen.GetEnvironmentParams{ID: id, ProjectID: projectID})
	if err != nil {
		return nil, mapErr(err)
	}
	return envFromRow(row), nil
}

func (s *Store) ListEnvironments(ctx context.Context, projectID uuid.UUID) ([]domain.Environment, error) {
	rows, err := s.q.ListEnvironments(ctx, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Environment, 0, len(rows))
	for _, r := range rows {
		out = append(out, *envFromRow(r))
	}
	return out, nil
}

func (s *Store) DefaultEnvironment(ctx context.Context, projectID uuid.UUID) (uuid.UUID, error) {
	row, err := s.q.DefaultEnvironment(ctx, projectID)
	if err != nil {
		return uuid.Nil, mapErr(err)
	}
	return row.ID, nil
}

func (s *Store) UpdateEnvironment(ctx context.Context, id, projectID uuid.UUID, name, slug, description, color string) (*domain.Environment, error) {
	row, err := s.q.UpdateEnvironment(ctx, gen.UpdateEnvironmentParams{
		ID: id, ProjectID: projectID, Name: name, Slug: slug, Description: description, Color: color,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return envFromRow(row), nil
}

func (s *Store) DeleteEnvironment(ctx context.Context, id, projectID uuid.UUID) error {
	return mapErr(s.q.DeleteEnvironment(ctx, gen.DeleteEnvironmentParams{ID: id, ProjectID: projectID}))
}

func (s *Store) CreateApp(ctx context.Context, app *domain.App) (*domain.App, error) {
	row, err := s.q.CreateApp(ctx, appParams(app))
	if err != nil {
		return nil, mapErr(err)
	}
	return appFromRow(row), nil
}

func (s *Store) GetApp(ctx context.Context, id, orgID uuid.UUID) (*domain.App, error) {
	row, err := s.q.GetApp(ctx, gen.GetAppParams{ID: id, OrgID: orgID})
	if err != nil {
		return nil, mapErr(err)
	}
	return appFromRow(row), nil
}

func (s *Store) GetAppByName(ctx context.Context, orgID uuid.UUID, name string) (*domain.App, error) {
	row, err := s.q.GetAppByName(ctx, gen.GetAppByNameParams{OrgID: orgID, Lower: name})
	if err != nil {
		return nil, mapErr(err)
	}
	return appFromRow(row), nil
}

func (s *Store) GetAppByID(ctx context.Context, id uuid.UUID) (*domain.App, error) {
	row, err := s.q.GetAppByID(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return appFromRow(row), nil
}

func (s *Store) ListAppsByProject(ctx context.Context, orgID, projectID uuid.UUID) ([]domain.App, error) {
	rows, err := s.q.ListAppsByProject(ctx, gen.ListAppsByProjectParams{OrgID: orgID, ProjectID: projectID})
	if err != nil {
		return nil, mapErr(err)
	}
	return appsFromRows(rows), nil
}

func (s *Store) ListAppsByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.App, error) {
	rows, err := s.q.ListAppsByOrg(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	return appsFromRows(rows), nil
}

func (s *Store) UpdateApp(ctx context.Context, app *domain.App) (*domain.App, error) {
	row, err := s.q.UpdateApp(ctx, gen.UpdateAppParams{
		ID: app.ID, OrgID: app.OrgID, Name: app.Name, Image: app.Image, GitUrl: app.GitURL,
		GitBranch: app.GitBranch, UploadID: app.UploadID, Dockerfile: app.Dockerfile, Port: int32(app.Port), Cpus: app.CPUs,
		MemMb: int32(app.MemMB), HcEnabled: app.HealthCheck.Enabled, HcPath: app.HealthCheck.Path,
		HcIntervalMs: int32(app.HealthCheck.IntervalMS), HcTimeoutMs: int32(app.HealthCheck.TimeoutMS),
		HcRetries: int32(app.HealthCheck.Retries), BuildType: app.BuildType,
		EnvironmentID: nullUUID(app.EnvironmentID), PreviewDomain: app.PreviewDomain,
		ImageRetention: int32(app.ImageRetention),
		StorageMb:      int32(app.StorageMB), InstallCommand: app.InstallCommand, BuildCommand: app.BuildCommand,
		StartCommand: app.StartCommand, RootFolder: app.RootFolder, DistFolder: app.DistFolder,
		WatchPaths: app.WatchPaths,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return appFromRow(row), nil
}

func (s *Store) DeleteApp(ctx context.Context, id, orgID uuid.UUID) error {
	return mapErr(s.q.DeleteApp(ctx, gen.DeleteAppParams{ID: id, OrgID: orgID}))
}

func (s *Store) SetWebhookSecret(ctx context.Context, id, orgID uuid.UUID, secret string) error {
	return mapErr(s.q.SetWebhookSecret(ctx, gen.SetWebhookSecretParams{ID: id, OrgID: orgID, WebhookSecret: secret}))
}

func (s *Store) UpsertEnvVar(ctx context.Context, appID uuid.UUID, name, value string, secret bool) error {
	stored := value
	if secret && s.Cipher != nil && stored != "" {
		enc, err := s.Cipher.Encrypt(stored)
		if err != nil {
			return err
		}
		stored = enc
	}
	return mapErr(s.q.UpsertAppEnv(ctx, gen.UpsertAppEnvParams{AppID: appID, Name: name, Value: stored, Secret: secret}))
}

func (s *Store) InsertMissingEnvVars(ctx context.Context, appID uuid.UUID, names []string) (int, error) {
	if len(names) == 0 {
		return 0, nil
	}
	inserted := 0
	for _, name := range names {
		result, err := s.db.ExecContext(ctx, "INSERT INTO app_env (app_id, name, value, secret) VALUES ($1, $2, '', false) ON CONFLICT (app_id, name) DO NOTHING", appID, name)
		if err != nil {
			return inserted, mapErr(err)
		}
		count, err := result.RowsAffected()
		if err != nil {
			return inserted, err
		}
		inserted += int(count)
	}
	return inserted, nil
}

func (s *Store) ListEnvVars(ctx context.Context, appID uuid.UUID) ([]domain.EnvVar, error) {
	rows, err := s.q.ListAppEnv(ctx, appID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.EnvVar, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.EnvVar{Name: r.Name, Value: r.Value, Secret: r.Secret})
	}
	return out, nil
}

func (s *Store) DeleteEnvVar(ctx context.Context, appID uuid.UUID, name string) error {
	return mapErr(s.q.DeleteAppEnv(ctx, gen.DeleteAppEnvParams{AppID: appID, Name: name}))
}
