package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"aether/internal/modules/services/domain"
	"aether/internal/platform/druntime/events"
	"aether/internal/platform/druntime/queue"
)

type LifecycleStore struct {
	pool *pgxpool.Pool
}

func NewLifecycleStore(pool *pgxpool.Pool) *LifecycleStore {
	return &LifecycleStore{pool: pool}
}

func (s *LifecycleStore) ResolveServiceID(ctx context.Context, specID, orgID uuid.UUID, kind domain.Kind) (uuid.UUID, error) {
	var query string
	switch kind {
	case domain.KindApp:
		query = `SELECT s.id FROM services s JOIN apps a ON a.service_id = s.id WHERE a.id = $1 AND s.org_id = $2 AND s.deleted_at IS NULL`
	case domain.KindCompose:
		query = `SELECT s.id FROM services s JOIN compose_apps c ON c.service_id = s.id WHERE c.id = $1 AND s.org_id = $2 AND s.deleted_at IS NULL`
	case domain.KindDatabase:
		query = `SELECT s.id FROM services s JOIN databases d ON d.service_id = s.id WHERE d.id = $1 AND s.org_id = $2 AND s.deleted_at IS NULL`
	default:
		return uuid.Nil, domain.ErrLifecycleConflict
	}
	var serviceID uuid.UUID
	if err := s.pool.QueryRow(ctx, query, specID, orgID).Scan(&serviceID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, domain.ErrNotFound
		}
		return uuid.Nil, err
	}
	return serviceID, nil
}

func (s *LifecycleStore) RequestLifecycle(ctx context.Context, request domain.LifecycleRequest) (*domain.LifecycleOperation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var kind, status string
	if err := tx.QueryRow(ctx, `SELECT kind, status FROM services WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL FOR UPDATE`, request.ServiceID, request.OrgID).Scan(&kind, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if domain.Kind(kind) != request.Kind {
		return nil, domain.ErrLifecycleConflict
	}

	var nextStatus domain.Status
	var activeStatus string
	var settledStatus domain.Status
	if request.Action == domain.LifecycleStart {
		nextStatus = domain.StatusStarting
		activeStatus = string(domain.StatusStarting)
		settledStatus = domain.StatusRunning
	} else {
		nextStatus = domain.StatusStopping
		activeStatus = string(domain.StatusStopping)
		settledStatus = domain.StatusStopped
	}
	var operation domain.LifecycleOperation
	activeErr := tx.QueryRow(ctx, `SELECT id, service_id, org_id, spec_id, kind, action, status, attempts, lease_until FROM service_lifecycle_operations WHERE service_id = $1 AND status IN ('accepted', 'running') FOR UPDATE`, request.ServiceID).Scan(&operation.ID, &operation.ServiceID, &operation.OrgID, &operation.SpecID, &operation.Kind, &operation.Action, &operation.Status, &operation.Attempts, &operation.LeaseUntil)
	if activeErr == nil {
		if operation.Action != request.Action {
			return nil, domain.ErrLifecycleConflict
		}
		if status != activeStatus {
			if _, err := tx.Exec(ctx, `UPDATE services SET status = $1, updated_at = now() WHERE id = $2 AND org_id = $3 AND deleted_at IS NULL`, nextStatus, request.ServiceID, request.OrgID); err != nil {
				return nil, err
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return &operation, nil
	}
	if !errors.Is(activeErr, pgx.ErrNoRows) {
		return nil, activeErr
	}
	if status == string(settledStatus) {
		return &domain.LifecycleOperation{ServiceID: request.ServiceID, OrgID: request.OrgID, SpecID: request.SpecID, Kind: request.Kind, Action: request.Action, Status: string(settledStatus)}, nil
	}
	if status == string(domain.StatusDeploying) {
		var deploymentActive bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM deployments WHERE service_id = $1 AND status IN ('queued', 'building', 'starting', 'health_checking'))`, request.ServiceID).Scan(&deploymentActive); err != nil {
			return nil, err
		}
		if deploymentActive {
			return nil, domain.ErrLifecycleConflict
		}
	}
	if status == string(domain.StatusStarting) || status == string(domain.StatusStopping) {
		return nil, domain.ErrLifecycleConflict
	}

	operationID := uuid.New()
	if _, err := tx.Exec(ctx, `INSERT INTO service_lifecycle_operations (id, service_id, org_id, spec_id, kind, action, status, lease_until) VALUES ($1, $2, $3, $4, $5, $6, 'accepted', now() + interval '45 seconds')`, operationID, request.ServiceID, request.OrgID, request.SpecID, request.Kind, request.Action); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE services SET status = $1, updated_at = now() WHERE id = $2 AND org_id = $3 AND deleted_at IS NULL`, nextStatus, request.ServiceID, request.OrgID); err != nil {
		return nil, err
	}

	jobPayload, err := json.Marshal(map[string]string{"operation_id": operationID.String()})
	if err != nil {
		return nil, err
	}
	job := queue.Job{ID: operationID.String(), Type: "service.lifecycle.execute", OrgID: request.OrgID.String(), Payload: jobPayload}
	eventPayload, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}
	event := events.Event{ID: operationID.String(), Type: "service.lifecycle.queued", AggregateType: "service", AggregateID: request.ServiceID.String(), Payload: eventPayload, TS: time.Now().UTC()}
	serializedEvent, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO outbox_events (id, topic, event_type, aggregate_type, aggregate_id, payload) VALUES ($1, 'service-operations', $2, $3, $4, $5)`, operationID, event.Type, event.AggregateType, event.AggregateID, serializedEvent); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &domain.LifecycleOperation{ID: operationID, ServiceID: request.ServiceID, OrgID: request.OrgID, SpecID: request.SpecID, Kind: request.Kind, Action: request.Action, Status: "accepted"}, nil
}

func (s *LifecycleStore) GetLifecycleOperation(ctx context.Context, operationID uuid.UUID) (*domain.LifecycleOperation, error) {
	var operation domain.LifecycleOperation
	err := s.pool.QueryRow(ctx, `SELECT id, service_id, org_id, spec_id, kind, action, status, attempts, lease_until FROM service_lifecycle_operations WHERE id = $1`, operationID).Scan(&operation.ID, &operation.ServiceID, &operation.OrgID, &operation.SpecID, &operation.Kind, &operation.Action, &operation.Status, &operation.Attempts, &operation.LeaseUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &operation, nil
}

func (s *LifecycleStore) ClaimLifecycleOperation(ctx context.Context, operationID uuid.UUID) (*domain.LifecycleOperation, bool, error) {
	var operation domain.LifecycleOperation
	err := s.pool.QueryRow(ctx, `UPDATE service_lifecycle_operations SET status = 'running', attempts = attempts + 1, lease_until = now() + interval '45 seconds', updated_at = now() WHERE id = $1 AND (status = 'accepted' OR (status = 'running' AND lease_until < now())) RETURNING id, service_id, org_id, spec_id, kind, action, status, attempts, lease_until`, operationID).Scan(&operation.ID, &operation.ServiceID, &operation.OrgID, &operation.SpecID, &operation.Kind, &operation.Action, &operation.Status, &operation.Attempts, &operation.LeaseUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &operation, true, nil
}

func (s *LifecycleStore) RenewLifecycleOperation(ctx context.Context, operationID uuid.UUID, attempt int) error {
	result, err := s.pool.Exec(ctx, `UPDATE service_lifecycle_operations SET lease_until = now() + interval '45 seconds', updated_at = now() WHERE id = $1 AND status = 'running' AND attempts = $2`, operationID, attempt)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("service lifecycle operation lease was lost")
	}
	return nil
}

func (s *LifecycleStore) CompleteLifecycleOperation(ctx context.Context, operationID uuid.UUID, attempt int, finalStatus, operationError string) (*domain.LifecycleOperation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var operation domain.LifecycleOperation
	if err := tx.QueryRow(ctx, `SELECT id, service_id, org_id, spec_id, kind, action, status, attempts FROM service_lifecycle_operations WHERE id = $1 FOR UPDATE`, operationID).Scan(&operation.ID, &operation.ServiceID, &operation.OrgID, &operation.SpecID, &operation.Kind, &operation.Action, &operation.Status, &operation.Attempts); err != nil {
		return nil, err
	}
	if operation.Status == "succeeded" || operation.Status == "failed" {
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return &operation, nil
	}
	if operation.Status != "running" || operation.Attempts != attempt {
		return nil, fmt.Errorf("service lifecycle operation lease was lost")
	}
	if operationError == "" {
		operation.Status = "succeeded"
	} else {
		operation.Status = "failed"
	}
	if _, err := tx.Exec(ctx, `UPDATE service_lifecycle_operations SET status = $2, error = $3, lease_until = NULL, updated_at = now(), completed_at = now() WHERE id = $1 AND attempts = $4`, operationID, operation.Status, operationError, attempt); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE services SET status = $1, updated_at = now() WHERE id = $2 AND org_id = $3 AND deleted_at IS NULL`, finalStatus, operation.ServiceID, operation.OrgID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &operation, nil
}

func (s *LifecycleStore) RecoverLifecycleOperations(ctx context.Context, limit int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id, service_id, org_id FROM service_lifecycle_operations WHERE status IN ('accepted', 'running') AND (lease_until IS NULL OR lease_until < now()) ORDER BY created_at, id FOR UPDATE SKIP LOCKED LIMIT $1`, limit)
	if err != nil {
		return err
	}
	type staleOperation struct {
		id        uuid.UUID
		serviceID uuid.UUID
		orgID     uuid.UUID
	}
	operations := make([]staleOperation, 0, limit)
	for rows.Next() {
		var operation staleOperation
		if err := rows.Scan(&operation.id, &operation.serviceID, &operation.orgID); err != nil {
			rows.Close()
			return err
		}
		operations = append(operations, operation)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, operation := range operations {
		if _, err := tx.Exec(ctx, `UPDATE service_lifecycle_operations SET status = 'accepted', lease_until = now() + interval '45 seconds', updated_at = now() WHERE id = $1`, operation.id); err != nil {
			return err
		}
		dispatchID := uuid.New()
		jobPayload, err := json.Marshal(map[string]string{"operation_id": operation.id.String()})
		if err != nil {
			return err
		}
		job, err := json.Marshal(queue.Job{ID: dispatchID.String(), Type: "service.lifecycle.execute", OrgID: operation.orgID.String(), Payload: jobPayload})
		if err != nil {
			return err
		}
		event := events.Event{ID: dispatchID.String(), Type: "service.lifecycle.queued", AggregateType: "service", AggregateID: operation.serviceID.String(), Payload: job, TS: time.Now().UTC()}
		payload, err := json.Marshal(event)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO outbox_events (id, topic, event_type, aggregate_type, aggregate_id, payload) VALUES ($1, 'service-operations', $2, $3, $4, $5) ON CONFLICT (id) DO NOTHING`, dispatchID, event.Type, event.AggregateType, event.AggregateID, payload); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
