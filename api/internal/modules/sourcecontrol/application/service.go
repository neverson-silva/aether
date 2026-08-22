package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"aether/internal/modules/sourcecontrol/domain"
	"aether/internal/modules/sourcecontrol/ports"
)

func (s *Service) GetSource(ctx context.Context, serviceID, organizationID uuid.UUID) (*domain.ServiceSource, error) {
	return s.Sources.GetByService(ctx, serviceID, organizationID)
}

func (s *Service) SaveSource(ctx context.Context, source *domain.ServiceSource) (*domain.ServiceSource, error) {
	if source.Branch == "" {
		source.Branch = source.DefaultBranch
	}
	return s.Sources.Upsert(ctx, source)
}

func (s *Service) DeleteSource(ctx context.Context, serviceID, organizationID uuid.UUID) error {
	return s.Sources.DeleteByService(ctx, serviceID, organizationID)
}

type Service struct {
	Sources    ports.SourceStore
	Deliveries ports.DeliveryStore
	Files      ports.ChangedFilesResolver
	Deploy     ports.DeploymentTrigger
}

func (s *Service) HandlePush(ctx context.Context, event domain.PushEvent) error {
	return s.HandlePushWithFiles(ctx, event, s.Files)
}

func (s *Service) HandlePushWithFiles(ctx context.Context, event domain.PushEvent, files ports.ChangedFilesResolver) error {
	delivery := domain.WebhookDelivery{
		Provider: event.Provider, DeliveryID: event.DeliveryID, EventType: event.EventType,
		InstallationID: event.InstallationID, RepositoryID: event.Repository.ID,
	}
	claimed, accepted, err := s.Deliveries.Claim(ctx, delivery)
	if err != nil {
		return err
	}
	if !accepted {
		return nil
	}
	if err := s.processPush(ctx, event, files); err != nil {
		_ = s.Deliveries.Complete(ctx, claimed.ID, "failed", err.Error())
		return err
	}
	return s.Deliveries.Complete(ctx, claimed.ID, "processed", "")
}

func (s *Service) processPush(ctx context.Context, event domain.PushEvent, files ports.ChangedFilesResolver) error {
	sources, err := s.Sources.ListByRepository(ctx, event.Provider, event.InstallationID, event.Repository.ID)
	if err != nil {
		return err
	}
	changedFiles, filesKnown := []string(nil), false
	if files != nil && event.BeforeSHA != "" && event.AfterSHA != "" {
		changedFiles, err = files.GetChangedFiles(ctx, event.Repository.ID, event.BeforeSHA, event.AfterSHA)
		filesKnown = err == nil
	}
	for _, source := range sources {
		if source.Branch != "" && source.Branch != event.Branch {
			continue
		}
		decision := domain.EvaluateTrigger(domain.BuildTriggerRules{
			Branch: source.Branch, AutoDeploy: source.AutoDeploy,
			RootDirectory: source.RootDirectory, WatchPaths: source.WatchPaths,
			IgnorePaths: source.IgnorePaths, WatchRootFiles: source.WatchRootFiles,
		}, changedFiles, filesKnown)
		if !decision.Trigger {
			continue
		}
		if s.Deploy == nil {
			return fmt.Errorf("deployment trigger is not configured")
		}
		if _, err := s.Deploy.Deploy(ctx, source.ServiceID, source.OrganizationID, "webhook", event.AfterSHA); err != nil {
			return err
		}
	}
	return nil
}
