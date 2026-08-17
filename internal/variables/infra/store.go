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
	"aether/internal/variables/domain"
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

func (s *Store) UpsertVariable(ctx context.Context, variable *domain.Variable) (*domain.Variable, error) {
	value := variable.Value
	if variable.IsSecret && s.Cipher != nil && value != "" {
		enc, err := s.Cipher.Encrypt(value)
		if err != nil {
			return nil, err
		}
		value = enc
	}
	if variable.EnvironmentID == uuid.Nil {
		row, err := s.q.UpsertProjectVariable(ctx, gen.UpsertProjectVariableParams{
			ProjectID: variable.ProjectID, Key: variable.Key, Value: value, IsSecret: variable.IsSecret,
		})
		if err != nil {
			return nil, mapErr(err)
		}
		return variableFromRow(row), nil
	}
	row, err := s.q.UpsertEnvVariable(ctx, gen.UpsertEnvVariableParams{
		ProjectID: variable.ProjectID, EnvironmentID: validUUID(variable.EnvironmentID),
		Key: variable.Key, Value: value, IsSecret: variable.IsSecret,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return variableFromRow(row), nil
}

func (s *Store) ListVariables(ctx context.Context, projectID, environmentID uuid.UUID) ([]domain.Variable, error) {
	var rows []gen.EnvVariable
	var err error
	if environmentID == uuid.Nil {
		rows, err = s.q.ListProjectVariables(ctx, projectID)
	} else {
		rows, err = s.q.ListEnvVariables(ctx, gen.ListEnvVariablesParams{ProjectID: projectID, EnvironmentID: validUUID(environmentID)})
	}
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Variable, 0, len(rows))
	for _, r := range rows {
		out = append(out, *variableFromRow(r))
	}
	return out, nil
}

func (s *Store) DeleteVariable(ctx context.Context, projectID, environmentID uuid.UUID, key string) error {
	if environmentID == uuid.Nil {
		return mapErr(s.q.DeleteProjectVariable(ctx, gen.DeleteProjectVariableParams{ProjectID: projectID, Key: key}))
	}
	return mapErr(s.q.DeleteEnvVariable(ctx, gen.DeleteEnvVariableParams{
		ProjectID: projectID, EnvironmentID: validUUID(environmentID), Key: key,
	}))
}

func (s *Store) RecordAudit(ctx context.Context, projectID uuid.UUID, environmentID *uuid.UUID, userID uuid.UUID, action, key string) error {
	return mapErr(s.q.RecordVariableAudit(ctx, gen.RecordVariableAuditParams{
		ProjectID: projectID, EnvironmentID: nullUUID(environmentID),
		UserID: uuid.NullUUID{UUID: userID, Valid: true}, Action: action, Key: key,
	}))
}

func (s *Store) ListAudit(ctx context.Context, projectID uuid.UUID, limit int) ([]domain.AuditEvent, error) {
	rows, err := s.q.ListVariableAudit(ctx, gen.ListVariableAuditParams{ProjectID: projectID, Limit: int32(limit)})
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.AuditEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.AuditEvent{
			ID: r.ID, ProjectID: r.ProjectID, EnvironmentID: uuidPtr(r.EnvironmentID),
			UserID: r.UserID.UUID, Action: r.Action, Key: r.Key, CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

func (s *Store) SetDefaultEnvironment(ctx context.Context, projectID, environmentID uuid.UUID) error {
	return mapErr(s.q.SetDefaultEnvironment(ctx, gen.SetDefaultEnvironmentParams{
		ProjectID: projectID, ID: environmentID,
	}))
}

func variableFromRow(row gen.EnvVariable) *domain.Variable {
	var envID uuid.UUID
	if row.EnvironmentID.Valid {
		envID = row.EnvironmentID.UUID
	}
	return &domain.Variable{
		ID: row.ID, ProjectID: row.ProjectID, EnvironmentID: envID,
		Key: row.Key, Value: row.Value, IsSecret: row.IsSecret,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func nullUUID(v *uuid.UUID) uuid.NullUUID {
	if v == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *v, Valid: true}
}

func validUUID(id uuid.UUID) uuid.NullUUID {
	return uuid.NullUUID{UUID: id, Valid: true}
}

func uuidPtr(v uuid.NullUUID) *uuid.UUID {
	if !v.Valid {
		return nil
	}
	return &v.UUID
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
