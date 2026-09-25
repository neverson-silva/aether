package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/services/domain"
)

type LifecycleRepository interface {
	RequestLifecycle(context.Context, domain.LifecycleRequest) (*domain.LifecycleOperation, error)
	ResolveServiceID(context.Context, uuid.UUID, uuid.UUID, domain.Kind) (uuid.UUID, error)
}

type Lifecycle struct {
	Repository LifecycleRepository
}

func (l *Lifecycle) RequestBySpec(ctx context.Context, specID, orgID uuid.UUID, kind domain.Kind, action domain.LifecycleAction) (*domain.LifecycleOperation, error) {
	if l == nil || l.Repository == nil {
		return nil, fmt.Errorf("service lifecycle is not configured")
	}
	serviceID, err := l.Repository.ResolveServiceID(ctx, specID, orgID, kind)
	if err != nil {
		return nil, err
	}
	return l.request(ctx, domain.LifecycleRequest{ServiceID: serviceID, SpecID: specID, OrgID: orgID, Kind: kind, Action: action})
}

func (l *Lifecycle) Request(ctx context.Context, serviceID, specID, orgID uuid.UUID, kind domain.Kind, action domain.LifecycleAction) (*domain.LifecycleOperation, error) {
	if l == nil || l.Repository == nil {
		return nil, fmt.Errorf("service lifecycle is not configured")
	}
	if action != domain.LifecycleStart && action != domain.LifecycleStop {
		return nil, fmt.Errorf("unsupported service lifecycle action %q", action)
	}
	switch kind {
	case domain.KindApp, domain.KindCompose, domain.KindDatabase:
	default:
		return nil, fmt.Errorf("unsupported service kind %q", kind)
	}
	return l.request(ctx, domain.LifecycleRequest{ServiceID: serviceID, SpecID: specID, OrgID: orgID, Kind: kind, Action: action})
}

func (l *Lifecycle) request(ctx context.Context, request domain.LifecycleRequest) (*domain.LifecycleOperation, error) {
	startedAt := time.Now()
	operation, err := l.Repository.RequestLifecycle(ctx, request)
	if err != nil {
		return nil, err
	}
	if operation.ID != uuid.Nil {
		slog.InfoContext(ctx, "service lifecycle command accepted", "service_id", operation.ServiceID, "org_id", operation.OrgID, "operation_id", operation.ID, "operation", operation.Action, "status", operation.Status, "acceptance_duration", time.Since(startedAt).String())
	}
	return operation, nil
}
