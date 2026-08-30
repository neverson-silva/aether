package infra

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"aether/internal/modules/templates/domain"
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

func (s *Store) ListTemplates(ctx context.Context, filter domain.Filter) ([]domain.Template, error) {
	rows, err := s.q.ListTemplates(ctx, gen.ListTemplatesParams{
		Column1: filter.Category, Column2: filter.Search, Column3: filter.Featured,
		Column4: filter.Verified, Column5: filter.EditorsChoice,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Template, 0, len(rows))
	for _, r := range rows {
		out = append(out, *templateFromRow(gen.GetTemplateRow(r)))
	}
	return out, nil
}

func (s *Store) GetTemplate(ctx context.Context, id uuid.UUID) (*domain.Template, error) {
	row, err := s.q.GetTemplate(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return templateFromRow(row), nil
}

func (s *Store) IncrementInstalls(ctx context.Context, id uuid.UUID) error {
	return mapErr(s.q.IncrementTemplateInstalls(ctx, id))
}

func (s *Store) CreateComposeApp(ctx context.Context, app *domain.ComposeApp) (*domain.ComposeApp, error) {
	row, err := s.q.CreateComposeApp(ctx, gen.CreateComposeAppParams{
		OrgID: app.OrgID, ProjectID: app.ProjectID, EnvironmentID: nullUUIDPtr(app.EnvironmentID),
		Name: app.Name, Compose: app.Compose, Port: int32(app.Port), Status: app.Status,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return composeFromParts(row.ID, row.ServiceID, row.OrgID, row.ProjectID, row.EnvironmentID, row.Name, row.Compose, int(row.Port), row.Status, row.CreatedAt), nil
}

func (s *Store) NextComposePort(ctx context.Context) (int, error) {
	var port int
	err := s.db.QueryRowContext(ctx, `
SELECT candidate
FROM generate_series(1000, 1999) AS candidate
WHERE NOT EXISTS (SELECT 1 FROM compose_apps WHERE compose_apps.port = candidate)
  AND NOT EXISTS (SELECT 1 FROM apps WHERE apps.port = candidate)
  AND NOT EXISTS (SELECT 1 FROM databases WHERE databases.port = candidate)
ORDER BY candidate
LIMIT 1`).Scan(&port)
	return port, mapErr(err)
}

func (s *Store) GetComposeApp(ctx context.Context, id uuid.UUID) (*domain.ComposeApp, error) {
	row, err := s.q.GetComposeApp(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return composeFromParts(row.ID, row.ServiceID, row.OrgID, row.ProjectID, row.EnvironmentID, row.Name, row.Compose, int(row.Port), row.Status, row.CreatedAt), nil
}

func (s *Store) GetServiceID(ctx context.Context, composeID uuid.UUID) (uuid.UUID, error) {
	var serviceID uuid.UUID
	if err := s.db.QueryRowContext(ctx, `SELECT service_id FROM compose_apps WHERE id = $1`, composeID).Scan(&serviceID); err != nil {
		return uuid.Nil, mapErr(err)
	}
	return serviceID, nil
}

func (s *Store) ListComposeAppsByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.ComposeApp, error) {
	rows, err := s.q.ListComposeAppsByOrg(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.ComposeApp, 0, len(rows))
	for _, r := range rows {
		out = append(out, *composeFromParts(r.ID, r.ServiceID, r.OrgID, r.ProjectID, r.EnvironmentID, r.Name, r.Compose, int(r.Port), r.Status, r.CreatedAt))
	}
	return out, nil
}

func (s *Store) DeleteComposeApp(ctx context.Context, id, orgID uuid.UUID) error {
	return mapErr(s.q.DeleteComposeApp(ctx, gen.DeleteComposeAppParams{ID: id, OrgID: orgID}))
}

func (s *Store) SetComposeStatus(ctx context.Context, id uuid.UUID, status string) error {
	return mapErr(s.q.SetComposeStatus(ctx, gen.SetComposeStatusParams{ID: id, Status: status}))
}

func templateFromRow(row gen.GetTemplateRow) *domain.Template {
	return &domain.Template{
		ID: row.ID, Name: row.Name, Description: row.Description, Category: row.Category,
		Icon: row.Icon, Version: row.Version, Definition: row.Definition, Readme: row.Readme,
		Homepage: row.Homepage, GitHub: row.Github, License: row.License, Installs: int(row.Installs),
		Featured: row.Featured, Verified: row.Verified, EditorsChoice: row.EditorsChoice,
		Tags: row.Tags, UpdatedAt: row.UpdatedAt, ComposeYAML: row.ComposeYaml,
	}
}

func composeFromParts(id, serviceID, orgID, projectID uuid.UUID, environmentID uuid.NullUUID, name, compose string, port int, status string, createdAt time.Time) *domain.ComposeApp {
	var envID *uuid.UUID
	if environmentID.Valid {
		id := environmentID.UUID
		envID = &id
	}
	return &domain.ComposeApp{
		ID: id, ServiceID: serviceID, OrgID: orgID, ProjectID: projectID, EnvironmentID: envID,
		Name: name, Compose: compose, Port: port, Status: status, CreatedAt: createdAt,
	}
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

func nullUUIDPtr(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}
