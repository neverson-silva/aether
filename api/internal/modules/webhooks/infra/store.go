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

	"aether/internal/modules/webhooks/domain"
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

func (s *Store) CreateOutWebhook(ctx context.Context, hook *domain.OutWebhook) (*domain.OutWebhook, error) {
	row, err := s.q.CreateOutWebhook(ctx, gen.CreateOutWebhookParams{
		OrgID: hook.OrgID, Name: hook.Name, Url: hook.URL, SecretEnc: hook.SecretEnc,
		Events: hook.Events, Enabled: hook.Enabled,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return webhookFromRow(row), nil
}

func (s *Store) GetOutWebhook(ctx context.Context, id uuid.UUID) (*domain.OutWebhook, error) {
	row, err := s.q.GetOutWebhook(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return webhookFromRow(row), nil
}

func (s *Store) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.OutWebhook, error) {
	rows, err := s.q.ListOutWebhooksByOrg(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.OutWebhook, 0, len(rows))
	for _, r := range rows {
		out = append(out, *webhookFromRow(r))
	}
	return out, nil
}

func (s *Store) ListByEvent(ctx context.Context, event string) ([]domain.OutWebhook, error) {
	rows, err := s.q.ListEnabledWebhooksByEvent(ctx, event)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.OutWebhook, 0, len(rows))
	for _, r := range rows {
		out = append(out, *webhookFromRow(r))
	}
	return out, nil
}

func (s *Store) DeleteOutWebhook(ctx context.Context, id, orgID uuid.UUID) error {
	return mapErr(s.q.DeleteOutWebhook(ctx, gen.DeleteOutWebhookParams{ID: id, OrgID: orgID}))
}

func webhookFromRow(row gen.OutWebhook) *domain.OutWebhook {
	return &domain.OutWebhook{
		ID: row.ID, OrgID: row.OrgID, Name: row.Name, URL: row.Url,
		SecretEnc: row.SecretEnc, Events: row.Events, Enabled: row.Enabled, CreatedAt: row.CreatedAt,
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
