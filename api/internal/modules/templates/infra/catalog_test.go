package infra

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
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
	if len(template.Environment) != 7 {
		t.Fatalf("RustFS environment variables = %d, want 7", len(template.Environment))
	}
	if template.Environment[0].Name != "RUSTFS_ACCESS_KEY" || template.Environment[0].Value != "" {
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
