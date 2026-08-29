package application

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	appsdomain "aether/internal/modules/apps/domain"
	realtimedomain "aether/internal/modules/realtime/domain"
	sourcedomain "aether/internal/modules/sourcecontrol/domain"
	"aether/internal/modules/templates/domain"
	variablesDomain "aether/internal/modules/variables/domain"
)

type Compose struct {
	Store           domain.Store
	Apps            AppStore
	Deployments     DeploymentStore
	ServiceIdentity func(context.Context, uuid.UUID) (uuid.UUID, error)
	DataDir         string
	ProjectVars     ProjectVarStore
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

type AppStore interface {
	GetProject(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.Project, error)
	GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error)
	GetEnvironment(ctx context.Context, id, projectID uuid.UUID) (*appsdomain.Environment, error)
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
	return c.Store.CreateComposeApp(ctx, &domain.ComposeApp{
		OrgID: orgID, ProjectID: projectID, EnvironmentID: environmentID, Name: name, Compose: content, Status: "pending",
	})
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
	if err := c.runCompose(ctx, app, "up", "-d"); err != nil {
		_ = c.Store.SetComposeStatus(ctx, id, "error")
		return err
	}
	if err := c.Store.SetComposeStatus(ctx, id, "running"); err != nil {
		return err
	}
	c.recordEvent(ctx, app, "compose.running")
	return nil
}

func (c *Compose) Down(ctx context.Context, id, orgID uuid.UUID) error {
	app, err := c.Get(ctx, id, orgID)
	if err != nil {
		return err
	}
	if err := c.Store.SetComposeStatus(ctx, id, "deploying"); err != nil {
		return err
	}
	if err := c.runCompose(ctx, app, "down"); err != nil {
		_ = c.Store.SetComposeStatus(ctx, id, "error")
		return err
	}
	if err := c.Store.SetComposeStatus(ctx, id, "stopped"); err != nil {
		return err
	}
	c.recordEvent(ctx, app, "compose.stopped")
	return nil
}

func (c *Compose) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	app, err := c.Get(ctx, id, orgID)
	if err != nil {
		return err
	}
	if err := c.runCompose(ctx, app, "down"); err != nil {
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
	if _, err := c.Get(ctx, id, orgID); err != nil {
		return "", err
	}
	serviceID, err := c.GetServiceID(ctx, id)
	if err != nil {
		serviceID = id
	}
	values := []uuid.UUID{serviceID}
	if id != serviceID {
		values = append(values, id)
	}
	for _, value := range values {
		out, queryErr := exec.CommandContext(ctx, "podman", "ps", "-q", "--filter", "label=aether.service-id="+value.String(), "--filter", "status=running").Output()
		if queryErr != nil {
			return "", fmt.Errorf("resolve compose container: %w", queryErr)
		}
		containerID := strings.TrimSpace(string(out))
		if containerID != "" {
			return strings.Split(containerID, "\n")[0], nil
		}
	}
	return "", errors.New("no active container")
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
		"image":   app.Image,
		"restart": "unless-stopped",
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

func (c *Compose) runCompose(ctx context.Context, app *domain.ComposeApp, args ...string) error {
	if c.DataDir == "" {
		return fmt.Errorf("data dir not configured")
	}
	dir := filepath.Join(c.DataDir, "compose", app.ID.String())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	workDir := dir
	file := filepath.Join(dir, "docker-compose.yml")
	content := app.Compose
	if c.Source != nil && c.Clone != nil && len(args) > 0 && args[0] == "up" {
		serviceID, err := c.GetServiceID(ctx, app.ID)
		if err != nil {
			return err
		}
		source, err := c.Source.GetByService(ctx, serviceID, app.OrgID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if source != nil {
			if c.Clone == nil {
				return errors.New("git source deployment is not configured")
			}
			checkoutPath := filepath.Join(dir, "checkout")
			if err := os.RemoveAll(checkoutPath); err != nil {
				return err
			}
			checkout, err := c.Clone.Clone(ctx, source, checkoutPath)
			if err != nil {
				return err
			}
			root := source.RootDirectory
			if root == "" || root == "." {
				root = ""
			}
			workDir = filepath.Join(checkout, root)
			composeFile := source.ComposeFile
			if composeFile == "" {
				composeFile = "docker-compose.yml"
			}
			file = filepath.Join(workDir, composeFile)
			if !pathWithin(checkout, file) {
				return errors.New("compose file escapes repository checkout")
			}
			workDir = filepath.Dir(file)
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("read compose file from checkout: %w", err)
			}
			content = string(data)
		}
	}
	if len(args) > 0 && args[0] == "up" {
		serviceID, err := c.GetServiceID(ctx, app.ID)
		if err != nil {
			serviceID = app.ID
		}
		if injected, err := injectComposeLabels(content, map[string]string{
			"aether.owner":        "user",
			"aether.service-type": "compose",
			"aether.service-id":   serviceID.String(),
			"aether.spec-id":      app.ID.String(),
			"aether.project-id":   app.ProjectID.String(),
			"aether.service-name": app.Name,
		}); err == nil {
			content = injected
		}
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		return err
	}
	if c.ProjectVars != nil {
		c.writeEnvFile(ctx, workDir, app)
	}
	c.prepareConfig(ctx, workDir, content)
	cmdArgs := append([]string{"compose", "-f", file}, args...)
	cmd := exec.CommandContext(ctx, "podman", cmdArgs...)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
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
				existing.Content = append(existing.Content, keyNode(k), valueNode(v))
			}
		case yaml.SequenceNode:
			for k, v := range labels {
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

func keyNode(k string) *yaml.Node   { return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k} }
func valueNode(v string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v} }

func (c *Compose) writeEnvFile(ctx context.Context, dir string, app *domain.ComposeApp) {
	merged := map[string]string{}
	project, err := c.ProjectVars.ListVariables(ctx, app.ProjectID, uuid.Nil)
	if err != nil {
		return
	}
	for _, v := range project {
		merged[v.Key] = v.Value
	}
	if app.EnvironmentID != nil {
		env, err := c.ProjectVars.ListVariables(ctx, app.ProjectID, *app.EnvironmentID)
		if err == nil {
			for _, v := range env {
				merged[v.Key] = v.Value
			}
		}
	}
	if len(merged) == 0 {
		return
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
	_ = os.WriteFile(filepath.Join(dir, ".env"), []byte(sb.String()), 0o600)
}

func (c *Compose) prepareConfig(ctx context.Context, dir, compose string) {
	if !strings.Contains(compose, "/root/.affine/config") {
		return
	}
	cfg := `{
  "$schema": "https://github.com/toeverything/affine/releases/latest/download/config.schema.json",
  "server": {
    "name": "AFFiNE Self-hosted",
    "externalUrl": "http://localhost:3010"
  },
  "copilot": { "enabled": true, "byok": { "enabled": true } }
}
`
	cmd := exec.CommandContext(ctx, "podman", "run", "--rm",
		"-v", "aether-affine-config:/cfg",
		"docker.io/library/alpine:3.20",
		"sh", "-c", "cat > /cfg/config.json")
	cmd.Stdin = strings.NewReader(cfg)
	_ = cmd.Run()
}

func validYAML(content string) bool {
	var parsed any
	return yaml.Unmarshal([]byte(content), &parsed) == nil
}
