package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesystemRedactsACMESecrets(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "acme"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "acme", "acme.json"), []byte(`{"letsencrypt":{"Account":{"PrivateKey":"secret"},"Certificates":[{"certificate":"certificate","key":"private"}]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := (&Filesystem{Root: root}).Read("acme/acme.json")
	if err != nil {
		t.Fatal(err)
	}
	if !file.Redacted || file.Editable || strings.Contains(file.Content, "secret") || strings.Contains(file.Content, "private") {
		t.Fatalf("unexpected ACME content: %+v", file)
	}
}

func TestFilesystemRejectsEscape(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "dynamic"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := (&Filesystem{Root: root}).Read("../outside"); err != ErrInvalidPath {
		t.Fatalf("expected invalid path, got %v", err)
	}
}

func TestFilesystemValidatesAndWritesConfig(t *testing.T) {
	root := t.TempDir()
	filesystem := &Filesystem{Root: root}
	if err := filesystem.Write("dynamic/domain.yml", "http:\n  routers: {}\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "dynamic", "domain.yml")); err != nil {
		t.Fatal(err)
	}
	if err := filesystem.Write("dynamic/domain.yml", "http: ["); err == nil {
		t.Fatal("expected invalid YAML")
	}
	if err := filesystem.Delete("traefik.yml"); err != ErrFileNotEditable {
		t.Fatalf("expected protected static config, got %v", err)
	}
}
