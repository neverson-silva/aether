package infra

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"aether/internal/modules/templates/domain"
	composeengine "aether/internal/platform/compose"
)

//go:embed catalogdata/meta.json
var bundledDokployMetadata []byte

//go:embed catalogdata/blueprints/*
var bundledDokployAssets embed.FS

type DokployTemplateMetadata struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Logo        string            `json:"logo"`
	Links       map[string]string `json:"links"`
	Tags        []string          `json:"tags"`
}

type DokployCatalog struct {
	StateDir string
}

func NewDokployCatalog(stateDir string) *DokployCatalog {
	return &DokployCatalog{StateDir: stateDir}
}

func (c *DokployCatalog) List(ctx context.Context) ([]domain.Template, error) {
	metadata, err := EnsureDokployCatalog(ctx, c.StateDir)
	if err != nil {
		return nil, err
	}
	templates := make([]domain.Template, 0, len(metadata))
	for _, item := range metadata {
		if !safeRemoteID(item.ID) {
			continue
		}
		template := remoteTemplate(item)
		if compose, composeErr := c.ensureBlueprint(ctx, item.ID); composeErr == nil {
			template.Environment = composeEnvironmentVariables(compose)
		}
		templates = append(templates, template)
	}
	return templates, nil
}

func (c *DokployCatalog) Get(ctx context.Context, id uuid.UUID) (*domain.Template, error) {
	metadata, err := EnsureDokployCatalog(ctx, c.StateDir)
	if err != nil {
		return nil, err
	}
	for _, item := range metadata {
		if remoteTemplateID(item.ID) != id {
			continue
		}
		compose, err := c.ensureBlueprint(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		template := remoteTemplate(item)
		template.ComposeYAML = compose
		template.Environment = composeEnvironmentVariables(compose)
		return &template, nil
	}
	return nil, domain.ErrNotFound
}

func composeEnvironmentVariables(content string) []domain.TemplateEnvironmentVariable {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return nil
	}
	root := documentRoot(&document)
	if root == nil || root.Kind != yaml.MappingNode {
		return nil
	}
	services := mappingValue(root, "services")
	if services == nil || services.Kind != yaml.MappingNode {
		return nil
	}
	seen := map[string]struct{}{}
	variables := make([]domain.TemplateEnvironmentVariable, 0)
	for i := 0; i+1 < len(services.Content); i += 2 {
		environment := mappingValue(services.Content[i+1], "environment")
		if environment == nil {
			continue
		}
		appendEnvironmentVariables(&variables, seen, environment)
	}
	return variables
}

func appendEnvironmentVariables(variables *[]domain.TemplateEnvironmentVariable, seen map[string]struct{}, environment *yaml.Node) {
	switch environment.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(environment.Content); i += 2 {
			name := strings.TrimSpace(environment.Content[i].Value)
			if name == "" {
				continue
			}
			value := environment.Content[i+1].Value
			if strings.Contains(value, "${") {
				value = ""
			}
			appendEnvironmentVariable(variables, seen, name, value)
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
			appendEnvironmentVariable(variables, seen, strings.TrimSpace(name), value)
		}
	}
}

func appendEnvironmentVariable(variables *[]domain.TemplateEnvironmentVariable, seen map[string]struct{}, name, value string) {
	if name == "" {
		return
	}
	if _, exists := seen[name]; exists {
		return
	}
	seen[name] = struct{}{}
	*variables = append(*variables, domain.TemplateEnvironmentVariable{Name: name, Value: value})
}

func documentRoot(document *yaml.Node) *yaml.Node {
	if document.Kind == yaml.DocumentNode && len(document.Content) > 0 {
		return document.Content[0]
	}
	return document
}

func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
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

func (c *DokployCatalog) Logo(ctx context.Context, id uuid.UUID) ([]byte, string, error) {
	metadata, err := EnsureDokployCatalog(ctx, c.StateDir)
	if err != nil {
		return nil, "", err
	}
	for _, item := range metadata {
		if remoteTemplateID(item.ID) != id || !safeRemoteID(item.ID) || !safeAssetName(item.Logo) {
			continue
		}
		if data, readErr := bundledDokployAssets.ReadFile("catalogdata/blueprints/" + item.ID + "-logo"); readErr == nil {
			return data, logoContentType(item.Logo), nil
		}
		return nil, "", fmt.Errorf("Dokploy logo is not available in the bundled catalog")
	}
	return nil, "", domain.ErrNotFound
}

func safeAssetName(name string) bool {
	return name != "" && filepath.Base(name) == name && !strings.Contains(name, "\\")
}

func safeRemoteID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, character := range id {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '-' || character == '_' || character == '.' {
			continue
		}
		return false
	}
	return true
}

func logoContentType(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".svg":
		return "image/svg+xml"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	default:
		return "image/png"
	}
}

func remoteTemplateID(id string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("dokploy:"+id))
}

func remoteTemplate(item DokployTemplateMetadata) domain.Template {
	category := ""
	if len(item.Tags) > 0 {
		category = item.Tags[0]
	}
	links := item.Links
	return domain.Template{ID: remoteTemplateID(item.ID), RemoteID: item.ID, Name: item.Name, Description: item.Description, Category: category, Version: item.Version, Icon: item.Logo, GitHub: links["github"], Homepage: links["website"], Tags: item.Tags, Verified: true, UpdatedAt: time.Now().UTC()}
}

func (c *DokployCatalog) ensureBlueprint(ctx context.Context, id string) (string, error) {
	if !safeRemoteID(id) {
		return "", fmt.Errorf("fetch Dokploy blueprint: invalid template ID")
	}
	if data, err := bundledDokployAssets.ReadFile("catalogdata/blueprints/" + id + ".docker-compose.yml"); err == nil && strings.TrimSpace(string(data)) != "" {
		return composeengine.NormalizeNamedResourceDefinitions(string(data))
	}
	_ = ctx
	return "", fmt.Errorf("Dokploy Compose is not available in the bundled catalog")
}

func EnsureDokployCatalog(ctx context.Context, stateDir string) ([]DokployTemplateMetadata, error) {
	cachePath := filepath.Join(stateDir, "templates", "dokploy-meta.json")
	var metadata []DokployTemplateMetadata
	if err := json.Unmarshal(bundledDokployMetadata, &metadata); err != nil {
		return nil, fmt.Errorf("decode bundled Dokploy catalog: %w", err)
	}
	if len(metadata) == 0 {
		return nil, fmt.Errorf("decode Dokploy catalog: empty catalog")
	}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode Dokploy catalog: %w", err)
	}
	directory := filepath.Dir(cachePath)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create Dokploy catalog directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".dokploy-meta-*")
	if err != nil {
		return nil, fmt.Errorf("create Dokploy catalog cache: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return nil, fmt.Errorf("protect Dokploy catalog cache: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return nil, fmt.Errorf("write Dokploy catalog cache: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return nil, fmt.Errorf("close Dokploy catalog cache: %w", err)
	}
	if err := os.Rename(temporaryName, cachePath); err != nil {
		return nil, fmt.Errorf("commit Dokploy catalog cache: %w", err)
	}
	_ = ctx
	return metadata, nil
}

func writeAtomicCache(cachePath string, data []byte) error {
	directory := filepath.Dir(cachePath)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create template cache directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".template-cache-*")
	if err != nil {
		return fmt.Errorf("create template cache: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("protect template cache: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write template cache: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close template cache: %w", err)
	}
	if err := os.Rename(temporaryName, cachePath); err != nil {
		return fmt.Errorf("commit template cache: %w", err)
	}
	return nil
}
