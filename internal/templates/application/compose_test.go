package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"aether/internal/templates/domain"
	variablesDomain "aether/internal/variables/domain"
)

type fakeVarStore struct {
	vars []variablesDomain.Variable
}

func (f *fakeVarStore) ListVariables(ctx context.Context, projectID, environmentID uuid.UUID) ([]variablesDomain.Variable, error) {
	var out []variablesDomain.Variable
	for _, v := range f.vars {
		if v.EnvironmentID == environmentID {
			out = append(out, v)
		}
	}
	return out, nil
}

type fakeVarMulti struct {
	project ProjectVarStore
	env     ProjectVarStore
}

func (f *fakeVarMulti) ListVariables(ctx context.Context, projectID, environmentID uuid.UUID) ([]variablesDomain.Variable, error) {
	if environmentID == uuid.Nil {
		return f.project.ListVariables(ctx, projectID, environmentID)
	}
	return f.env.ListVariables(ctx, projectID, environmentID)
}

func TestComposeWriteEnvFile(t *testing.T) {
	dir := t.TempDir()
	projID := uuid.New()
	c := &Compose{
		DataDir: dir,
		ProjectVars: &fakeVarStore{vars: []variablesDomain.Variable{
			{ProjectID: projID, Key: "REGION", Value: "sa-east-1"},
			{ProjectID: projID, Key: "DB_PASS", Value: "topsecret", IsSecret: true},
		}},
	}
	composeDir := filepath.Join(dir, "compose", "stack-1")
	_ = os.MkdirAll(composeDir, 0o755)
	app := &domain.ComposeApp{ID: uuid.New(), ProjectID: projID}
	c.writeEnvFile(context.Background(), composeDir, app)

	data, err := os.ReadFile(filepath.Join(composeDir, ".env"))
	if err != nil {
		t.Fatalf("ler .env: %v", err)
	}
	content := string(data)
	if content != "DB_PASS=topsecret\nREGION=sa-east-1\n" {
		t.Fatalf(".env inesperado: %q", content)
	}
}

func TestComposeWriteEnvFileEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	projID := uuid.New()
	envID := uuid.New()
	c := &Compose{
		DataDir: dir,
		ProjectVars: &fakeVarStore{vars: []variablesDomain.Variable{
			{ProjectID: projID, Key: "HOST", Value: "project.example.com"},
		}},
	}
	// ambiente com HOST diferente — deve sobrescrever
	envStore := &fakeVarStore{vars: []variablesDomain.Variable{
		{ProjectID: projID, EnvironmentID: envID, Key: "HOST", Value: "staging.example.com"},
	}}
	c.ProjectVars = &fakeVarMulti{project: c.ProjectVars, env: envStore}

	composeDir := filepath.Join(dir, "compose", "stack-2")
	_ = os.MkdirAll(composeDir, 0o755)
	app := &domain.ComposeApp{ID: uuid.New(), ProjectID: projID, EnvironmentID: &envID}
	c.writeEnvFile(context.Background(), composeDir, app)

	data, _ := os.ReadFile(filepath.Join(composeDir, ".env"))
	if string(data) != "HOST=staging.example.com\n" {
		t.Fatalf("env deveria sobrescrever project: %q", string(data))
	}
}

func TestInjectComposeLabels(t *testing.T) {
	src := `services:
  api:
    image: nginx:alpine
    labels:
      - app=custom
  worker:
    image: alpine
    environment:
      - X=1
`
	out, err := injectComposeLabels(src, map[string]string{
		"aether.owner":        "user",
		"aether.service-type": "compose",
		"aether.service-id":   "abc",
	})
	if err != nil {
		t.Fatalf("inject: %v", err)
	}
	var doc map[string]map[string]map[string]any
	if err := yaml.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("invalid yaml after inject: %v", err)
	}
	apiLabels := doc["services"]["api"]["labels"].([]any)
	found := false
	for _, l := range apiLabels {
		if s, ok := l.(string); ok && s == "aether.owner=user" {
			found = true
		}
	}
	if !found {
		t.Fatalf("api service labels missing aether.owner: %v", apiLabels)
	}
	workerLabels := doc["services"]["worker"]["labels"].(map[string]any)
	if workerLabels["aether.service-id"] != "abc" {
		t.Fatalf("worker labels wrong: %v", workerLabels)
	}
	// environment must survive
	if v := doc["services"]["worker"]["environment"].([]any); len(v) != 1 {
		t.Fatalf("environment not preserved: %v", v)
	}
}

func TestInjectComposeLabelsPreservesAnchors(t *testing.T) {
	src := `x-common: &common
  restart: always
services:
  api:
    <<: *common
    image: nginx
`
	out, err := injectComposeLabels(src, map[string]string{"aether.owner": "user"})
	if err != nil {
		t.Fatalf("inject: %v", err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("invalid yaml: %v", err)
	}
	svc := doc["services"].(map[string]any)["api"].(map[string]any)
	if svc["restart"] != "always" {
		t.Fatalf("anchor merge lost: %v", svc)
	}
	labels := svc["labels"].(map[string]any)
	if labels["aether.owner"] != "user" {
		t.Fatalf("labels missing: %v", labels)
	}
}

func TestInjectComposeLabelsRejectsInvalid(t *testing.T) {
	if _, err := injectComposeLabels("not: [valid", map[string]string{"aether.owner": "user"}); err == nil {
		t.Fatal("expected error for invalid yaml")
	}
}
