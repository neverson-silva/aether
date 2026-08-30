package application

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"aether/internal/modules/sourcecontrol/domain"
	"aether/internal/modules/sourcecontrol/ports"
)

func (s *Service) GetSource(ctx context.Context, serviceID, organizationID uuid.UUID) (*domain.ServiceSource, error) {
	return s.Sources.GetByService(ctx, serviceID, organizationID)
}

func (s *Service) SaveSource(ctx context.Context, source *domain.ServiceSource) (*domain.ServiceSource, error) {
	root, err := safeRepositoryPath(source.RootDirectory)
	if err != nil {
		return nil, err
	}
	if root == "." {
		source.RootDirectory = ""
	} else {
		source.RootDirectory = root
	}
	source.WatchPaths = normalizeWatchPaths(source.WatchPaths)
	source.IgnorePaths = normalizeWatchPaths(source.IgnorePaths)
	if source.Branch == "" {
		source.Branch = source.DefaultBranch
	}
	if source.EnvironmentTemplatePath == "" {
		source.EnvironmentTemplatePath = defaultEnvironmentTemplatePath
	}
	if _, err := serviceTemplatePath(source.RootDirectory, source.EnvironmentTemplatePath); err != nil {
		return nil, err
	}
	saved, err := s.Sources.Upsert(ctx, source)
	if err != nil {
		return nil, err
	}
	_, _, _ = s.ImportTemplate(ctx, saved)
	return saved, nil
}

func normalizeWatchPaths(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	for _, item := range paths {
		item = strings.TrimSpace(strings.ReplaceAll(item, "\\", "/"))
		if item == "" {
			continue
		}
		normalized = append(normalized, strings.TrimPrefix(path.Clean(item), "./"))
	}
	return normalized
}

func (s *Service) DeleteSource(ctx context.Context, serviceID, organizationID uuid.UUID) error {
	return s.Sources.DeleteByService(ctx, serviceID, organizationID)
}

type Service struct {
	Sources       ports.SourceStore
	Deliveries    ports.DeliveryStore
	Files         ports.ChangedFilesResolver
	Deploy        ports.DeploymentTrigger
	ServiceDeploy interface {
		DeployService(context.Context, uuid.UUID, uuid.UUID, string, string) (any, error)
	}
	Templates ports.TemplateFileReader
	Apps      ServiceVariableImporter
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
		if s.ServiceDeploy == nil && s.Deploy == nil {
			return fmt.Errorf("deployment trigger is not configured")
		}
		var deployErr error
		if s.ServiceDeploy != nil {
			_, deployErr = s.ServiceDeploy.DeployService(ctx, source.ServiceID, source.OrganizationID, "webhook", event.AfterSHA)
		} else {
			_, deployErr = s.Deploy.Deploy(ctx, source.ServiceID, source.OrganizationID, "webhook", event.AfterSHA)
		}
		if deployErr != nil {
			return deployErr
		}
	}
	return nil
}

const defaultEnvironmentTemplatePath = ".env.example"

var environmentKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type ServiceVariableImporter interface {
	ImportMissingEnvVars(ctx context.Context, appID, organizationID uuid.UUID, names []string) (int, error)
}

func (s *Service) ImportTemplate(ctx context.Context, source *domain.ServiceSource) (int, bool, error) {
	if s.Templates == nil || s.Apps == nil {
		return 0, false, nil
	}
	templatePath, err := serviceTemplatePath(source.RootDirectory, source.EnvironmentTemplatePath)
	if err != nil {
		return 0, false, err
	}
	ref := source.Branch
	if ref == "" {
		ref = source.DefaultBranch
	}
	content, err := s.Templates.GetServiceFile(ctx, source.ConnectionID, source.RepositoryID, templatePath, ref)
	if err != nil {
		return 0, false, nil
	}
	count, err := s.Apps.ImportMissingEnvVars(ctx, source.ServiceID, source.OrganizationID, ParseEnvironmentKeys(content))
	return count, true, err
}

func serviceTemplatePath(rootDirectory, templatePath string) (string, error) {
	root, err := safeRepositoryPath(rootDirectory)
	if err != nil {
		return "", err
	}
	template, err := safeRepositoryPath(templatePath)
	if err != nil {
		return "", err
	}
	if root == "." {
		return template, nil
	}
	return path.Join(root, template), nil
}

func safeRepositoryPath(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return ".", nil
	}
	clean := path.Clean(strings.TrimPrefix(value, "./"))
	if clean == "." || clean == "" {
		return ".", nil
	}
	if path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("repository path escapes checkout")
	}
	return clean, nil
}

func ParseEnvironmentKeys(content string) []string {
	keys := make([]string, 0)
	seen := make(map[string]struct{})
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(strings.TrimPrefix(raw, "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		name, _, ok := strings.Cut(line, "=")
		name = strings.TrimSpace(name)
		if !ok || !environmentKeyPattern.MatchString(name) {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		keys = append(keys, name)
	}
	return keys
}

func parseEnvironmentKeys(content string) []string {
	return ParseEnvironmentKeys(content)
}
