package application

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	appsdomain "aether/internal/modules/apps/domain"
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

func (f *fakeVarStore) UpsertVariable(ctx context.Context, variable *variablesDomain.Variable) (*variablesDomain.Variable, error) {
	for i := range f.vars {
		if f.vars[i].ProjectID == variable.ProjectID && f.vars[i].EnvironmentID == variable.EnvironmentID && f.vars[i].Key == variable.Key {
			f.vars[i] = *variable
			return variable, nil
		}
	}
	f.vars = append(f.vars, *variable)
	return variable, nil
}

type fakeVarMulti struct {
	project ProjectVarStore
	env     ProjectVarStore
}

type fakeServiceVarStore struct {
	vars []appsdomain.EnvVar
}

func (f *fakeServiceVarStore) ListServiceEnvVars(context.Context, uuid.UUID) ([]appsdomain.EnvVar, error) {
	return append([]appsdomain.EnvVar(nil), f.vars...), nil
}

func (f *fakeServiceVarStore) UpsertServiceEnvVar(_ context.Context, _ uuid.UUID, name, value string, secret bool) error {
	for i := range f.vars {
		if f.vars[i].Name == name {
			f.vars[i] = appsdomain.EnvVar{Name: name, Value: value, Secret: secret}
			return nil
		}
	}
	f.vars = append(f.vars, appsdomain.EnvVar{Name: name, Value: value, Secret: secret})
	return nil
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

func TestComposeContainerPortUsesAddress(t *testing.T) {
	content := `services:
  rustfs:
    environment:
      - RUSTFS_ADDRESS=0.0.0.0:9000
      - RUSTFS_CONSOLE_ADDRESS=0.0.0.0:9001
`
	port, err := composeContainerPort(content)
	if err != nil {
		t.Fatal(err)
	}
	if port != 9000 {
		t.Fatalf("container port = %d, want 9000", port)
	}
	content, err = addComposePort(content, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "1024:9000") {
		t.Fatalf("published mapping = %s", content)
	}
}

func TestRepairComposePortMapping(t *testing.T) {
	content, repaired, err := repairComposePortMapping(`services:
  rustfs:
    ports:
      - "1024:1024"
    environment:
      - RUSTFS_ADDRESS=0.0.0.0:9000
`, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if !repaired || !strings.Contains(content, "1024:9000") {
		t.Fatalf("compose mapping was not repaired: repaired=%v content=%s", repaired, content)
	}
}

func TestInjectComposeSecurityDefaults(t *testing.T) {
	content, err := injectComposeSecurityDefaults("services:\n  web:\n    image: rustfs/rustfs@sha256:41fe89380f4120a337790c02af192c3fe7bb55c3edc2e6e9357b487b47c6ab21\n    restart: unless-stopped\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := composeengine.ValidatePolicy(content); err != nil {
		t.Fatalf("security defaults rejected: %v", err)
	}
	if !strings.Contains(content, "cap_drop") || !strings.Contains(content, "no-new-privileges:true") {
		t.Fatalf("security defaults missing: %s", content)
	}
	if !strings.Contains(content, "restart: no") {
		t.Fatalf("restart policy was not normalized: %s", content)
	}
}

func TestRustFSComposeAdmission(t *testing.T) {
	content := `version: "3.8"
services:
  rustfs:
    image: rustfs/rustfs@sha256:41fe89380f4120a337790c02af192c3fe7bb55c3edc2e6e9357b487b47c6ab21
    volumes:
      - rustfs-data:/data
    environment:
      - RUSTFS_ACCESS_KEY
      - RUSTFS_SECRET_KEY
    command: /data
    restart: unless-stopped
volumes:
  rustfs-data:`

	port, hasPort, err := composePublishedPort(content)
	if err != nil {
		t.Fatalf("parse RustFS ports: %v", err)
	}
	if hasPort || port != 0 {
		t.Fatalf("RustFS should not have a published port: has=%v port=%d", hasPort, port)
	}
	content, err = addComposePort(content, 1024)
	if err != nil {
		t.Fatalf("add RustFS port: %v", err)
	}
	content, err = injectComposeSecurityDefaults(content)
	if err != nil {
		t.Fatalf("normalize RustFS compose: %v", err)
	}
	if err := composeengine.ValidatePolicy(content); err != nil {
		t.Fatalf("RustFS compose rejected: %v\n%s", err, content)
	}
}

func TestTemplateEnvironmentVariablesReadsBareAndExplicitValues(t *testing.T) {
	variables := templateEnvironmentVariables(`services:
  rustfs:
    environment:
      - RUSTFS_ACCESS_KEY
      - RUSTFS_SECRET_KEY
      - RUSTFS_ADDRESS=0.0.0.0:9000
`)
	if len(variables) != 3 {
		t.Fatalf("environment variables = %d, want 3", len(variables))
	}
	if variables[0].Name != "RUSTFS_ACCESS_KEY" || variables[0].Value != "" {
		t.Fatalf("unexpected access key variable: %+v", variables[0])
	}
	if variables[2].Name != "RUSTFS_ADDRESS" || variables[2].Value != "0.0.0.0:9000" {
		t.Fatalf("unexpected address variable: %+v", variables[2])
	}
}

func TestEnsureComposeVariablesGeneratesMissingSecrets(t *testing.T) {
	projectID := uuid.New()
	vars := &fakeVarStore{}
	compose := &Compose{ProjectVars: vars}
	app := &domain.ComposeApp{ProjectID: projectID}
	if err := compose.ensureComposeVariables(context.Background(), app, `services:
  rustfs:
    environment:
      - RUSTFS_ACCESS_KEY
      - RUSTFS_SECRET_KEY
      - RUSTFS_ADDRESS=0.0.0.0:9000
`); err != nil {
		t.Fatal(err)
	}
	if len(vars.vars) != 3 {
		t.Fatalf("generated variables = %d, want 3", len(vars.vars))
	}
	secretCount := 0
	for _, variable := range vars.vars {
		if variable.Value == "" || variable.ProjectID != projectID {
			t.Fatalf("unexpected generated variable: %+v", variable)
		}
		if variable.IsSecret {
			secretCount++
		}
	}
	if secretCount != 2 {
		t.Fatalf("generated secrets = %d, want 2", secretCount)
	}
}

func TestEnsureServiceComposeVariablesCopiesProjectValues(t *testing.T) {
	projectID := uuid.New()
	serviceID := uuid.New()
	projectVars := &fakeVarStore{vars: []variablesDomain.Variable{
		{ProjectID: projectID, Key: "RUSTFS_ACCESS_KEY", Value: "access"},
		{ProjectID: projectID, Key: "RUSTFS_SECRET_KEY", Value: "secret", IsSecret: true},
	}}
	serviceVars := &fakeServiceVarStore{}
	compose := &Compose{ProjectVars: projectVars}
	app := &domain.ComposeApp{ID: uuid.New(), ProjectID: projectID, ServiceID: serviceID}
	if err := compose.ensureServiceComposeVariables(context.Background(), serviceVars, app, `services:
  rustfs:
    environment:
      - RUSTFS_ACCESS_KEY
      - RUSTFS_SECRET_KEY
      - RUSTFS_ADDRESS=0.0.0.0:9000
`); err != nil {
		t.Fatal(err)
	}
	if len(serviceVars.vars) != 3 {
		t.Fatalf("service variables = %d, want 3", len(serviceVars.vars))
	}
	if serviceVars.vars[1].Name != "RUSTFS_SECRET_KEY" || !serviceVars.vars[1].Secret {
		t.Fatalf("secret variable was not preserved: %+v", serviceVars.vars)
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
