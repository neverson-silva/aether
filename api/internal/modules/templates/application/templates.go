package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"aether/internal/modules/templates/domain"
)

type Templates struct {
	Store domain.Store
	Apps  AppStore
}

func (t *Templates) List(ctx context.Context, filter domain.Filter) ([]domain.Template, error) {
	return t.Store.ListTemplates(ctx, filter)
}

func (t *Templates) Install(ctx context.Context, templateID, orgID, projectID uuid.UUID, name string, overrides map[string]string) (*domain.Template, error) {
	tpl, err := t.Store.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if _, err := t.Apps.GetProject(ctx, projectID, orgID); err != nil {
		return nil, err
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
	if _, err := t.Store.CreateComposeApp(ctx, &domain.ComposeApp{
		OrgID: orgID, ProjectID: projectID, Name: appName, Compose: compose, Status: "stopped",
	}); err != nil {
		return nil, err
	}
	if err := t.Store.IncrementInstalls(ctx, templateID); err != nil {
		return nil, err
	}
	tpl.Installs++
	tpl.ComposeYAML = compose
	return tpl, nil
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
			"image":   image,
			"restart": "unless-stopped",
		}
		if svc.Port > 0 {
			entry["ports"] = []string{fmt.Sprintf("%d:%d", svc.Port, svc.Port)}
		}
		if len(svc.Env) > 0 {
			entry["environment"] = svc.Env
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
