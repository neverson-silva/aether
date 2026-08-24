package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/backups/domain"
)

func (s *DatabaseBackups) ListConfigurations(ctx context.Context, dbID, orgID uuid.UUID) ([]domain.BackupConfiguration, error) {
	if _, err := s.Databases.Get(ctx, dbID, orgID); err != nil {
		return nil, err
	}
	return s.Store.ListConfigurationsByDatabase(ctx, dbID)
}

func (s *DatabaseBackups) GetConfiguration(ctx context.Context, dbID, configID, orgID uuid.UUID) (*domain.BackupConfiguration, error) {
	if _, err := s.Databases.Get(ctx, dbID, orgID); err != nil {
		return nil, err
	}
	cfg, err := s.Store.GetConfiguration(ctx, configID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if cfg == nil || cfg.DatabaseID != dbID {
		return nil, domain.ErrNotFound
	}
	return cfg, nil
}

func (s *DatabaseBackups) SaveConfiguration(ctx context.Context, orgID uuid.UUID, cfg *domain.BackupConfiguration, create bool) (*domain.BackupConfiguration, error) {
	if cfg.DatabaseID == uuid.Nil {
		return nil, domain.ErrValidation
	}
	if _, err := s.Databases.Get(ctx, cfg.DatabaseID, orgID); err != nil {
		return nil, err
	}
	if !create {
		current, err := s.GetConfiguration(ctx, cfg.DatabaseID, cfg.ID, orgID)
		if err != nil {
			return nil, err
		}
		cfg.CreatedAt = current.CreatedAt
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
		return nil, domain.ErrValidation
	}
	next, err := NextRun(cfg.Schedule, time.Now())
	if err != nil {
		return nil, err
	}
	cfg.NextRunAt = &next
	var saved *domain.BackupConfiguration
	if create {
		saved, err = s.Store.CreateConfiguration(ctx, cfg)
	} else {
		saved, err = s.Store.UpdateConfiguration(ctx, cfg)
	}
	if err != nil {
		return nil, err
	}
	if s.Scheduler != nil {
		if err := s.scheduleConfiguration(ctx, *saved); err != nil {
			return nil, err
		}
	}
	s.Audit.Record(ctx, orgID, "backup.configuration.update", "database", cfg.DatabaseID.String(), string(cfg.Schedule.Type)+" "+cfg.Schedule.Timezone)
	return saved, nil
}

func (s *DatabaseBackups) UpsertConfiguration(ctx context.Context, orgID uuid.UUID, cfg *domain.BackupConfiguration) (*domain.BackupConfiguration, error) {
	return s.SaveConfiguration(ctx, orgID, cfg, cfg.ID == uuid.Nil)
}

func (s *DatabaseBackups) DeleteConfiguration(ctx context.Context, dbID, configID, orgID uuid.UUID) error {
	if _, err := s.GetConfiguration(ctx, dbID, configID, orgID); err != nil {
		return err
	}
	if err := s.Store.DeleteConfiguration(ctx, configID); err != nil {
		return err
	}
	s.Audit.Record(ctx, orgID, "backup.configuration.delete", "database", dbID.String(), "")
	return nil
}
