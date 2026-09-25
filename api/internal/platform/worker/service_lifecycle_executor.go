package worker

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	databasesdomain "aether/internal/modules/databases/domain"
	servicesdomain "aether/internal/modules/services/domain"
)

type AppServiceLifecycle interface {
	StartService(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (string, error)
	StopService(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (string, error)
}

type ComposeServiceLifecycle interface {
	Start(context.Context, uuid.UUID, uuid.UUID) error
	Stop(context.Context, uuid.UUID, uuid.UUID) error
}

type DatabaseServiceLifecycle interface {
	Start(context.Context, uuid.UUID, uuid.UUID) (*databasesdomain.Database, error)
	Stop(context.Context, uuid.UUID, uuid.UUID) (*databasesdomain.Database, error)
}

func NewServiceLifecycleExecutor(apps AppServiceLifecycle, composes ComposeServiceLifecycle, databases DatabaseServiceLifecycle) func(context.Context, *servicesdomain.LifecycleOperation) error {
	return func(ctx context.Context, operation *servicesdomain.LifecycleOperation) error {
		if operation == nil {
			return fmt.Errorf("service lifecycle operation is required")
		}
		switch operation.Kind {
		case servicesdomain.KindApp:
			if operation.Action == servicesdomain.LifecycleStart {
				_, err := apps.StartService(ctx, operation.SpecID, operation.ServiceID, operation.OrgID)
				return err
			}
			if operation.Action == servicesdomain.LifecycleStop {
				_, err := apps.StopService(ctx, operation.SpecID, operation.ServiceID, operation.OrgID)
				return err
			}
		case servicesdomain.KindCompose:
			if operation.Action == servicesdomain.LifecycleStart {
				return composes.Start(ctx, operation.SpecID, operation.OrgID)
			}
			if operation.Action == servicesdomain.LifecycleStop {
				return composes.Stop(ctx, operation.SpecID, operation.OrgID)
			}
		case servicesdomain.KindDatabase:
			if operation.Action == servicesdomain.LifecycleStart {
				_, err := databases.Start(ctx, operation.SpecID, operation.OrgID)
				return err
			}
			if operation.Action == servicesdomain.LifecycleStop {
				_, err := databases.Stop(ctx, operation.SpecID, operation.OrgID)
				return err
			}
		}
		return fmt.Errorf("unsupported service lifecycle kind or action")
	}
}
