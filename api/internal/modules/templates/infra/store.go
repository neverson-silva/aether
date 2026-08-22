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
		Name: app.Name, Compose: app.Compose, Status: app.Status,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return composeFromRow(gen.CreateComposeAppRow(row)), nil
}

func (s *Store) GetComposeApp(ctx context.Context, id uuid.UUID) (*domain.ComposeApp, error) {
	row, err := s.q.GetComposeApp(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return composeFromRow(gen.CreateComposeAppRow(row)), nil
}

func (s *Store) ListComposeAppsByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.ComposeApp, error) {
	rows, err := s.q.ListComposeAppsByOrg(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.ComposeApp, 0, len(rows))
	for _, r := range rows {
		out = append(out, *composeFromRow(gen.CreateComposeAppRow(r)))
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

func composeFromRow(row gen.CreateComposeAppRow) *domain.ComposeApp {
	var envID *uuid.UUID
	if row.EnvironmentID.Valid {
		id := row.EnvironmentID.UUID
		envID = &id
	}
	return &domain.ComposeApp{
		ID: row.ID, OrgID: row.OrgID, ProjectID: row.ProjectID, EnvironmentID: envID,
		Name: row.Name, Compose: row.Compose, Status: row.Status, CreatedAt: row.CreatedAt,
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
