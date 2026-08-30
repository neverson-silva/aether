package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	sourcedomain "aether/internal/modules/sourcecontrol/domain"
	"aether/internal/modules/templates/domain"
	variablesDomain "aether/internal/modules/variables/domain"
	composeengine "aether/internal/platform/compose"
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

type fakeComposeSource struct {
	source *sourcedomain.ServiceSource
}

func (f fakeComposeSource) GetByService(context.Context, uuid.UUID, uuid.UUID) (*sourcedomain.ServiceSource, error) {
	return f.source, nil
}

type fakeComposeClone struct{}

func TestPublishedPort(t *testing.T) {
	port, found, err := PublishedPort("services:\n  app:\n    ports:\n      - \"5000:5000\"\n")
	if err != nil {
		t.Fatalf("parse compose port: %v", err)
	}
	if !found || port != 5000 {
		t.Fatalf("unexpected compose port: found=%v port=%d", found, port)
	}
}

func (fakeComposeClone) Clone(ctx context.Context, source *sourcedomain.ServiceSource, destination string) (string, error) {
	file := filepath.Join(destination, "api-funvest", "infra", "waf", "docker-compose.yml")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(file, []byte("services:\n  waf:\n    build:\n      context: .\n"), 0o644); err != nil {
		return "", err
	}
	return destination, nil
}

type fakeComposeExecutor struct {
	project composeengine.Project
	args    []string
}

func (f *fakeComposeExecutor) Execute(ctx context.Context, project composeengine.Project, args ...string) (string, error) {
	f.project = project
	f.args = append([]string(nil), args...)
	return "ok", nil
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
	if err := c.writeEnvFile(context.Background(), composeDir, composeDir, app); err != nil {
		t.Fatal(err)
	}

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
	if err := c.writeEnvFile(context.Background(), composeDir, composeDir, app); err != nil {
		t.Fatal(err)
	}

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

func TestRepositoryPathRejectsEscape(t *testing.T) {
	for _, value := range []string{"../Dockerfile", "/tmp/compose.yml", "../../secret"} {
		if _, err := repositoryPath(value); err == nil {
			t.Fatalf("expected path rejection for %q", value)
		}
	}
}

func TestRepositoryPathNormalizesNestedFile(t *testing.T) {
	value, err := repositoryPath("infra/waf/../docker-compose.yml")
	if err != nil {
		t.Fatal(err)
	}
	if value != filepath.Join("infra", "docker-compose.yml") {
		t.Fatalf("path = %q", value)
	}
}

func TestInjectComposeLabelsUpdatesExistingValues(t *testing.T) {
	src := `services:
  api:
    image: nginx
    labels:
      aether.service-id: old
`
	out, err := injectComposeLabels(src, map[string]string{"aether.service-id": "new"})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	service := doc["services"].(map[string]any)["api"].(map[string]any)
	labels := service["labels"].(map[string]any)
	if labels["aether.service-id"] != "new" {
		t.Fatalf("labels = %v", labels)
	}
}

func TestComposeGitDeploymentUsesNestedComposeDirectory(t *testing.T) {
	dir := t.TempDir()
	appID := uuid.New()
	orgID := uuid.New()
	executor := &fakeComposeExecutor{}
	compose := &Compose{
		DataDir:        dir,
		ComposeRuntime: executor,
		Source: fakeComposeSource{source: &sourcedomain.ServiceSource{
			RepositoryFullName: "owner/repository",
			Branch:             "feature",
			ComposeFile:        "api-funvest/infra/waf/docker-compose.yml",
		}},
		Clone: fakeComposeClone{},
	}
	app := &domain.ComposeApp{ID: appID, OrgID: orgID, ProjectID: uuid.New(), Name: "waf", Compose: "services: {}\n"}
	if _, err := compose.runCompose(context.Background(), app, true, "up", "-d"); err != nil {
		t.Fatal(err)
	}
	wantDir := filepath.Join(dir, "compose", appID.String(), "checkout", "api-funvest", "infra", "waf")
	if executor.project.Directory != wantDir {
		t.Fatalf("project directory = %q, want %q", executor.project.Directory, wantDir)
	}
	if executor.project.File != filepath.Join(dir, "compose", appID.String(), "compose.generated.yml") {
		t.Fatalf("project file = %q", executor.project.File)
	}
	if len(executor.args) != 2 || executor.args[0] != "up" || executor.args[1] != "-d" {
		t.Fatalf("compose args = %v", executor.args)
	}
}
