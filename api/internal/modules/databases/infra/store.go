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

	"aether/internal/modules/databases/domain"
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

func (s *Store) CreateDatabase(ctx context.Context, db *domain.Database) (*domain.Database, error) {
	row, err := s.q.CreateDatabase(ctx, gen.CreateDatabaseParams{
		OrgID: db.OrgID, ProjectID: db.ProjectID, Name: db.Name, Engine: string(db.Engine),
		Version: db.Version, Port: int32(db.Port), DbName: db.DBName, DbUser: db.User,
		PassEnc: db.PassEnc, MemMb: int32(db.MemMB), StorageMb: int32(db.StorageMB),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return databaseFromRow(row), nil
}

func (s *Store) GetDatabase(ctx context.Context, id uuid.UUID) (*domain.Database, error) {
	row, err := s.q.GetDatabase(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return databaseFromRow(row), nil
}

func (s *Store) ListDatabasesByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.Database, error) {
	rows, err := s.q.ListDatabasesByOrg(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Database, 0, len(rows))
	for _, r := range rows {
		out = append(out, *databaseFromRow(r))
	}
	return out, nil
}

func (s *Store) UpdateDatabaseStatus(ctx context.Context, id uuid.UUID, status, containerID string) error {
	return mapErr(s.q.UpdateDatabaseStatus(ctx, gen.UpdateDatabaseStatusParams{ID: id, Status: status, ContainerID: containerID}))
}

func (s *Store) UpdateDatabasePort(ctx context.Context, id uuid.UUID, port int) error {
	return mapErr(s.q.UpdateDatabasePort(ctx, gen.UpdateDatabasePortParams{ID: id, Port: int32(port)}))
}

func (s *Store) DeleteDatabase(ctx context.Context, id, orgID uuid.UUID) error {
	return mapErr(s.q.DeleteDatabase(ctx, gen.DeleteDatabaseParams{ID: id, OrgID: orgID}))
}

func databaseFromRow(row gen.Database) *domain.Database {
	return &domain.Database{
		ID: row.ID, OrgID: row.OrgID, ProjectID: row.ProjectID, Name: row.Name,
		Engine: domain.Engine(row.Engine), Version: row.Version, Port: int(row.Port),
		DBName: row.DbName, User: row.DbUser, PassEnc: row.PassEnc, MemMB: int(row.MemMb),
		StorageMB: int(row.StorageMb), Status: row.Status, ContainerID: row.ContainerID,
		CreatedAt: row.CreatedAt,
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
