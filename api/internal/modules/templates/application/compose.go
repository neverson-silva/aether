package application

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	appsdomain "aether/internal/modules/apps/domain"
	realtimedomain "aether/internal/modules/realtime/domain"
	sourcedomain "aether/internal/modules/sourcecontrol/domain"
	"aether/internal/modules/templates/domain"
	variablesDomain "aether/internal/modules/variables/domain"
	composeengine "aether/internal/platform/compose"
	"aether/internal/platform/worker"
)

type Compose struct {
	Store           domain.Store
	Apps            AppStore
	Deployments     DeploymentStore
	ServiceIdentity func(context.Context, uuid.UUID) (uuid.UUID, error)
	DataDir         string
	Runtime         worker.Runtime
	ComposeRuntime  composeengine.Executor
	ProjectVars     ProjectVarStore
	Variables       EffectiveVariableResolver
	Events          EventLog
	Source          ComposeSource
	Clone           ComposeClone
}

type ComposeSource interface {
	GetByService(context.Context, uuid.UUID, uuid.UUID) (*sourcedomain.ServiceSource, error)
}

type ComposeClone interface {
	Clone(context.Context, *sourcedomain.ServiceSource, string) (string, error)
}

func (c *Compose) GetServiceID(ctx context.Context, appID uuid.UUID) (uuid.UUID, error) {
	if c.ServiceIdentity == nil {
		return appID, nil
	}
	return c.ServiceIdentity(ctx, appID)
}

type EventLog interface {
	Append(ctx context.Context, orgID uuid.UUID, event realtimedomain.Event) (int64, error)
	Recent(ctx context.Context, orgID uuid.UUID, limit int) ([]realtimedomain.Event, error)
}

type ProjectVarStore interface {
	ListVariables(ctx context.Context, projectID, environmentID uuid.UUID) ([]variablesDomain.Variable, error)
}

type ProjectVarWriter interface {
	UpsertVariable(ctx context.Context, variable *variablesDomain.Variable) (*variablesDomain.Variable, error)
}

type ServiceVariableStore interface {
	ListServiceEnvVars(ctx context.Context, serviceID uuid.UUID) ([]appsdomain.EnvVar, error)
	UpsertServiceEnvVar(ctx context.Context, serviceID uuid.UUID, name, value string, secret bool) error
}

type ResolvedProjectVariableStore interface {
	ListResolvedVariables(ctx context.Context, projectID, environmentID uuid.UUID) ([]variablesDomain.Variable, error)
}

type EffectiveVariableResolver interface {
	Effective(ctx context.Context, appID, orgID uuid.UUID) (map[string]string, error)
}

type composePortDefinition struct {
	Services map[string]struct {
		Ports []any `yaml:"ports"`
	} `yaml:"services"`
}

func (c *Compose) Environment(ctx context.Context, id, orgID uuid.UUID) ([]variablesDomain.Variable, error) {
	app, err := c.Get(ctx, id, orgID)
	if err != nil {
		return nil, err
	}
	if c.ProjectVars == nil {
		return []variablesDomain.Variable{}, nil
	}
	merged := map[string]variablesDomain.Variable{}
	project, err := c.ProjectVars.ListVariables(ctx, app.ProjectID, uuid.Nil)
	if err != nil {
		return nil, err
	}
	for _, variable := range project {
		merged[variable.Key] = variable
	}
	if app.EnvironmentID != nil {
		environment, err := c.ProjectVars.ListVariables(ctx, app.ProjectID, *app.EnvironmentID)
		if err != nil {
			return nil, err
		}
		for _, variable := range environment {
			merged[variable.Key] = variable
		}
	}
	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]variablesDomain.Variable, 0, len(keys))
	for _, key := range keys {
		result = append(result, merged[key])
	}
	return result, nil
}

type AppStore interface {
	GetProject(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.Project, error)
	GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error)
	GetEnvironment(ctx context.Context, id, projectID uuid.UUID) (*appsdomain.Environment, error)
}

type AppPortUpdater interface {
	UpdateAppPort(ctx context.Context, id uuid.UUID, port int) error
}

type DeploymentStore interface {
	GetDeploymentCompose(ctx context.Context, depID uuid.UUID) (string, error)
}

func (c *Compose) Create(ctx context.Context, orgID, projectID uuid.UUID, name, content string, environmentID *uuid.UUID) (*domain.ComposeApp, error) {
	if _, err := c.Apps.GetProject(ctx, projectID, orgID); err != nil {
		return nil, err
	}
	if environmentID != nil {
		if _, err := c.Apps.GetEnvironment(ctx, *environmentID, projectID); err != nil {
			return nil, err
		}
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrValidation
	}
	if !validYAML(content) {
		return nil, domain.ErrValidation
	}
	content, err := composeengine.NormalizeUserCompose(content)
	if err != nil {
		return nil, fmt.Errorf("%w: normalize compose configuration: %v", domain.ErrValidation, err)
	}
	if err := composeengine.ValidatePolicy(content); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrValidation, err)
	}
	port, hasPort, err := composePortSelection(content)
	if err != nil {
		return nil, domain.ErrValidation
	}
	if !hasPort {
		port, err = composeContainerPort(content)
		if err != nil {
			return nil, fmt.Errorf("parse compose container port: %w", err)
		}
	}
	return c.Store.CreateComposeApp(ctx, &domain.ComposeApp{
		OrgID: orgID, ProjectID: projectID, EnvironmentID: environmentID, Name: name, Compose: content, Port: port, Status: "pending",
	})
}

func composePublishedPort(content string) (int, bool, error) {
	var definition composePortDefinition
	if err := yaml.Unmarshal([]byte(content), &definition); err != nil {
		return 0, false, err
	}
	for _, service := range definition.Services {
		for _, raw := range service.Ports {
			switch value := raw.(type) {
			case string:
				parts := strings.Split(strings.TrimSpace(value), ":")
				if len(parts) < 2 {
					continue
				}
				host := strings.TrimSpace(parts[len(parts)-2])
				host = strings.TrimPrefix(host, "[")
				host = strings.TrimSuffix(host, "]")
				if parsed, err := strconv.Atoi(host); err == nil && parsed > 0 {
					return parsed, true, nil
				}
			case map[string]any:
				if published, ok := value["published"]; ok {
					if parsed, err := strconv.Atoi(fmt.Sprint(published)); err == nil && parsed > 0 {
						return parsed, true, nil
					}
				}
			}
		}
	}
	return 0, false, nil
}

func composePortSelection(content string) (int, bool, error) {
	port, published, err := composePublishedPort(content)
	if err != nil || published {
		return port, published, err
	}
	port, err = composeContainerPort(content)
	return port, false, err
}

func composeRuntimePort(content string, configured int, variables map[string]string) (int, error) {
	if variables != nil {
		if port := parseComposePort(variables["PORT"]); port > 0 {
			return port, nil
		}
	}
	resolved, err := interpolateComposeVariables(content, variables)
	if err != nil {
		return 0, err
	}
	port, _, err := composePortSelection(resolved)
	if err != nil {
		return 0, err
	}
	if port > 0 {
		return port, nil
	}
	if configured > 0 {
		return configured, nil
	}
	return 0, nil
}

func materializeComposePortBindings(content string, fallback int) (string, error) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", err
	}
	root := &document
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	services := nodeValue(root, "services")
	if services == nil || services.Kind != yaml.MappingNode {
		return content, nil
	}
	if fallback <= 0 && composeHasPortBindings(services) {
		return "", errors.New("compose port cannot be resolved; set PORT or declare a concrete port")
	}
	for i := 0; i+1 < len(services.Content); i += 2 {
		service := services.Content[i+1]
		ports := nodeValue(service, "ports")
		if ports == nil || ports.Kind != yaml.SequenceNode {
			continue
		}
		for _, port := range ports.Content {
			if port.Kind == yaml.ScalarNode {
				port.Value = materializeComposePortValue(port.Value, fallback)
				port.Tag = "!!str"
				continue
			}
			if port.Kind != yaml.MappingNode {
				continue
			}
			for _, key := range []string{"published", "target"} {
				value := nodeValue(port, key)
				if value == nil || value.Kind != yaml.ScalarNode {
					continue
				}
				value.Value = materializeComposePortValue(value.Value, fallback)
				value.Tag = "!!str"
			}
			if nodeValue(port, "target") == nil {
				setScalarValue(port, "target", strconv.Itoa(fallback))
			}
		}
	}
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return "", err
	}
	_ = encoder.Close()
	return buffer.String(), nil
}

func materializeComposePortValue(value string, fallback int) string {
	value = strings.TrimSpace(value)
	value = replaceComposePortVariables(value, fallback)
	if value == "" || strings.Trim(value, ": ") == "" {
		return strconv.Itoa(fallback)
	}
	if strings.HasPrefix(value, ":") {
		value = strconv.Itoa(fallback) + value
	}
	if strings.HasSuffix(value, ":") {
		value += strconv.Itoa(fallback)
	}
	return value
}

func ensureComposeRuntimePortEnvironment(content string, port int, serviceType string) (string, error) {
	if serviceType != "app" || port <= 0 {
		return content, nil
	}
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", err
	}
	root := &document
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	services := nodeValue(root, "services")
	if services == nil || services.Kind != yaml.MappingNode {
		return content, nil
	}
	service := nodeValue(services, "app")
	if service == nil {
		return content, nil
	}
	environment := nodeValue(service, "environment")
	if environment == nil {
		environment = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		service.Content = append(service.Content, keyNode("environment"), environment)
	}
	value := strconv.Itoa(port)
	switch environment.Kind {
	case yaml.MappingNode:
		existing := nodeValue(environment, "PORT")
		if existing == nil {
			setScalarValue(environment, "PORT", value)
		} else if existing.Kind == yaml.ScalarNode && strings.TrimSpace(existing.Value) == "" {
			existing.Tag = "!!str"
			existing.Value = value
		}
	case yaml.SequenceNode:
		found := false
		for _, item := range environment.Content {
			if item.Kind != yaml.ScalarNode {
				continue
			}
			key, itemValue, hasValue := strings.Cut(item.Value, "=")
			if strings.TrimSpace(key) != "PORT" {
				continue
			}
			if !hasValue || strings.TrimSpace(itemValue) == "" {
				item.Value = "PORT=" + value
			}
			found = true
		}
		if !found {
			environment.Content = append(environment.Content, valueNode("PORT="+value))
		}
	}
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return "", err
	}
	_ = encoder.Close()
	return buffer.String(), nil
}

func composeHasPortBindings(services *yaml.Node) bool {
	for i := 0; i+1 < len(services.Content); i += 2 {
		service := services.Content[i+1]
		if nodeValue(service, "ports") != nil {
			return true
		}
	}
	return false
}

func replaceComposePortVariables(value string, fallback int) string {
	port := strconv.Itoa(fallback)
	for {
		start := strings.Index(value, "${")
		if start < 0 {
			return strings.ReplaceAll(value, "$PORT", port)
		}
		relativeEnd := strings.IndexByte(value[start+2:], '}')
		if relativeEnd < 0 {
			return value
		}
		end := start + 2 + relativeEnd
		token := value[start+2 : end]
		replacement := port
		if _, defaultValue, ok := strings.Cut(token, ":-"); ok {
			if parsed := parseComposePort(defaultValue); parsed > 0 {
				replacement = strconv.Itoa(parsed)
			}
		}
		value = value[:start] + replacement + value[end+1:]
	}
}

func PublishedPort(content string) (int, bool, error) {
	return composePublishedPort(content)
}

func composeContainerPort(content string) (int, error) {
	var document map[string]any
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return 0, err
	}
	services, ok := document["services"].(map[string]any)
	if !ok || len(services) == 0 {
		return 0, nil
	}
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		service, ok := services[name].(map[string]any)
		if !ok {
			continue
		}
		if port := composeServicePort(service["expose"]); port > 0 {
			return port, nil
		}
		if port := composeAddressPort(service["environment"]); port > 0 {
			return port, nil
		}
		if port := composeEnvironmentPort(service["environment"]); port > 0 {
			return port, nil
		}
	}
	for _, name := range names {
		service, ok := services[name].(map[string]any)
		if !ok {
			continue
		}
		if port := composeServicePort(service["ports"]); port > 0 {
			return port, nil
		}
	}
	return 0, nil
}

func composeServicePort(raw any) int {
	values, ok := raw.([]any)
	if !ok {
		return 0
	}
	for _, value := range values {
		if port := composePortValue(value); port > 0 {
			return port
		}
	}
	return 0
}

func composePortValue(raw any) int {
	switch value := raw.(type) {
	case string:
		parts := strings.Split(strings.TrimSpace(value), ":")
		if len(parts) > 1 {
			return parseComposePort(parts[len(parts)-1])
		}
		return parseComposePort(value)
	case int:
		return value
	case uint64:
		return int(value)
	case map[string]any:
		if target, ok := value["target"]; ok {
			return composePortValue(target)
		}
	}
	return 0
}

func composeAddressPort(raw any) int {
	var values []string
	switch value := raw.(type) {
	case []any:
		for _, item := range value {
			values = append(values, fmt.Sprint(item))
		}
	case map[string]any:
		for key, value := range value {
			values = append(values, key+"="+fmt.Sprint(value))
		}
	default:
		return 0
	}
	for _, value := range values {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 || !strings.Contains(strings.ToUpper(parts[0]), "ADDRESS") || strings.Contains(strings.ToUpper(parts[0]), "CONSOLE") {
			continue
		}
		if port := parseComposePort(parts[1]); port > 0 {
			return port
		}
	}
	return 0
}

func composeEnvironmentPort(raw any) int {
	var values []string
	switch value := raw.(type) {
	case []any:
		for _, item := range value {
			values = append(values, fmt.Sprint(item))
		}
	case map[string]any:
		for key, value := range value {
			values = append(values, key+"="+fmt.Sprint(value))
		}
	default:
		return 0
	}
	for _, value := range values {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.ToUpper(strings.TrimSpace(parts[0]))
		if name != "PORT" && !strings.HasSuffix(name, "_PORT") || strings.Contains(name, "CONSOLE") {
			continue
		}
		if port := parseComposePort(parts[1]); port > 0 {
			return port
		}
	}
	return 0
}

func parseComposePort(value string) int {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		token := strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")
		if _, defaultValue, ok := strings.Cut(token, ":-"); ok {
			value = defaultValue
		} else if _, defaultValue, ok := strings.Cut(token, "-"); ok {
			value = defaultValue
		} else {
			return 0
		}
	}
	if slash := strings.IndexByte(value, '/'); slash >= 0 {
		value = value[:slash]
	}
	if colon := strings.LastIndexByte(value, ':'); colon >= 0 {
		value = value[colon+1:]
	}
	value = strings.Trim(value, "[]")
	port, err := strconv.Atoi(value)
	if err != nil || port <= 0 || port > 65535 {
		return 0
	}
	return port
}

func (c *Compose) Get(ctx context.Context, id, orgID uuid.UUID) (*domain.ComposeApp, error) {
	app, err := c.Store.GetComposeApp(ctx, id)
	if err != nil {
		return nil, err
	}
	if app.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	return app, nil
}

func (c *Compose) Up(ctx context.Context, id, orgID uuid.UUID) error {
	app, err := c.Get(ctx, id, orgID)
	if err != nil {
		return err
	}
	if err := c.Store.SetComposeStatus(ctx, id, "deploying"); err != nil {
		return err
	}
	if _, err := c.runCompose(ctx, app, true, "up", "-d", "--build"); err != nil {
		_ = c.Store.SetComposeStatus(ctx, id, "error")
		return err
	}
	if c.Runtime != nil {
		serviceID, serviceErr := c.GetServiceID(ctx, id)
		if serviceErr != nil {
			_ = c.Store.SetComposeStatus(ctx, id, "error")
			return serviceErr
		}
		if waitErr := c.waitForComposeContainers(ctx, serviceID, id); waitErr != nil {
			_ = c.Store.SetComposeStatus(ctx, id, "error")
			return waitErr
		}
	}
	if err := c.Store.SetComposeStatus(ctx, id, "running"); err != nil {
		return err
	}
	c.recordEvent(ctx, app, "compose.running")
	return nil
}

func (c *Compose) UpApp(ctx context.Context, id, orgID uuid.UUID) (string, error) {
	if c.Apps == nil {
		return "", errors.New("application store unavailable")
	}
	app, err := c.Apps.GetApp(ctx, id, orgID)
	if err != nil {
		return "", err
	}
	composeApp := &domain.ComposeApp{
		ID: id, OrgID: app.OrgID, ProjectID: app.ProjectID, EnvironmentID: app.EnvironmentID,
		ServiceID: id, Name: app.Name, Compose: "", Port: app.Port,
	}
	if _, err := c.runComposeForService(ctx, composeApp, true, "app", "up", "-d", "--build"); err != nil {
		return "", err
	}
	serviceID, err := c.GetServiceID(ctx, id)
	if err != nil {
		serviceID = id
	}
	if c.Runtime == nil {
		return "", errors.New("compose runtime unavailable")
	}
	if err := c.waitForComposeContainers(ctx, serviceID, id); err != nil {
		return "", err
	}
	containers, err := c.Runtime.ListContainers(ctx)
	if err != nil {
		return "", err
	}
	for _, container := range containers {
		if container.Labels["aether.service-id"] == serviceID.String() || container.Labels["aether.spec-id"] == id.String() {
			return container.ID, nil
		}
	}
	return "", errors.New("compose container not found")
}

func (c *Compose) waitForComposeContainers(ctx context.Context, serviceID, specID uuid.UUID) error {
	lister, ok := c.Runtime.(worker.ServiceContainerRuntime)
	if !ok {
		return nil
	}
	deadline := time.NewTimer(2 * time.Minute)
	defer deadline.Stop()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var readySince time.Time
	for {
		containers, err := lister.ListServiceContainers(ctx, serviceID, specID)
		if err != nil {
			return fmt.Errorf("inspect compose containers: %w", err)
		}
		if len(containers) > 0 {
			running := 0
			for _, container := range containers {
				switch container.State {
				case "running", "restarting":
					running++
				case "exited", "dead":
					if container.ExitCode != 0 {
						return fmt.Errorf("compose container %s exited with code %d", container.Name, container.ExitCode)
					}
				default:
					return fmt.Errorf("compose container %s is %s", container.Name, container.State)
				}
				if container.Healthy != nil && !*container.Healthy {
					return fmt.Errorf("compose container %s is unhealthy", container.Name)
				}
			}
			if running == 0 {
				readySince = time.Time{}
			} else if readySince.IsZero() {
				readySince = time.Now()
			} else if time.Since(readySince) >= 10*time.Second {
				return nil
			}
		} else {
			readySince = time.Time{}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return errors.New("compose containers did not become ready before timeout")
		case <-ticker.C:
		}
	}
}

func (c *Compose) Down(ctx context.Context, id, orgID uuid.UUID) error {
	app, err := c.Get(ctx, id, orgID)
	if err != nil {
		return err
	}
	if _, err := c.runCompose(ctx, app, false, "down"); err != nil {
		_ = c.Store.SetComposeStatus(ctx, id, "error")
		return err
	}
	if err := c.Store.SetComposeStatus(ctx, id, "stopped"); err != nil {
		return err
	}
	c.recordEvent(ctx, app, "compose.stopped")
	return nil
}

func (c *Compose) Start(ctx context.Context, id, orgID uuid.UUID) error {
	app, err := c.Get(ctx, id, orgID)
	if err != nil {
		return err
	}
	if _, err := c.runCompose(ctx, app, false, "start"); err != nil {
		_ = c.Store.SetComposeStatus(ctx, id, "error")
		return err
	}
	if err := c.Store.SetComposeStatus(ctx, id, "running"); err != nil {
		return err
	}
	c.recordEvent(ctx, app, "compose.running")
	return nil
}

func (c *Compose) Stop(ctx context.Context, id, orgID uuid.UUID) error {
	app, err := c.Get(ctx, id, orgID)
	if err != nil {
		return err
	}
	if _, err := c.runCompose(ctx, app, false, "stop"); err != nil {
		_ = c.Store.SetComposeStatus(ctx, id, "error")
		return err
	}
	if err := c.Store.SetComposeStatus(ctx, id, "stopped"); err != nil {
		return err
	}
	c.recordEvent(ctx, app, "compose.stopped")
	return nil
}

func (c *Compose) Restart(ctx context.Context, id, orgID uuid.UUID) error {
	app, err := c.Get(ctx, id, orgID)
	if err != nil {
		return err
	}
	if _, err := c.runCompose(ctx, app, false, "restart"); err != nil {
		_ = c.Store.SetComposeStatus(ctx, id, "error")
		return err
	}
	if err := c.Store.SetComposeStatus(ctx, id, "running"); err != nil {
		return err
	}
	c.recordEvent(ctx, app, "compose.running")
	return nil
}

func (c *Compose) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	app, err := c.Get(ctx, id, orgID)
	if err != nil {
		return err
	}
	var downErr error
	if _, err := c.runCompose(ctx, app, false, "down"); err != nil {
		downErr = err
	}
	removeErr := c.removeRuntimeContainers(ctx, id)
	if downErr != nil && removeErr != nil {
		return downErr
	}
	if removeErr != nil {
		return removeErr
	}
	return c.Store.DeleteComposeApp(ctx, id, orgID)
}

func (c *Compose) removeRuntimeContainers(ctx context.Context, id uuid.UUID) error {
	if c.Runtime == nil {
		return nil
	}
	labels := []string{"aether.spec-id=" + id.String()}
	serviceID, err := c.GetServiceID(ctx, id)
	if err == nil {
		labels = append(labels, "aether.service-id="+serviceID.String())
	}
	for _, label := range labels {
		if err := c.Runtime.RemoveByLabel(ctx, label); err != nil {
			return err
		}
	}
	return nil
}

func (c *Compose) recordEvent(ctx context.Context, app *domain.ComposeApp, eventType string) {
	if c.Events == nil {
		return
	}
	serviceID, err := c.GetServiceID(ctx, app.ID)
	if err != nil {
		serviceID = app.ID
	}
	_, _ = c.Events.Append(ctx, app.OrgID, realtimedomain.Event{
		Type: eventType, Aggregate: "service", OrgID: app.OrgID.String(), ProjectID: app.ProjectID.String(),
		ResourceType: "service", ResourceID: serviceID.String(), AppID: app.ID.String(), ServiceID: serviceID.String(), TS: time.Now().UTC(),
	})
}

func (c *Compose) Timeline(ctx context.Context, id, orgID uuid.UUID) ([]realtimedomain.Event, error) {
	if _, err := c.Get(ctx, id, orgID); err != nil {
		return nil, err
	}
	if c.Events == nil {
		return []realtimedomain.Event{}, nil
	}
	serviceID, err := c.GetServiceID(ctx, id)
	if err != nil {
		serviceID = id
	}
	events, err := c.Events.Recent(ctx, orgID, 200)
	if err != nil {
		return nil, err
	}
	out := make([]realtimedomain.Event, 0, len(events))
	for _, event := range events {
		if event.ResourceID == id.String() || event.ResourceID == serviceID.String() || event.ServiceID == serviceID.String() {
			out = append(out, event)
		}
	}
	return out, nil
}

func (c *Compose) ContainerID(ctx context.Context, id, orgID uuid.UUID) (string, error) {
	containers, err := c.ContainerIDs(ctx, id, orgID)
	if err != nil {
		return "", err
	}
	for _, item := range containers {
		if item.State == "running" || item.State == "restarting" {
			return item.ID, nil
		}
	}
	return "", errors.New("no active container")
}

func (c *Compose) ContainerIDs(ctx context.Context, id, orgID uuid.UUID) ([]worker.ContainerInfo, error) {
	if _, err := c.Get(ctx, id, orgID); err != nil {
		return nil, err
	}
	serviceID, err := c.GetServiceID(ctx, id)
	if err != nil {
		serviceID = id
	}
	values := []uuid.UUID{serviceID}
	if id != serviceID {
		values = append(values, id)
	}
	if c.Runtime == nil {
		return nil, errors.New("container runtime unavailable")
	}
	containers, queryErr := c.Runtime.ListContainers(ctx)
	if queryErr != nil {
		return nil, fmt.Errorf("resolve compose containers: %w", queryErr)
	}
	matched := make([]worker.ContainerInfo, 0)
	for _, item := range containers {
		for _, value := range values {
			if item.Labels["aether.service-id"] == value.String() || item.Labels["aether.spec-id"] == value.String() {
				matched = append(matched, item)
				break
			}
		}
	}
	if len(matched) == 0 {
		return nil, errors.New("no compose containers")
	}
	return matched, nil
}

type ComposeValidation struct {
	Valid      bool                 `json:"valid"`
	Services   []ComposeServiceInfo `json:"services"`
	Volumes    []string             `json:"volumes"`
	Networks   []string             `json:"networks"`
	Errors     []string             `json:"errors"`
	Warnings   []string             `json:"warnings"`
	DependsOn  map[string][]string  `json:"depends_on"`
	TotalPorts int                  `json:"total_ports"`
}

type ComposeServiceInfo struct {
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Build   string   `json:"build"`
	Ports   []string `json:"ports"`
	Volumes []string `json:"volumes"`
	Restart string   `json:"restart"`
}

func (c *Compose) Validate(content string) ComposeValidation {
	out := ComposeValidation{Errors: []string{}, Warnings: []string{}, Services: []ComposeServiceInfo{}, Volumes: []string{}, Networks: []string{}}
	out.DependsOn = map[string][]string{}
	var cf struct {
		Services map[string]struct {
			Image     string            `yaml:"image"`
			Build     any               `yaml:"build"`
			Command   string            `yaml:"command"`
			Ports     []string          `yaml:"ports"`
			Env       map[string]string `yaml:"environment"`
			Volumes   []string          `yaml:"volumes"`
			Restart   string            `yaml:"restart"`
			DependsOn any               `yaml:"depends_on"`
		} `yaml:"services"`
		Volumes  map[string]any `yaml:"volumes"`
		Networks map[string]any `yaml:"networks"`
	}
	if err := yaml.Unmarshal([]byte(content), &cf); err != nil {
		out.Valid = false
		out.Errors = append(out.Errors, "YAML parse error: "+err.Error())
		return out
	}
	if err := composeengine.ValidatePolicy(content); err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	if len(cf.Services) == 0 {
		out.Errors = append(out.Errors, "no services defined")
		return out
	}
	volumes := map[string]bool{}
	for name := range cf.Volumes {
		volumes[name] = true
		out.Volumes = append(out.Volumes, name)
	}
	for name := range cf.Networks {
		out.Networks = append(out.Networks, name)
	}
	for name, svc := range cf.Services {
		info := ComposeServiceInfo{Name: name, Image: svc.Image, Ports: svc.Ports, Volumes: svc.Volumes, Restart: svc.Restart}
		if svc.Build != nil {
			info.Build = fmt.Sprintf("%v", svc.Build)
		}
		out.Services = append(out.Services, info)
		if list := dependsOnList(svc.DependsOn); len(list) > 0 {
			out.DependsOn[name] = list
		}
		for _, v := range svc.Volumes {
			if strings.HasPrefix(v, "./") || strings.HasPrefix(v, "/") || strings.HasPrefix(v, "~") {
				continue
			}
			if !strings.Contains(v, ":") && !volumes[v] {
				out.Warnings = append(out.Warnings, fmt.Sprintf("service %s: volume %q not declared in volumes:", name, v))
			}
		}
		out.TotalPorts += len(svc.Ports)
	}
	if len(out.Errors) == 0 {
		out.Valid = true
	}
	return out
}

func dependsOnList(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case map[any]any:
		out := make([]string, 0, len(v))
		for key := range v {
			if s, ok := key.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func (c *Compose) AppCompose(ctx context.Context, appID, orgID uuid.UUID) (string, error) {
	app, err := c.Apps.GetApp(ctx, appID, orgID)
	if err != nil {
		return "", err
	}
	service := map[string]any{
		"image":      app.Image,
		"restart":    "no",
		"mem_limit":  "512m",
		"cpus":       "1.0",
		"pids_limit": 256,
	}
	if app.Port > 0 {
		service["ports"] = []string{fmt.Sprintf("%d:%d", app.Port, app.Port)}
	}
	if app.MemMB > 0 {
		service["mem_limit"] = fmt.Sprintf("%dm", app.MemMB)
	}
	compose := map[string]any{
		"version":  "3.8",
		"services": map[string]any{app.Name: service},
	}
	raw, err := yaml.Marshal(compose)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (c *Compose) DeploymentCompose(ctx context.Context, depID uuid.UUID) (string, error) {
	return c.Deployments.GetDeploymentCompose(ctx, depID)
}

func (c *Compose) Logs(ctx context.Context, id, orgID uuid.UUID, follow bool) (string, error) {
	app, err := c.Get(ctx, id, orgID)
	if err != nil {
		return "", err
	}
	args := []string{"logs", "--no-color"}
	if follow {
		args = append(args, "--follow")
	}
	return c.runCompose(ctx, app, false, args...)
}

func (c *Compose) runCompose(ctx context.Context, app *domain.ComposeApp, refresh bool, args ...string) (string, error) {
	return c.runComposeForService(ctx, app, refresh, "compose", args...)
}

func (c *Compose) runComposeForService(ctx context.Context, app *domain.ComposeApp, refresh bool, serviceType string, args ...string) (string, error) {
	if c.DataDir == "" {
		return "", fmt.Errorf("data dir not configured")
	}
	if c.ComposeRuntime == nil {
		return "", errors.New("compose runtime unavailable")
	}
	dir := filepath.Join(c.DataDir, "compose", app.ID.String())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	workDir := dir
	file := filepath.Join(dir, "docker-compose.yml")
	content := app.Compose
	isUpCommand := hasComposeCommand(args, "up")
	if c.Source != nil && c.Clone != nil {
		serviceID, err := c.GetServiceID(ctx, app.ID)
		if err != nil {
			return "", err
		}
		source, err := c.Source.GetByService(ctx, serviceID, app.OrgID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return "", err
		}
		if source != nil {
			checkoutPath := filepath.Join(dir, "checkout")
			if refresh {
				if err := os.RemoveAll(checkoutPath); err != nil {
					return "", err
				}
			}
			checkout := checkoutPath
			if _, statErr := os.Stat(checkoutPath); errors.Is(statErr, os.ErrNotExist) {
				checkout, err = c.Clone.Clone(ctx, source, checkoutPath)
				if err != nil {
					return "", err
				}
			}
			root, err := repositoryPath(source.RootDirectory)
			if err != nil {
				return "", fmt.Errorf("invalid compose root directory: %w", err)
			}
			projectRoot := filepath.Join(checkout, root)
			if !pathWithin(checkout, projectRoot) {
				return "", errors.New("compose root directory escapes repository checkout")
			}
			composeFile := source.ComposeFile
			if composeFile == "" {
				composeFile = "docker-compose.yml"
			}
			composeFile, err = repositoryPath(composeFile)
			if err != nil {
				return "", fmt.Errorf("invalid compose file path: %w", err)
			}
			file = filepath.Join(projectRoot, composeFile)
			if !pathWithin(checkout, file) {
				return "", errors.New("compose file escapes repository checkout")
			}
			workDir = filepath.Dir(file)
			data, err := os.ReadFile(file)
			if err != nil {
				return "", fmt.Errorf("read compose file from checkout: %w", err)
			}
			content = string(data)
		}
	}
	normalized, normalizeErr := composeengine.NormalizeNamedResourceDefinitions(content)
	if normalizeErr != nil {
		return "", normalizeErr
	}
	content = normalized
	userCompose, userComposeErr := composeengine.NormalizeUserCompose(content)
	if userComposeErr != nil {
		return "", userComposeErr
	}
	content = userCompose
	if isUpCommand {
		content, inlineErr := inlineTemplateConfigFiles(content, dir)
		if inlineErr != nil {
			return "", fmt.Errorf("materialize legacy template configs: %w", inlineErr)
		}
		content, mountErr := materializeTemplateConfigFiles(content, dir)
		if mountErr != nil {
			return "", fmt.Errorf("materialize template configs: %w", mountErr)
		}
		materialized, portErr := ensureRustFSPublishedPorts(content)
		if portErr != nil {
			return "", fmt.Errorf("materialize RustFS ports: %w", portErr)
		}
		content = materialized
	}
	if serviceType == "app" {
		applicationCompose, normalizeErr := composeengine.NormalizeUserCompose(content)
		if normalizeErr != nil {
			return "", fmt.Errorf("normalize application compose ports: %w", normalizeErr)
		}
		content = applicationCompose
	}
	if err := composeengine.ValidatePolicy(content); err != nil {
		return "", err
	}
	if isUpCommand {
		serviceID, err := c.GetServiceID(ctx, app.ID)
		if err != nil {
			serviceID = app.ID
		}
		injected, err := injectComposeLabels(content, map[string]string{
			"aether.owner":        "user",
			"aether.service-type": serviceType,
			"aether.service-id":   serviceID.String(),
			"aether.spec-id":      app.ID.String(),
			"aether.project-id":   app.ProjectID.String(),
			"aether.service-name": app.Name,
		})
		if err != nil {
			return "", fmt.Errorf("inject compose labels: %w", err)
		}
		if err := c.ensureComposeVariables(ctx, app, content); err != nil {
			return "", err
		}
		variables, err := c.effectiveVariables(ctx, app)
		if err != nil {
			return "", err
		}
		if variables == nil {
			variables = map[string]string{}
		}
		if strings.TrimSpace(variables["PORT"]) == "" && filepath.Clean(workDir) != filepath.Clean(dir) {
			if data, readErr := os.ReadFile(filepath.Join(workDir, ".env")); readErr == nil {
				if port := strings.TrimSpace(parseEnvFile(string(data))["PORT"]); port != "" {
					variables["PORT"] = port
				}
			}
		}
		configuredPort := app.Port
		if serviceType == "app" {
			configuredPort = 0
		}
		runtimePort, err := composeRuntimePort(content, configuredPort, variables)
		if err != nil {
			return "", fmt.Errorf("resolve compose port: %w", err)
		}
		if runtimePort > 0 && strings.TrimSpace(variables["PORT"]) == "" {
			variables["PORT"] = strconv.Itoa(runtimePort)
		}
		if serviceType == "app" {
			if updater, ok := c.Apps.(AppPortUpdater); ok && app.Port != runtimePort {
				if err := updater.UpdateAppPort(ctx, app.ID, runtimePort); err != nil {
					return "", fmt.Errorf("persist compose port: %w", err)
				}
			}
			app.Port = runtimePort
		} else if app.Port == 0 && runtimePort > 0 {
			if updater, ok := c.Apps.(AppPortUpdater); ok {
				if err := updater.UpdateAppPort(ctx, app.ID, runtimePort); err != nil {
					return "", fmt.Errorf("persist compose port: %w", err)
				}
			}
			app.Port = runtimePort
		}
		injected, err = injectComposeEnvironment(injected, variables)
		if err != nil {
			return "", fmt.Errorf("inject compose environment: %w", err)
		}
		injected, err = interpolateComposeVariables(injected, variables)
		if err != nil {
			return "", fmt.Errorf("interpolate compose variables: %w", err)
		}
		injected, err = ensureComposeRuntimePortEnvironment(injected, runtimePort, serviceType)
		if err != nil {
			return "", fmt.Errorf("inject compose port environment: %w", err)
		}
		injected, err = injectComposeSecurityDefaults(injected)
		if err != nil {
			return "", fmt.Errorf("inject compose security defaults: %w", err)
		}
		injected, err = materializeComposePortBindings(injected, runtimePort)
		if err != nil {
			return "", fmt.Errorf("materialize compose ports: %w", err)
		}
		content = injected
		if err := composeengine.ValidatePolicy(content); err != nil {
			return "", err
		}
		overlay := filepath.Join(dir, "compose.generated.yml")
		if err := os.WriteFile(overlay, []byte(content), 0o644); err != nil {
			return "", err
		}
		file = overlay
	}
	if !isUpCommand {
		generated := filepath.Join(dir, "compose.generated.yml")
		if _, err := os.Stat(generated); err == nil {
			file = generated
		}
	}
	envFile := filepath.Join(dir, ".env")
	if err := c.writeEnvFile(ctx, dir, workDir, app); err != nil {
		return "", err
	}
	if !isUpCommand {
		if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
				return "", err
			}
		}
	}
	project := composeengine.Project{Directory: workDir, File: file, Name: "aether-" + app.ID.String()[:8]}
	if _, err := os.Stat(envFile); err == nil {
		project.EnvFile = envFile
	}
	var output string
	var err error
	logSink := func(line string) { worker.EmitDeploymentLog(ctx, line) }
	streamed := false
	if streaming, ok := c.ComposeRuntime.(composeengine.StreamingExecutor); ok {
		streamed = true
		output, err = streaming.ExecuteWithLogs(ctx, project, logSink, args...)
	} else {
		output, err = c.ComposeRuntime.Execute(ctx, project, args...)
	}
	if err != nil {
		if !streamed && strings.TrimSpace(output) != "" {
			logSink(output)
		}
		return "", err
	}
	if !streamed && strings.TrimSpace(output) != "" {
		logSink(output)
	}
	return output, nil
}

func hasComposeCommand(args []string, command string) bool {
	for _, arg := range args {
		if arg == command {
			return true
		}
	}
	return false
}

func inlineTemplateConfigFiles(content, baseDir string) (string, error) {
	var document map[string]any
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", err
	}
	configs, ok := document["configs"].(map[string]any)
	if !ok {
		return content, nil
	}
	changed := false
	for name, raw := range configs {
		if !strings.HasPrefix(name, "aether-template-file-") {
			continue
		}
		config, ok := raw.(map[string]any)
		if !ok || config["content"] != nil {
			continue
		}
		file, ok := config["file"].(string)
		if !ok || !strings.HasPrefix(file, "./template-mounts/") {
			continue
		}
		candidate := filepath.Join(baseDir, filepath.FromSlash(strings.TrimPrefix(file, "./")))
		if !pathWithin(baseDir, candidate) {
			return "", fmt.Errorf("template config %s escapes its workspace", name)
		}
		data, err := os.ReadFile(candidate)
		if err != nil {
			return "", fmt.Errorf("read template config %s: %w", name, err)
		}
		config["content"] = string(data)
		delete(config, "file")
		changed = true
	}
	if !changed {
		return content, nil
	}
	encoded, err := yaml.Marshal(document)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func materializeTemplateConfigFiles(content, baseDir string) (string, error) {
	var document map[string]any
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", err
	}
	configs, ok := document["configs"].(map[string]any)
	if !ok {
		return content, nil
	}
	mountDir := filepath.Join(baseDir, "template-mounts")
	changed := false
	for name, raw := range configs {
		if !strings.HasPrefix(name, "aether-template-file-") {
			continue
		}
		config, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		contentValue, ok := config["content"].(string)
		if !ok {
			continue
		}
		fileName := strings.TrimPrefix(name, "aether-template-file-")
		if fileName == "" || strings.ContainsAny(fileName, `/\\`) {
			return "", fmt.Errorf("template config %s has an invalid file name", name)
		}
		filePath := filepath.Join(mountDir, fileName)
		if !pathWithin(baseDir, filePath) {
			return "", fmt.Errorf("template config %s escapes its workspace", name)
		}
		if err := os.MkdirAll(mountDir, 0o755); err != nil {
			return "", err
		}
		contentValue = strings.ReplaceAll(contentValue, "$", "$$")
		if err := os.WriteFile(filePath, []byte(contentValue), 0o600); err != nil {
			return "", err
		}
		config["file"] = "./template-mounts/" + fileName
		delete(config, "content")
		changed = true
	}
	if !changed {
		return content, nil
	}
	encoded, err := yaml.Marshal(document)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func ensureRustFSPublishedPorts(content string) (string, error) {
	var document map[string]any
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", err
	}
	services, ok := document["services"].(map[string]any)
	if !ok {
		return content, nil
	}
	changed := false
	for _, raw := range services {
		service, ok := raw.(map[string]any)
		if !ok || !strings.HasPrefix(strings.ToLower(strings.TrimSpace(fmt.Sprint(service["image"]))), "rustfs/rustfs") {
			continue
		}
		ports, ok := service["ports"].([]any)
		if !ok {
			ports = []any{}
		}
		for _, port := range []int{9000, 9001} {
			if containsPublishedContainerPort(ports, port) {
				continue
			}
			ports = append(ports, fmt.Sprintf("%d:%d", port, port))
			changed = true
		}
		if changed {
			service["ports"] = ports
		}
	}
	if !changed {
		return content, nil
	}
	encoded, err := yaml.Marshal(document)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func containsPublishedContainerPort(ports []any, containerPort int) bool {
	target := strconv.Itoa(containerPort)
	for _, raw := range ports {
		value := strings.TrimSpace(fmt.Sprint(raw))
		parts := strings.Split(value, ":")
		if len(parts) >= 2 && strings.TrimSpace(parts[len(parts)-1]) == target {
			return true
		}
	}
	return false
}

func (c *Compose) ensureComposeVariables(ctx context.Context, app *domain.ComposeApp, content string) error {
	if writer, ok := c.Apps.(ServiceVariableStore); ok {
		return c.ensureServiceComposeVariables(ctx, writer, app, content)
	}
	writer, ok := c.ProjectVars.(ProjectVarWriter)
	if !ok {
		return nil
	}
	existing, err := c.ProjectVars.ListVariables(ctx, app.ProjectID, uuid.Nil)
	if err != nil {
		return fmt.Errorf("list compose variables: %w", err)
	}
	values := make(map[string]string, len(existing))
	for _, variable := range existing {
		values[variable.Key] = variable.Value
	}
	if app.EnvironmentID != nil {
		environment, err := c.ProjectVars.ListVariables(ctx, app.ProjectID, *app.EnvironmentID)
		if err != nil {
			return fmt.Errorf("list compose environment variables: %w", err)
		}
		for _, variable := range environment {
			values[variable.Key] = variable.Value
		}
	}
	for _, variable := range templateEnvironmentVariables(content) {
		if strings.TrimSpace(values[variable.Name]) != "" {
			continue
		}
		value := strings.TrimSpace(variable.Value)
		if value == "" && isGeneratedTemplateSecret(variable.Name) {
			value, err = generatedTemplateSecret()
			if err != nil {
				return fmt.Errorf("generate compose secret: %w", err)
			}
		}
		if value == "" {
			continue
		}
		if _, err := writer.UpsertVariable(ctx, &variablesDomain.Variable{
			ProjectID: app.ProjectID,
			Key:       variable.Name,
			Value:     value,
			IsSecret:  isGeneratedTemplateSecret(variable.Name),
		}); err != nil {
			return fmt.Errorf("save compose variable %s: %w", variable.Name, err)
		}
		values[variable.Name] = value
	}
	return nil
}

func (c *Compose) ensureServiceComposeVariables(ctx context.Context, writer ServiceVariableStore, app *domain.ComposeApp, content string) error {
	serviceID := app.ServiceID
	if serviceID == uuid.Nil {
		var err error
		serviceID, err = c.GetServiceID(ctx, app.ID)
		if err != nil {
			return fmt.Errorf("resolve compose service: %w", err)
		}
	}
	existing, err := writer.ListServiceEnvVars(ctx, serviceID)
	if err != nil {
		return fmt.Errorf("list service compose variables: %w", err)
	}
	known := make(map[string]struct{}, len(existing))
	for _, variable := range existing {
		known[variable.Name] = struct{}{}
	}
	projectValues, err := c.resolvedComposeProjectValues(ctx, app)
	if err != nil {
		return err
	}
	for _, variable := range templateEnvironmentVariables(content) {
		if _, exists := known[variable.Name]; exists {
			continue
		}
		value := strings.TrimSpace(projectValues[variable.Name])
		if value == "" {
			value = strings.TrimSpace(variable.Value)
		}
		if value == "" && isGeneratedTemplateSecret(variable.Name) {
			value, err = generatedTemplateSecret()
			if err != nil {
				return fmt.Errorf("generate compose secret: %w", err)
			}
		}
		if value == "" {
			continue
		}
		if err := writer.UpsertServiceEnvVar(ctx, serviceID, variable.Name, value, isGeneratedTemplateSecret(variable.Name)); err != nil {
			return fmt.Errorf("save service compose variable %s: %w", variable.Name, err)
		}
	}
	return nil
}

func (c *Compose) resolvedComposeProjectValues(ctx context.Context, app *domain.ComposeApp) (map[string]string, error) {
	if c.ProjectVars == nil {
		return map[string]string{}, nil
	}
	read := func(environmentID uuid.UUID) ([]variablesDomain.Variable, error) {
		if reader, ok := c.ProjectVars.(ResolvedProjectVariableStore); ok {
			return reader.ListResolvedVariables(ctx, app.ProjectID, environmentID)
		}
		return c.ProjectVars.ListVariables(ctx, app.ProjectID, environmentID)
	}
	project, err := read(uuid.Nil)
	if err != nil {
		return nil, fmt.Errorf("list resolved compose variables: %w", err)
	}
	values := make(map[string]string, len(project))
	for _, variable := range project {
		if variable.Value != "" {
			values[variable.Key] = variable.Value
		}
	}
	if app.EnvironmentID != nil {
		environment, err := read(*app.EnvironmentID)
		if err != nil {
			return nil, fmt.Errorf("list resolved compose environment variables: %w", err)
		}
		for _, variable := range environment {
			if variable.Value != "" {
				values[variable.Key] = variable.Value
			}
		}
	}
	return values, nil
}

func pathWithin(root, candidate string) bool {
	root, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	candidate, err = filepath.Abs(candidate)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}

func repositoryPath(value string) (string, error) {
	value = filepath.Clean(strings.TrimSpace(value))
	if value == "." {
		return "", nil
	}
	if filepath.IsAbs(value) || value == ".." || strings.HasPrefix(value, ".."+string(os.PathSeparator)) {
		return "", errors.New("path must stay inside the repository")
	}
	return value, nil
}

func injectComposeLabels(content string, labels map[string]string) (string, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return "", err
	}
	root := &doc
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	if root.Kind != yaml.MappingNode {
		return "", fmt.Errorf("compose root is not a mapping")
	}
	var services *yaml.Node
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "services" {
			services = root.Content[i+1]
			break
		}
	}
	if services == nil || services.Kind != yaml.MappingNode {
		return "", fmt.Errorf("compose has no services mapping")
	}
	for i := 0; i+1 < len(services.Content); i += 2 {
		svc := services.Content[i+1]
		if svc.Kind != yaml.MappingNode {
			continue
		}
		svc = injectServiceLabels(svc, labels)
		services.Content[i+1] = svc
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return "", err
	}
	_ = enc.Close()
	return buf.String(), nil
}

func injectServiceLabels(svc *yaml.Node, labels map[string]string) *yaml.Node {
	for i := 0; i+1 < len(svc.Content); i += 2 {
		if svc.Content[i].Value != "labels" {
			continue
		}
		existing := svc.Content[i+1]
		switch existing.Kind {
		case yaml.MappingNode:
			for k, v := range labels {
				updated := false
				for j := 0; j+1 < len(existing.Content); j += 2 {
					if existing.Content[j].Value == k {
						existing.Content[j+1] = valueNode(v)
						updated = true
						break
					}
				}
				if !updated {
					existing.Content = append(existing.Content, keyNode(k), valueNode(v))
				}
			}
		case yaml.SequenceNode:
			for k, v := range labels {
				found := false
				for _, item := range existing.Content {
					if item.Value == k+"="+v || strings.HasPrefix(item.Value, k+"=") {
						item.Value = k + "=" + v
						found = true
						break
					}
				}
				if found {
					continue
				}
				existing.Content = append(existing.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k + "=" + v})
			}
		}
		return svc
	}
	labelsNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for k, v := range labels {
		labelsNode.Content = append(labelsNode.Content, keyNode(k), valueNode(v))
	}
	svc.Content = append(svc.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "labels"}, labelsNode)
	return svc
}

func injectComposeSecurityDefaults(content string) (string, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return "", err
	}
	root := &doc
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	services := nodeValue(root, "services")
	if services == nil || services.Kind != yaml.MappingNode {
		return "", fmt.Errorf("compose has no services mapping")
	}
	allowHostPorts := strings.Contains(strings.ToLower(content), "dokploy: allow-host-ports")
	if marker := nodeValue(root, "x-aether-allow-host-ports"); marker != nil {
		allowHostPorts = allowHostPorts || strings.EqualFold(strings.TrimSpace(marker.Value), "true")
	}
	if allowHostPorts {
		ensureScalarValue(root, "x-aether-allow-host-ports", "true")
	}
	for i := 0; i+1 < len(services.Content); i += 2 {
		service := services.Content[i+1]
		if service.Kind != yaml.MappingNode {
			continue
		}
		setScalarValue(service, "restart", "no")
		ensureScalarValue(service, "mem_limit", "512m")
		ensureScalarValue(service, "cpus", "1.0")
		ensureScalarValue(service, "pids_limit", "256")
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return "", err
	}
	_ = enc.Close()
	return buf.String(), nil
}

func nodeValue(mapping *yaml.Node, key string) *yaml.Node {
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

func ensureSequenceValue(mapping *yaml.Node, key, value string) {
	existing := nodeValue(mapping, key)
	if existing == nil {
		mapping.Content = append(mapping.Content, keyNode(key), &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{valueNode(value)}})
		return
	}
	if existing.Kind != yaml.SequenceNode {
		return
	}
	for _, item := range existing.Content {
		if strings.EqualFold(strings.TrimSpace(item.Value), value) {
			return
		}
	}
	existing.Content = append(existing.Content, valueNode(value))
}

func ensureScalarValue(mapping *yaml.Node, key, value string) {
	if nodeValue(mapping, key) == nil {
		mapping.Content = append(mapping.Content, keyNode(key), valueNode(value))
	}
}

func setScalarValue(mapping *yaml.Node, key, value string) {
	existing := nodeValue(mapping, key)
	if existing == nil {
		mapping.Content = append(mapping.Content, keyNode(key), valueNode(value))
		return
	}
	if existing.Kind == yaml.ScalarNode {
		existing.Tag = "!!str"
		existing.Value = value
	}
}

func keyNode(k string) *yaml.Node   { return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k} }
func valueNode(v string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v} }

func (c *Compose) writeEnvFile(ctx context.Context, dir, sourceDir string, app *domain.ComposeApp) error {
	merged := map[string]string{}
	if data, err := os.ReadFile(filepath.Join(sourceDir, ".env")); err == nil {
		for key, value := range parseEnvFile(string(data)) {
			merged[key] = value
		}
	}
	if c.Variables != nil {
		variables, err := c.effectiveVariables(ctx, app)
		if err != nil {
			return err
		}
		for key, value := range variables {
			merged[key] = value
		}
		if app.Port > 0 && strings.TrimSpace(merged["PORT"]) == "" {
			merged["PORT"] = strconv.Itoa(app.Port)
		}
		return writeEnvValues(filepath.Join(dir, ".env"), merged)
	}
	if c.ProjectVars == nil {
		if app.Port > 0 && strings.TrimSpace(merged["PORT"]) == "" {
			merged["PORT"] = strconv.Itoa(app.Port)
		}
		return writeEnvValues(filepath.Join(dir, ".env"), merged)
	}
	project, err := c.ProjectVars.ListVariables(ctx, app.ProjectID, uuid.Nil)
	if err != nil {
		return err
	}
	for _, v := range project {
		merged[v.Key] = v.Value
	}
	if app.EnvironmentID != nil {
		env, err := c.ProjectVars.ListVariables(ctx, app.ProjectID, *app.EnvironmentID)
		if err != nil {
			return err
		}
		for _, v := range env {
			merged[v.Key] = v.Value
		}
	}
	if app.Port > 0 && strings.TrimSpace(merged["PORT"]) == "" {
		merged["PORT"] = strconv.Itoa(app.Port)
	}
	if len(merged) == 0 {
		return nil
	}
	return writeEnvValues(filepath.Join(dir, ".env"), merged)
}

func (c *Compose) effectiveVariables(ctx context.Context, app *domain.ComposeApp) (map[string]string, error) {
	if c.Variables == nil {
		return map[string]string{}, nil
	}
	serviceID, err := c.GetServiceID(ctx, app.ID)
	if err != nil {
		return nil, err
	}
	return c.Variables.Effective(ctx, serviceID, app.OrgID)
}

func injectComposeEnvironment(content string, variables map[string]string) (string, error) {
	if len(variables) == 0 {
		return content, nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return "", err
	}
	root := &doc
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	if root.Kind != yaml.MappingNode {
		return "", fmt.Errorf("compose root is not a mapping")
	}
	var services *yaml.Node
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "services" {
			services = root.Content[i+1]
			break
		}
	}
	if services == nil || services.Kind != yaml.MappingNode {
		return "", fmt.Errorf("compose has no services mapping")
	}
	for i := 0; i+1 < len(services.Content); i += 2 {
		svc := services.Content[i+1]
		if svc.Kind == yaml.MappingNode {
			services.Content[i+1] = injectServiceEnvironment(svc, variables)
		}
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return "", err
	}
	_ = enc.Close()
	return buf.String(), nil
}

func interpolateComposeVariables(content string, variables map[string]string) (string, error) {
	if len(variables) == 0 {
		return content, nil
	}
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", err
	}
	var replace func(*yaml.Node)
	replace = func(node *yaml.Node) {
		if node.Kind == yaml.ScalarNode {
			node.Value = replaceComposeVariableReferences(node.Value, variables)
			return
		}
		for _, child := range node.Content {
			replace(child)
		}
	}
	replace(&document)
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&document); err != nil {
		return "", err
	}
	_ = enc.Close()
	return buf.String(), nil
}

func replaceComposeVariableReferences(value string, variables map[string]string) string {
	var result strings.Builder
	for position := 0; position < len(value); {
		start := strings.Index(value[position:], "${")
		if start < 0 {
			result.WriteString(value[position:])
			break
		}
		start += position
		result.WriteString(value[position:start])
		end := strings.IndexByte(value[start+2:], '}')
		if end < 0 {
			result.WriteString(value[start:])
			break
		}
		end += start + 2
		token := value[start+2 : end]
		key := token
		if defaultValue, _, ok := strings.Cut(token, ":-"); ok {
			key = defaultValue
		}
		if replacement, ok := variables[key]; ok {
			result.WriteString(replacement)
		} else {
			result.WriteString(value[start : end+1])
		}
		position = end + 1
	}
	return result.String()
}

func injectServiceEnvironment(svc *yaml.Node, variables map[string]string) *yaml.Node {
	keys := make([]string, 0, len(variables))
	for key := range variables {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for i := 0; i+1 < len(svc.Content); i += 2 {
		if svc.Content[i].Value != "environment" {
			continue
		}
		environment := svc.Content[i+1]
		switch environment.Kind {
		case yaml.MappingNode:
			existing := make(map[string]struct{}, len(environment.Content)/2)
			for j := 0; j+1 < len(environment.Content); j += 2 {
				key := environment.Content[j].Value
				existing[key] = struct{}{}
				if value, ok := variables[key]; ok && strings.TrimSpace(environment.Content[j+1].Value) == "" {
					environment.Content[j+1] = valueNode(value)
				}
			}
			for _, key := range keys {
				if _, ok := existing[key]; !ok {
					environment.Content = append(environment.Content, keyNode(key), valueNode(variables[key]))
				}
			}
		case yaml.SequenceNode:
			existing := make(map[string]struct{}, len(environment.Content))
			for _, item := range environment.Content {
				key, explicitValue, found := strings.Cut(item.Value, "=")
				if !found {
					key = item.Value
				}
				existing[key] = struct{}{}
				if value, ok := variables[key]; ok && (!found || strings.TrimSpace(explicitValue) == "") {
					item.Value = key + "=" + value
				}
			}
			for _, key := range keys {
				if _, ok := existing[key]; !ok {
					environment.Content = append(environment.Content, valueNode(key+"="+variables[key]))
				}
			}
		}
		return svc
	}
	environment := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, key := range keys {
		environment.Content = append(environment.Content, keyNode(key), valueNode(variables[key]))
	}
	svc.Content = append(svc.Content, keyNode("environment"), environment)
	return svc
}

func parseEnvFile(content string) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) == "" {
			continue
		}
		values[strings.TrimSpace(key)] = value
	}
	return values
}

func writeEnvValues(path string, merged map[string]string) error {
	if len(merged) == 0 {
		return nil
	}
	var sb strings.Builder
	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		sb.WriteString(k + "=" + merged[k] + "\n")
	}
	return os.WriteFile(path, []byte(sb.String()), 0o600)
}

func validYAML(content string) bool {
	var parsed any
	return yaml.Unmarshal([]byte(content), &parsed) == nil
}
