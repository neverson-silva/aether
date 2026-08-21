package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"aether/internal/backups/domain"
)

func (s *DatabaseBackups) GetConfiguration(ctx context.Context, dbID, orgID uuid.UUID) (*domain.BackupConfiguration, error) {
	if _, err := s.Databases.Get(ctx, dbID, orgID); err != nil {
		return nil, err
	}
	cfg, err := s.Store.GetConfiguration(ctx, dbID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *DatabaseBackups) UpsertConfiguration(ctx context.Context, orgID uuid.UUID, cfg *domain.BackupConfiguration) (*domain.BackupConfiguration, error) {
	if cfg.DatabaseID == uuid.Nil {
		return nil, domain.ErrValidation
	}
	if _, err := s.Databases.Get(ctx, cfg.DatabaseID, orgID); err != nil {
		return nil, err
	}
	if err := ValidateSchedule(cfg.Schedule); err != nil {
		return nil, err
	}
	if cfg.Retention.Type != domain.RetentionAll && cfg.Retention.Type != domain.RetentionLatest {
		return nil, domain.ErrValidation
	}
	if cfg.Schedule.Timezone == "" {
		cfg.Schedule.Timezone = "UTC"
	}
	if _, err := s.Destinations.GetProvider(ctx, cfg.DestinationID, orgID); err != nil {
		return nil, err
	}
	next, err := NextRun(cfg.Schedule, time.Now())
	if err != nil {
		return nil, err
	}
	cfg.NextRunAt = &next
	saved, err := s.Store.UpsertConfiguration(ctx, cfg)
	if err != nil {
		return nil, err
	}
	s.Audit.Record(ctx, orgID, "backup.configuration.update", "database", cfg.DatabaseID.String(), string(cfg.Schedule.Type)+" "+cfg.Schedule.Timezone)
	return saved, nil
}

func (s *DatabaseBackups) DeleteConfiguration(ctx context.Context, dbID, orgID uuid.UUID) error {
	if _, err := s.Databases.Get(ctx, dbID, orgID); err != nil {
		return err
	}
	if err := s.Store.DeleteConfiguration(ctx, dbID); err != nil {
		return err
	}
	s.Audit.Record(ctx, orgID, "backup.configuration.delete", "database", dbID.String(), "")
	return nil
}