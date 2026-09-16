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
	if err := composeengine.ValidatePolicy(content); err != nil {
		return nil, domain.ErrValidation
	}
	port, hasPort, err := composePublishedPort(content)
	if err != nil {
		return nil, domain.ErrValidation
	}
	if !hasPort {
		port, err = composeContainerPort(content)
		if err != nil {
			return nil, fmt.Errorf("parse compose container port: %w", err)
		}
		if port == 0 {
			port, err = c.Store.NextComposePort(ctx)
			if err != nil {
				return nil, fmt.Errorf("allocate compose port: %w", err)
			}
			content, err = addComposePort(content, port)
			if err != nil {
				return nil, fmt.Errorf("add compose port: %w", err)
			}
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

func PublishedPort(content string) (int, bool, error) {
	return composePublishedPort(content)
}

func addComposePort(content string, port int) (string, error) {
	containerPort, err := composeContainerPort(content)
	if err != nil {
		return "", err
	}
	if containerPort == 0 {
		containerPort = port
	}
	return addComposePortMapping(content, port, containerPort)
}

func addComposePortMapping(content string, hostPort, containerPort int) (string, error) {
	var document map[string]any
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", err
	}
	services, ok := document["services"].(map[string]any)
	if !ok || len(services) == 0 {
		return "", errors.New("compose has no services mapping")
	}
	for _, raw := range services {
		service, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		service["ports"] = []string{fmt.Sprintf("%d:%d", hostPort, containerPort)}
		break
	}
	encoded, err := yaml.Marshal(document)
	return string(encoded), err
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

func parseComposePort(value string) int {
	value = strings.TrimSpace(value)
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

func repairComposePortMapping(content string, publishedPort int) (string, bool, error) {
	if publishedPort <= 0 {
		return content, false, nil
	}
	containerPort, err := composeContainerPort(content)
	if err != nil {
		return "", false, err
	}
	if containerPort <= 0 || containerPort == publishedPort {
		return content, false, nil
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", false, err
	}
	services, ok := document["services"].(map[string]any)
	if !ok {
		return content, false, nil
	}
	for _, raw := range services {
		service, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ports, ok := service["ports"].([]any)
		if !ok {
			continue
		}
		for index, rawPort := range ports {
			switch value := rawPort.(type) {
			case string:
				parts := strings.Split(value, ":")
				if len(parts) < 2 || parseComposePort(parts[len(parts)-2]) != publishedPort || parseComposePort(parts[len(parts)-1]) != publishedPort {
					continue
				}
				target := parts[len(parts)-1]
				suffix := ""
				if slash := strings.IndexByte(target, '/'); slash >= 0 {
					suffix = target[slash:]
				}
				parts[len(parts)-1] = strconv.Itoa(containerPort) + suffix
				ports[index] = strings.Join(parts, ":")
				encoded, err := yaml.Marshal(document)
				return string(encoded), true, err
			case map[string]any:
				published, publishedOK := value["published"]
				target, targetOK := value["target"]
				if !publishedOK || !targetOK || composePortValue(published) != publishedPort || composePortValue(target) != publishedPort {
					continue
				}
				value["target"] = containerPort
				encoded, err := yaml.Marshal(document)
				return string(encoded), true, err
			}
		}
	}
	return content, false, nil
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
	if _, err := c.runCompose(ctx, app, true, "up", "-d"); err != nil {
		_ = c.Store.SetComposeStatus(ctx, id, "error")
		return err
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
	if _, err := c.runComposeForService(ctx, composeApp, true, "app", "up", "-d"); err != nil {
		return "", err
	}
	serviceID, err := c.GetServiceID(ctx, id)
	if err != nil {
		serviceID = id
	}
	if c.Runtime == nil {
		return "", errors.New("compose runtime unavailable")
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
	if _, err := c.runCompose(ctx, app, false, "down"); err != nil {
		return err
	}
	return c.Store.DeleteComposeApp(ctx, id, orgID)
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
	if len(args) > 0 && args[0] == "up" {
		repaired, _, err := repairComposePortMapping(content, app.Port)
		if err != nil {
			return "", fmt.Errorf("repair compose port mapping: %w", err)
		}
		content = repaired
	}
	if err := composeengine.ValidatePolicy(content); err != nil {
		return "", err
	}
	if len(args) > 0 && args[0] == "up" {
		publishedPort, hasPort, err := composePublishedPort(content)
		if err != nil {
			return "", fmt.Errorf("parse compose ports: %w", err)
		}
		if !hasPort {
			containerPort, portErr := composeContainerPort(content)
			if portErr != nil {
				return "", fmt.Errorf("parse compose container port: %w", portErr)
			}
			if containerPort == 0 && c.Store != nil {
				publishedPort, err = c.Store.NextComposePort(ctx)
				if err != nil {
					return "", fmt.Errorf("allocate compose port: %w", err)
				}
				content, err = addComposePort(content, publishedPort)
				if err != nil {
					return "", fmt.Errorf("add compose port: %w", err)
				}
			} else if app.Port == 0 {
				publishedPort = containerPort
			}
		}
		if updater, ok := c.Apps.(AppPortUpdater); ok && publishedPort > 0 && app.Port != publishedPort {
			if err := updater.UpdateAppPort(ctx, app.ID, publishedPort); err != nil {
				return "", fmt.Errorf("persist compose port: %w", err)
			}
			app.Port = publishedPort
		}
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
		injected, err = injectComposeEnvironment(injected, variables)
		if err != nil {
			return "", fmt.Errorf("inject compose environment: %w", err)
		}
		injected, err = injectComposeSecurityDefaults(injected)
		if err != nil {
			return "", fmt.Errorf("inject compose security defaults: %w", err)
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
	envFile := filepath.Join(dir, ".env")
	if err := c.writeEnvFile(ctx, dir, workDir, app); err != nil {
		return "", err
	}
	if len(args) == 0 || args[0] != "up" {
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
	for i := 0; i+1 < len(services.Content); i += 2 {
		service := services.Content[i+1]
		if service.Kind != yaml.MappingNode {
			continue
		}
		ensureSequenceValue(service, "cap_drop", "ALL")
		ensureSequenceValue(service, "security_opt", "no-new-privileges:true")
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
		return writeEnvValues(filepath.Join(dir, ".env"), merged)
	}
	if c.ProjectVars == nil {
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

func injectServiceEnvironment(svc *yaml.Node, variables map[string]string) *yaml.Node {
	for i := 0; i+1 < len(svc.Content); i += 2 {
		if svc.Content[i].Value != "environment" {
			continue
		}
		environment := svc.Content[i+1]
		switch environment.Kind {
		case yaml.MappingNode:
			for key, value := range variables {
				updated := false
				for j := 0; j+1 < len(environment.Content); j += 2 {
					if environment.Content[j].Value == key {
						environment.Content[j+1] = valueNode(value)
						updated = true
						break
					}
				}
				if !updated {
					environment.Content = append(environment.Content, keyNode(key), valueNode(value))
				}
			}
		case yaml.SequenceNode:
			for key, value := range variables {
				found := false
				for _, item := range environment.Content {
					if item.Value == key || strings.HasPrefix(item.Value, key+"=") {
						item.Value = key + "=" + value
						found = true
						break
					}
				}
				if !found {
					environment.Content = append(environment.Content, valueNode(key+"="+value))
				}
			}
		}
		return svc
	}
	environment := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for key, value := range variables {
		environment.Content = append(environment.Content, keyNode(key), valueNode(value))
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
