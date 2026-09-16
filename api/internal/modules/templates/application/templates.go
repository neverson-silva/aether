package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	appsdomain "aether/internal/modules/apps/domain"
	"aether/internal/modules/templates/domain"
	variablesDomain "aether/internal/modules/variables/domain"
	composeengine "aether/internal/platform/compose"
)

type Templates struct {
	Store                    domain.Store
	Apps                     AppStore
	Catalog                  RemoteCatalog
	Variables                variablesDomain.Store
	ProvisionTemplateDomains func(context.Context, uuid.UUID, *domain.ComposeApp, []domain.TemplateDomain) error
}

var errCatalogUnavailable = errors.New("template catalog unavailable")

type RemoteCatalog interface {
	List(ctx context.Context) ([]domain.Template, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Template, error)
}

type RemoteCatalogLogo interface {
	Logo(ctx context.Context, id uuid.UUID) ([]byte, string, error)
}

func (t *Templates) Logo(ctx context.Context, id uuid.UUID) ([]byte, string, error) {
	provider, ok := t.Catalog.(RemoteCatalogLogo)
	if !ok {
		return nil, "", domain.ErrNotFound
	}
	return provider.Logo(ctx, id)
}

func (t *Templates) List(ctx context.Context, filter domain.Filter) ([]domain.Template, error) {
	if t.Catalog == nil {
		return nil, errCatalogUnavailable
	}
	remote, err := t.Catalog.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Template, 0, len(remote))
	for _, template := range remote {
		if filter.Category != "" && !strings.EqualFold(filter.Category, template.Category) {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(template.Name+" "+template.Description), strings.ToLower(filter.Search)) {
			continue
		}
		out = append(out, template)
	}
	return out, nil
}

func (t *Templates) Install(ctx context.Context, templateID, orgID, projectID uuid.UUID, name string, overrides map[string]string) (*domain.Template, error) {
	if t.Catalog == nil {
		return nil, errCatalogUnavailable
	}
	tpl, err := t.Catalog.Get(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if _, err := t.Apps.GetProject(ctx, projectID, orgID); err != nil {
		return nil, err
	}
	var environmentID *uuid.UUID
	if defaults, ok := t.Apps.(interface {
		DefaultEnvironment(context.Context, uuid.UUID) (uuid.UUID, error)
	}); ok {
		defaultID, err := defaults.DefaultEnvironment(ctx, projectID)
		if err != nil && !errors.Is(err, appsdomain.ErrNotFound) {
			return nil, fmt.Errorf("resolve default environment: %w", err)
		}
		if err == nil {
			environmentID = &defaultID
		}
	}
	appName := strings.TrimSpace(name)
	if appName == "" {
		appName = tpl.Name
	}
	if err := validateName(appName); err != nil {
		return nil, domain.ErrValidation
	}
	compose := strings.TrimSpace(tpl.ComposeYAML)
	if compose == "" {
		compose, err = composeYAML(tpl.Definition, overrides)
		if err != nil {
			return nil, domain.ErrValidation
		}
	}
	compose, err = composeengine.NormalizeNamedResourceDefinitions(compose)
	if err != nil {
		return nil, domain.ErrValidation
	}
	port, hasPort, err := composePublishedPort(compose)
	if err != nil {
		return nil, domain.ErrValidation
	}
	if hasPort && port < 1024 {
		port, err = t.Store.NextComposePort(ctx)
		if err != nil {
			return nil, fmt.Errorf("allocate compose port: %w", err)
		}
		compose, err = replaceComposePublishedPort(compose, port)
		if err != nil {
			return nil, domain.ErrValidation
		}
	}
	if !hasPort {
		port, err = composeContainerPort(compose)
		if err != nil {
			return nil, domain.ErrValidation
		}
		if port == 0 {
			port, err = t.Store.NextComposePort(ctx)
			if err != nil {
				return nil, fmt.Errorf("allocate compose port: %w", err)
			}
			compose, err = addComposePort(compose, port)
			if err != nil {
				return nil, domain.ErrValidation
			}
		}
	}
	compose, err = injectComposeSecurityDefaults(compose)
	if err != nil {
		return nil, domain.ErrValidation
	}
	if err := composeengine.ValidatePolicy(compose); err != nil {
		return nil, domain.ErrValidation
	}
	if err := t.ensureTemplateVariables(ctx, projectID, compose, overrides); err != nil {
		return nil, err
	}
	created, err := t.Store.CreateComposeApp(ctx, &domain.ComposeApp{
		OrgID: orgID, ProjectID: projectID, EnvironmentID: environmentID, Name: appName, Compose: compose, Port: port, Status: "stopped",
	})
	if err != nil {
		return nil, err
	}
	if err := t.ensureServiceVariables(ctx, created, compose, overrides); err != nil {
		return nil, err
	}
	if t.ProvisionTemplateDomains != nil && len(tpl.Domains) > 0 {
		if err := t.ProvisionTemplateDomains(ctx, orgID, created, tpl.Domains); err != nil {
			return nil, fmt.Errorf("provision template domains: %w", err)
		}
	}
	tpl.Installs++
	tpl.ComposeYAML = compose
	return tpl, nil
}

func (t *Templates) ensureServiceVariables(ctx context.Context, app *domain.ComposeApp, compose string, overrides map[string]string) error {
	writer, ok := t.Apps.(ServiceVariableStore)
	if !ok || app == nil || app.ServiceID == uuid.Nil {
		return nil
	}
	existing, err := writer.ListServiceEnvVars(ctx, app.ServiceID)
	if err != nil {
		return fmt.Errorf("list service variables: %w", err)
	}
	known := make(map[string]struct{}, len(existing))
	for _, variable := range existing {
		known[variable.Name] = struct{}{}
	}
	projectValues, err := t.resolvedProjectValues(ctx, app.ProjectID)
	if err != nil {
		return err
	}
	for _, variable := range templateEnvironmentVariables(compose) {
		if _, exists := known[variable.Name]; exists {
			continue
		}
		value := strings.TrimSpace(overrides[variable.Name])
		if value == "" {
			value = strings.TrimSpace(projectValues[variable.Name])
		}
		if value == "" {
			value = strings.TrimSpace(variable.Value)
		}
		if value == "" && isGeneratedTemplateSecret(variable.Name) {
			value, err = generatedTemplateSecret()
			if err != nil {
				return fmt.Errorf("generate service secret: %w", err)
			}
		}
		if value == "" {
			continue
		}
		if err := writer.UpsertServiceEnvVar(ctx, app.ServiceID, variable.Name, value, isGeneratedTemplateSecret(variable.Name)); err != nil {
			return fmt.Errorf("save service variable %s: %w", variable.Name, err)
		}
	}
	return nil
}

func (t *Templates) resolvedProjectValues(ctx context.Context, projectID uuid.UUID) (map[string]string, error) {
	if t.Variables == nil {
		return map[string]string{}, nil
	}
	var variables []variablesDomain.Variable
	if reader, ok := t.Variables.(ResolvedProjectVariableStore); ok {
		resolved, err := reader.ListResolvedVariables(ctx, projectID, uuid.Nil)
		if err != nil {
			return nil, fmt.Errorf("list resolved project variables: %w", err)
		}
		variables = resolved
	} else {
		listed, err := t.Variables.ListVariables(ctx, projectID, uuid.Nil)
		if err != nil {
			return nil, fmt.Errorf("list project variables: %w", err)
		}
		variables = listed
	}
	values := make(map[string]string, len(variables))
	for _, variable := range variables {
		if variable.Value != "" {
			values[variable.Key] = variable.Value
		}
	}
	return values, nil
}

func (t *Templates) ensureTemplateVariables(ctx context.Context, projectID uuid.UUID, compose string, overrides map[string]string) error {
	if t.Variables == nil {
		return nil
	}
	existing, err := t.Variables.ListVariables(ctx, projectID, uuid.Nil)
	if err != nil {
		return fmt.Errorf("list template variables: %w", err)
	}
	current := make(map[string]variablesDomain.Variable, len(existing))
	for _, variable := range existing {
		current[variable.Key] = variable
	}
	for _, variable := range templateEnvironmentVariables(compose) {
		value := strings.TrimSpace(overrides[variable.Name])
		if value == "" {
			value = variable.Value
		}
		if value == "" && isGeneratedTemplateSecret(variable.Name) {
			value, err = generatedTemplateSecret()
			if err != nil {
				return fmt.Errorf("generate template secret: %w", err)
			}
		}
		if value == "" {
			continue
		}
		if previous, ok := current[variable.Name]; ok && strings.TrimSpace(previous.Value) != "" {
			continue
		}
		if _, err := t.Variables.UpsertVariable(ctx, &variablesDomain.Variable{ProjectID: projectID, Key: variable.Name, Value: value, IsSecret: isGeneratedTemplateSecret(variable.Name)}); err != nil {
			return fmt.Errorf("save template variable %s: %w", variable.Name, err)
		}
	}
	return nil
}

type templateEnvironmentVariable struct {
	Name  string
	Value string
}

func templateEnvironmentVariables(content string) []templateEnvironmentVariable {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return nil
	}
	root := &document
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	if root.Kind != yaml.MappingNode {
		return nil
	}
	services := yamlMappingValue(root, "services")
	if services == nil || services.Kind != yaml.MappingNode {
		return nil
	}
	seen := map[string]struct{}{}
	variables := make([]templateEnvironmentVariable, 0)
	for i := 0; i+1 < len(services.Content); i += 2 {
		environment := yamlMappingValue(services.Content[i+1], "environment")
		if environment == nil {
			continue
		}
		switch environment.Kind {
		case yaml.MappingNode:
			for j := 0; j+1 < len(environment.Content); j += 2 {
				name := strings.TrimSpace(environment.Content[j].Value)
				value := environment.Content[j+1].Value
				if strings.Contains(value, "${") {
					value = ""
				}
				appendTemplateEnvironmentVariable(&variables, seen, name, value)
			}
		case yaml.SequenceNode:
			for _, item := range environment.Content {
				name, value, found := strings.Cut(item.Value, "=")
				if !found {
					name, value = item.Value, ""
				}
				if strings.Contains(value, "${") {
					value = ""
				}
				appendTemplateEnvironmentVariable(&variables, seen, strings.TrimSpace(name), value)
			}
		}
	}
	return variables
}

func appendTemplateEnvironmentVariable(variables *[]templateEnvironmentVariable, seen map[string]struct{}, name, value string) {
	if name == "" {
		return
	}
	if _, exists := seen[name]; exists {
		return
	}
	seen[name] = struct{}{}
	*variables = append(*variables, templateEnvironmentVariable{Name: name, Value: value})
}

func yamlMappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

func isGeneratedTemplateSecret(name string) bool {
	lower := strings.ToLower(name)
	for _, part := range []string{"password", "passwd", "secret", "token", "api_key", "access_key", "private_key", "credential"} {
		if strings.Contains(lower, part) {
			return true
		}
	}
	return false
}

func generatedTemplateSecret() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func replaceComposePublishedPort(content string, port int) (string, error) {
	var document map[string]any
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", err
	}
	services, ok := document["services"].(map[string]any)
	if !ok || len(services) == 0 {
		return "", fmt.Errorf("compose has no services mapping")
	}
	for _, raw := range services {
		service, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ports, ok := service["ports"].([]any)
		if !ok || len(ports) == 0 {
			continue
		}
		for index, rawPort := range ports {
			value, ok := rawPort.(string)
			if !ok {
				continue
			}
			parts := strings.Split(value, ":")
			if len(parts) < 2 {
				continue
			}
			parts[len(parts)-2] = fmt.Sprintf("%d", port)
			ports[index] = strings.Join(parts, ":")
			encoded, err := yaml.Marshal(document)
			return string(encoded), err
		}
	}
	return "", fmt.Errorf("compose has no published port")
}

func (t *Templates) ListCompose(ctx context.Context, orgID uuid.UUID) ([]domain.ComposeApp, error) {
	return t.Store.ListComposeAppsByOrg(ctx, orgID)
}

func (t *Templates) DeleteCompose(ctx context.Context, id, orgID uuid.UUID) error {
	return t.Store.DeleteComposeApp(ctx, id, orgID)
}

type serviceDef struct {
	Name     string            `json:"name"`
	Image    string            `json:"image"`
	Port     int               `json:"port"`
	Env      map[string]string `json:"env"`
	Volumes  []string          `json:"volumes"`
	Versions []string          `json:"versions"`
}

type templateDef struct {
	Services []serviceDef `json:"services"`
}

func composeYAML(definition string, overrides map[string]string) (string, error) {
	var def templateDef
	if err := json.Unmarshal([]byte(definition), &def); err != nil {
		return "", err
	}
	if len(def.Services) == 0 {
		return "", fmt.Errorf("template with no services")
	}
	compose := map[string]any{
		"version":  "3.8",
		"services": composeServices(def.Services, overrides),
	}
	raw, err := yaml.Marshal(compose)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func composeServices(services []serviceDef, overrides map[string]string) map[string]any {
	out := make(map[string]any, len(services))
	for _, svc := range services {
		name := svc.Name
		if name == "" {
			name = "app"
		}
		image := svc.Image
		if image == "" && len(svc.Versions) > 0 {
			image = svc.Versions[0]
		}
		if override, ok := overrides["image"]; ok && override != "" {
			image = override
		}
		entry := map[string]any{
			"image":      image,
			"restart":    "no",
			"mem_limit":  "512m",
			"cpus":       "1.0",
			"pids_limit": 256,
		}
		if svc.Port > 0 {
			entry["ports"] = []string{fmt.Sprintf("%d:%d", svc.Port, svc.Port)}
		}
		if len(svc.Env) > 0 {
			environment := make(map[string]string, len(svc.Env))
			for key, value := range svc.Env {
				if override, ok := overrides[key]; ok {
					value = override
				}
				environment[key] = value
			}
			entry["environment"] = environment
		}
		if len(svc.Volumes) > 0 {
			entry["volumes"] = svc.Volumes
		}
		out[name] = entry
	}
	return out
}

func validateName(name string) error {
	if len(name) < 1 || len(name) > 64 {
		return domain.ErrValidation
	}
	return nil
}
