package infra

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"aether/internal/platform/worker"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) ListRuntimeServiceTargets(ctx context.Context) ([]worker.RuntimeServiceTarget, error) {
	rows, err := s.db.Query(ctx, `
SELECT s.id, s.org_id, s.kind, s.status,
       EXISTS (SELECT 1 FROM deployments d WHERE d.service_id = s.id) AS ever_deployed,
       EXISTS (SELECT 1 FROM deployments d WHERE d.service_id = s.id AND d.status IN ('queued', 'building', 'starting', 'health_checking')) AS active_deployment
FROM services s
WHERE s.deleted_at IS NULL
ORDER BY s.created_at, s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	targets := make([]worker.RuntimeServiceTarget, 0)
	for rows.Next() {
		var target worker.RuntimeServiceTarget
		if err := rows.Scan(&target.ID, &target.OrganizationID, &target.Kind, &target.Status, &target.EverDeployed, &target.ActiveDeployment); err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return targets, nil
}

func (s *Store) UpdateRuntimeStatus(ctx context.Context, serviceID uuid.UUID, status string) (bool, error) {
	result, err := s.db.Exec(ctx, `
UPDATE services
SET status = $2, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL AND status IS DISTINCT FROM $2`, serviceID, status)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

var _ worker.ServiceStateStore = (*Store)(nil)
