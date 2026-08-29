package application

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"aether/internal/modules/snapshots/domain"
)

type Snapshots struct {
	Store domain.Store
}

func (s *Snapshots) Create(ctx context.Context, orgID uuid.UUID, appID *uuid.UUID, volume, name string) (*domain.Snapshot, error) {
	volume = strings.TrimSpace(volume)
	name = strings.TrimSpace(name)
	if volume == "" || name == "" {
		return nil, domain.ErrValidation
	}
	return s.Store.CreateSnapshot(ctx, &domain.Snapshot{
		OrgID: orgID, AppID: appID, Volume: volume, Name: name,
	})
}

func (s *Snapshots) List(ctx context.Context, orgID uuid.UUID, limit int) ([]domain.Snapshot, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.Store.ListSnapshotsByOrg(ctx, orgID, limit)
}

func (s *Snapshots) CreateForService(ctx context.Context, serviceID, orgID uuid.UUID, volume, name string) (*domain.Snapshot, error) {
	volume = strings.TrimSpace(volume)
	name = strings.TrimSpace(name)
	if volume == "" || name == "" {
		return nil, domain.ErrValidation
	}
	return s.Store.CreateSnapshotForService(ctx, &domain.Snapshot{OrgID: orgID, ServiceID: &serviceID, Volume: volume, Name: name})
}

func (s *Snapshots) ListForService(ctx context.Context, orgID, serviceID uuid.UUID, limit int) ([]domain.Snapshot, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.Store.ListSnapshotsByService(ctx, orgID, serviceID, limit)
}

func (s *Snapshots) Restore(ctx context.Context, snapshotID, orgID uuid.UUID) error {
	snapshot, err := s.Store.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return err
	}
	if snapshot.OrgID != orgID {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Snapshots) Delete(ctx context.Context, snapshotID, orgID uuid.UUID) error {
	return s.Store.DeleteSnapshot(ctx, snapshotID, orgID)
}

func (s *Snapshots) CreateSchedule(ctx context.Context, orgID uuid.UUID, appID *uuid.UUID, volume, namePrefix, cronExpr string, retention int, enabled bool) (*domain.Schedule, error) {
	volume = strings.TrimSpace(volume)
	if volume == "" {
		return nil, domain.ErrValidation
	}
	if _, err := cron.ParseStandard(cronExpr); err != nil {
		return nil, domain.ErrValidation
	}
	if retention <= 0 {
		retention = 7
	}
	return s.Store.CreateSchedule(ctx, &domain.Schedule{
		OrgID: orgID, AppID: appID, Volume: volume, NamePrefix: strings.TrimSpace(namePrefix),
		Cron: cronExpr, Retention: retention, Enabled: enabled,
	})
}

func (s *Snapshots) ListSchedules(ctx context.Context, orgID uuid.UUID) ([]domain.Schedule, error) {
	return s.Store.ListSchedulesByOrg(ctx, orgID)
}

func (s *Snapshots) CreateScheduleForService(ctx context.Context, serviceID, orgID uuid.UUID, volume, namePrefix, cronExpr string, retention int, enabled bool) (*domain.Schedule, error) {
	volume = strings.TrimSpace(volume)
	if volume == "" {
		return nil, domain.ErrValidation
	}
	if _, err := cron.ParseStandard(cronExpr); err != nil {
		return nil, domain.ErrValidation
	}
	if retention <= 0 {
		retention = 7
	}
	return s.Store.CreateScheduleForService(ctx, &domain.Schedule{OrgID: orgID, ServiceID: &serviceID, Volume: volume, NamePrefix: strings.TrimSpace(namePrefix), Cron: cronExpr, Retention: retention, Enabled: enabled})
}

func (s *Snapshots) ListSchedulesForService(ctx context.Context, orgID, serviceID uuid.UUID) ([]domain.Schedule, error) {
	return s.Store.ListSchedulesByService(ctx, orgID, serviceID)
}

func (s *Snapshots) DeleteSchedule(ctx context.Context, scheduleID, orgID uuid.UUID) error {
	return s.Store.DeleteSchedule(ctx, scheduleID, orgID)
}
