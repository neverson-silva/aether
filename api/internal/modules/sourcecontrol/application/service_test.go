package application

import (
	"strings"
	"testing"
)

func TestParseEnvironmentKeys(t *testing.T) {
	keys := parseEnvironmentKeys(strings.Join([]string{
		"DATABASE_URL=postgres://localhost/app",
		"REDIS_URL=",
		"JWT_SECRET=\"example-secret\"",
		"PORT=3000",
		"",
		"# COMMENTED_VAR=value",
		"NODE_ENV = development",
		"INVALID",
		"DATABASE_URL=duplicate",
	}, "\n"))
	want := []string{"DATABASE_URL", "REDIS_URL", "JWT_SECRET", "PORT", "NODE_ENV"}
	if strings.Join(keys, ",") != strings.Join(want, ",") {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
}

func TestServiceTemplatePathRespectsRootAndRejectsTraversal(t *testing.T) {
	got, err := serviceTemplatePath("apps/api", ".env.example")
	if err != nil || got != "apps/api/.env.example" {
		t.Fatalf("path = %q, err = %v", got, err)
	}
	if _, err := serviceTemplatePath("apps/api", "../../etc/passwd"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}
