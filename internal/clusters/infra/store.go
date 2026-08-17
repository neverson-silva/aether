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

	"aether/internal/clusters/domain"
	gen "aether/internal/infrastructure/pg/gen"
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

func (s *Store) CreateCluster(ctx context.Context, cluster *domain.Cluster) (*domain.Cluster, error) {
	row, err := s.q.CreateCluster(ctx, gen.CreateClusterParams{
		OrgID: cluster.OrgID, Name: cluster.Name, Labels: cluster.Labels,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return clusterFromRow(row), nil
}

func (s *Store) GetCluster(ctx context.Context, id uuid.UUID) (*domain.Cluster, error) {
	row, err := s.q.GetCluster(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return clusterFromRow(row), nil
}

func (s *Store) ListClustersByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.Cluster, error) {
	rows, err := s.q.ListClustersByOrg(ctx, orgID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Cluster, 0, len(rows))
	for _, r := range rows {
		out = append(out, *clusterFromRow(r))
	}
	return out, nil
}

func (s *Store) DeleteCluster(ctx context.Context, id, orgID uuid.UUID) error {
	return mapErr(s.q.DeleteCluster(ctx, gen.DeleteClusterParams{ID: id, OrgID: orgID}))
}

func (s *Store) ListServers(ctx context.Context) ([]domain.Server, error) {
	rows, err := s.q.ListServers(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]domain.Server, 0, len(rows))
	for _, r := range rows {
		out = append(out, *serverFromRow(r))
	}
	return out, nil
}

func (s *Store) GetServer(ctx context.Context, id uuid.UUID) (*domain.Server, error) {
	row, err := s.q.GetServer(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return serverFromRow(row), nil
}

func (s *Store) SetServerCluster(ctx context.Context, serverID uuid.UUID, clusterID *uuid.UUID) error {
	return mapErr(s.q.SetServerCluster(ctx, gen.SetServerClusterParams{ID: serverID, ClusterID: nullUUID(clusterID)}))
}

func (s *Store) DeleteServer(ctx context.Context, id uuid.UUID) error {
	return mapErr(s.q.DeleteServer(ctx, id))
}

func (s *Store) GetRegistry(ctx context.Context) (*domain.Registry, error) {
	row, err := s.q.GetRegistrySettings(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	return registryFromRow(row), nil
}

func (s *Store) SetRegistryEnabled(ctx context.Context, enabled bool) (*domain.Registry, error) {
	row, err := s.q.UpsertRegistrySettings(ctx, enabled)
	if err != nil {
		return nil, mapErr(err)
	}
	return registryFromRow(row), nil
}

func (s *Store) CreateServerToken(ctx context.Context, tokenHash string) error {
	return mapErr(s.q.CreateServerToken(ctx, tokenHash))
}

func clusterFromRow(row gen.Cluster) *domain.Cluster {
	return &domain.Cluster{
		ID: row.ID, OrgID: row.OrgID, Name: row.Name, Labels: row.Labels, CreatedAt: row.CreatedAt,
	}
}

func serverFromRow(row gen.Server) *domain.Server {
	return &domain.Server{
		ID: row.ID, Name: row.Name, Host: row.Host, Role: row.Role, Status: row.Status,
		Version: row.Version, Labels: row.Labels, CPUCores: int(row.CpuCores),
		MemTotalBytes: row.MemTotalBytes, Load: float64(row.Load),
		LastHeartbeat: nullTimePtr(row.LastHeartbeat), ClusterID: uuidPtr(row.ClusterID),
		CreatedAt: row.CreatedAt,
	}
}

func registryFromRow(row gen.RegistrySetting) *domain.Registry {
	return &domain.Registry{
		Enabled: row.Enabled, Host: row.Host, Port: int(row.Port),
		ContainerID: row.ContainerID, Status: row.Status,
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

func nullTimePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
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
