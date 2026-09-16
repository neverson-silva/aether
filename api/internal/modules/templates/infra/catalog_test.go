package infra

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

func TestBundledDokployCatalogIsAvailableOffline(t *testing.T) {
	metadata, err := EnsureDokployCatalog(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata) == 0 {
		t.Fatal("bundled catalog is empty")
	}
	catalog := NewDokployCatalog(t.TempDir())
	var item DokployTemplateMetadata
	for _, candidate := range metadata {
		if safeRemoteID(candidate.ID) {
			item = candidate
			break
		}
	}
	if item.ID == "" {
		t.Fatal("bundled catalog has no safe template")
	}
	template, err := catalog.Get(context.Background(), remoteTemplateID(item.ID))
	if err != nil {
		t.Fatal(err)
	}
	if template.ComposeYAML == "" {
		t.Fatal("bundled template has no Compose definition")
	}
	if _, err := uuid.Parse(template.ID.String()); err != nil {
		t.Fatal(err)
	}
}

func TestBundledDragonflyComposeAvoidsUnsupportedLimits(t *testing.T) {
	catalog := NewDokployCatalog(t.TempDir())
	metadata, err := EnsureDokployCatalog(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range metadata {
		if item.ID != "dragonfly-db" {
			continue
		}
		compose, err := catalog.ensureBlueprint(context.Background(), item.ID)
		if err != nil {
			t.Fatalf("%s Compose: %v", item.ID, err)
		}
		if strings.Contains(compose, "ulimits:") {
			t.Errorf("%s Compose contains unsupported ulimits", item.ID)
		}
	}
}

func TestBundledRustFSComposeUsesExplicitNamedVolumeDefinitions(t *testing.T) {
	catalog := NewDokployCatalog(t.TempDir())
	template, err := catalog.Get(context.Background(), remoteTemplateID("rustfs"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(template.ComposeYAML, "rustfs-data:\n") {
		t.Fatalf("RustFS volume definition was not normalized: %s", template.ComposeYAML)
	}
}

func TestBundledRustFSComposeExposesEnvironmentVariables(t *testing.T) {
	catalog := NewDokployCatalog(t.TempDir())
	template, err := catalog.Get(context.Background(), remoteTemplateID("rustfs"))
	if err != nil {
		t.Fatal(err)
	}
	if len(template.Environment) != 2 {
		t.Fatalf("RustFS template environment variables = %d, want 2", len(template.Environment))
	}
	if template.Environment[0].Name != "RUSTFS_ACCESS_KEY" || template.Environment[0].Value != "${access_key}" {
		t.Fatalf("unexpected first RustFS environment variable: %+v", template.Environment[0])
	}
}

func TestBundledRustFSTemplateExposesDokployDomains(t *testing.T) {
	catalog := NewDokployCatalog(t.TempDir())
	template, err := catalog.Get(context.Background(), remoteTemplateID("rustfs"))
	if err != nil {
		t.Fatal(err)
	}
	if len(template.Domains) != 2 {
		t.Fatalf("RustFS template domains = %d, want 2", len(template.Domains))
	}
	if template.Domains[0].ServiceName != "rustfs" || template.Domains[0].Port != 9001 {
		t.Fatalf("unexpected RustFS console domain: %+v", template.Domains[0])
	}
	if template.Domains[1].ServiceName != "rustfs" || template.Domains[1].Port != 9000 {
		t.Fatalf("unexpected RustFS API domain: %+v", template.Domains[1])
	}
}

func TestBundledDokployTemplateConfigsMatchComposeServices(t *testing.T) {
	catalog := NewDokployCatalog(t.TempDir())
	metadata, err := EnsureDokployCatalog(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, item := range metadata {
		if !safeRemoteID(item.ID) {
			continue
		}
		compose, err := catalog.ensureBlueprint(context.Background(), item.ID)
		if err != nil {
			t.Fatalf("%s Compose: %v", item.ID, err)
		}
		var document struct {
			Services map[string]any `yaml:"services"`
		}
		if err := yaml.Unmarshal([]byte(compose), &document); err != nil {
			t.Fatalf("%s Compose: %v", item.ID, err)
		}
		variables, environment, mounts, domains := catalog.templateConfig(item.ID)
		_ = variables
		_ = environment
		for _, mount := range mounts {
			if mount.ServiceName != "" && document.Services[mount.ServiceName] == nil {
				t.Fatalf("%s mount references missing service %s", item.ID, mount.ServiceName)
			}
		}
		for _, domain := range domains {
			if document.Services[domain.ServiceName] == nil {
				t.Fatalf("%s domain references missing service %s", item.ID, domain.ServiceName)
			}
		}
		checked++
	}
	if checked != len(metadata) {
		t.Fatalf("checked templates = %d, want %d", checked, len(metadata))
	}
}
