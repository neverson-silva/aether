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
	templatesinfra "aether/internal/modules/templates/infra"
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
}

func TestBundledDragonflyComposePassesTemplateSecurityPolicy(t *testing.T) {
	catalog := templatesinfra.NewDokployCatalog(t.TempDir())
	template, err := catalog.Get(context.Background(), uuid.NewSHA1(uuid.NameSpaceURL, []byte("dokploy:dragonfly-db")))
	if err != nil {
		t.Fatal(err)
	}
	secured, err := injectComposeSecurityDefaults(template.ComposeYAML)
	if err != nil {
		t.Fatal(err)
	}
	if err := composeengine.ValidatePolicy(secured); err != nil {
		t.Fatalf("Dragonfly Compose rejected: %v", err)
	}
}

func TestBundledAdGuardComposePassesTemplateSecurityPolicy(t *testing.T) {
	catalog := templatesinfra.NewDokployCatalog(t.TempDir())
	template, err := catalog.Get(context.Background(), uuid.NewSHA1(uuid.NameSpaceURL, []byte("dokploy:adguardhome")))
	if err != nil {
		t.Fatal(err)
	}
	secured, err := injectComposeSecurityDefaults(template.ComposeYAML)
	if err != nil {
		t.Fatal(err)
	}
	if err := composeengine.ValidatePolicy(secured); err != nil {
		t.Fatalf("AdGuard Compose rejected: %v\n%s", err, secured)
	}
	if strings.Contains(secured, "cap_drop:") || strings.Contains(secured, "no-new-privileges:true") {
		t.Fatalf("AdGuard host-port profile received incompatible runtime restrictions: %s", secured)
	}
}

func TestMaterializeTemplateMountsUsesInlineConfig(t *testing.T) {
	service := &Templates{}
	app := &domain.ComposeApp{ID: uuid.New(), Compose: `services:
  web:
    image: nginx:alpine
`}
	content, err := service.materializeTemplateMounts(app, []domain.TemplateMount{{FilePath: "/etc/nginx/nginx.conf", Content: "worker_processes 1;"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, "template-mounts") || !strings.Contains(content, "content: worker_processes 1;") {
		t.Fatalf("template mount was not materialized inline: %s", content)
	}
	if err := composeengine.ValidatePolicy(content); err != nil {
		t.Fatalf("inline template mount rejected: %v\n%s", err, content)
	}
}

func TestInlineTemplateConfigFilesMigratesLegacyMounts(t *testing.T) {
	baseDir := t.TempDir()
	mountDir := filepath.Join(baseDir, "template-mounts")
	if err := os.MkdirAll(mountDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mountDir, "0"), []byte("worker_processes 1;"), 0o600); err != nil {
		t.Fatal(err)
	}
	content, err := inlineTemplateConfigFiles(`services:
  web:
    image: nginx:alpine
configs:
  aether-template-file-0:
    file: ./template-mounts/0
`, baseDir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, "file: ./template-mounts/0") || !strings.Contains(content, "content: worker_processes 1;") {
		t.Fatalf("legacy template config was not migrated: %s", content)
	}
}

func TestComposePortSelectionKeepsInternalAddressPort(t *testing.T) {
	port, published, err := composePortSelection(`services:
  rustfs:
    environment:
      - RUSTFS_ADDRESS=0.0.0.0:9000
      - RUSTFS_CONSOLE_ADDRESS=0.0.0.0:9001
`)
	if err != nil {
		t.Fatal(err)
	}
	if published || port != 9000 {
		t.Fatalf("RustFS port selection = %d published=%v, want internal 9000", port, published)
	}
}

func TestComposePortSelectionKeepsInternalEnvironmentPort(t *testing.T) {
	port, published, err := composePortSelection(`services:
  web:
    environment:
      PORT: "8080"
`)
	if err != nil {
		t.Fatal(err)
	}
	if published || port != 8080 {
		t.Fatalf("environment port selection = %d published=%v, want internal 8080", port, published)
	}
}

func TestComposeRuntimePortResolvesEnvironmentPort(t *testing.T) {
	port, err := composeRuntimePort(`services:
  app:
    ports:
      - "${PORT}:${PORT}"
`, 0, map[string]string{"PORT": "8080"})
	if err != nil {
		t.Fatal(err)
	}
	if port != 8080 {
		t.Fatalf("runtime port = %d, want 8080", port)
	}
	content, err := materializeComposePortBindings(`services:
  app:
    ports:
      - "${PORT}:${PORT}"
      - ":3000"
      - ":"
`, port)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, "${PORT}") || !strings.Contains(content, "8080:3000") || !strings.Contains(content, "\"8080\"") {
		t.Fatalf("compose ports were not materialized: %s", content)
	}
}

func TestComposeRuntimePortUsesComposeDefault(t *testing.T) {
	port, err := composeRuntimePort(`services:
  app:
    environment:
      PORT: "${PORT:-3000}"
`, 0, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if port != 3000 {
		t.Fatalf("runtime port = %d, want 3000", port)
	}
}

func TestComposeRuntimePortDoesNotUseFixedFallback(t *testing.T) {
	port, err := composeRuntimePort(`services:
  app:
    image: example/app
`, 0, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if port != 0 {
		t.Fatalf("runtime port = %d, want 0", port)
	}
}

func TestComposeRuntimePortPrefersExplicitPortVariable(t *testing.T) {
	port, err := composeRuntimePort(`services:
  app:
    ports:
      - "8080:8080"
`, 8080, map[string]string{"PORT": "3000"})
	if err != nil {
		t.Fatal(err)
	}
	if port != 3000 {
		t.Fatalf("runtime port = %d, want 3000", port)
	}
}

func TestComposeRuntimePortPrefersComposeEnvironmentPort(t *testing.T) {
	port, err := composeRuntimePort(`services:
  app:
    environment:
      PORT: "3000"
`, 8080, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if port != 3000 {
		t.Fatalf("runtime port = %d, want 3000", port)
	}
}

func TestEnsureComposeRuntimePortEnvironment(t *testing.T) {
	content, err := ensureComposeRuntimePortEnvironment(`services:
  app:
    image: example/app
`, 8080, "app")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "PORT: \"8080\"") {
		t.Fatalf("PORT was not injected: %s", content)
	}
	content, err = ensureComposeRuntimePortEnvironment(`services:
  app:
    environment:
      PORT: "3000"
`, 8080, "app")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "PORT: \"3000\"") || strings.Contains(content, "PORT: \"8080\"") {
		t.Fatalf("explicit PORT was overwritten: %s", content)
	}
}

func TestEnsureRustFSPublishedPorts(t *testing.T) {
	content, err := ensureRustFSPublishedPorts(`services:
  rustfs:
    image: rustfs/rustfs:1.0
`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "9000:9000") || !strings.Contains(content, "9001:9001") {
		t.Fatalf("RustFS ports were not published: %s", content)
	}
}

func TestResolveTemplateConfigUsesDokployVariables(t *testing.T) {
	service := &Templates{TemplateDomainGenerator: func(seed string) string { return seed + ".example.com" }}
	configured, domains, mounts := service.resolveTemplateConfig(&domain.Template{
		Variables:   []domain.TemplateVariable{{Name: "main_domain", Value: "${domain}"}, {Name: "secret", Value: "${password:16}"}},
		Environment: []domain.TemplateEnvironmentVariable{{Name: "PUBLIC_URL", Value: "https://${main_domain}"}, {Name: "SECRET", Value: "${secret}"}},
		Domains:     []domain.TemplateDomain{{ServiceName: "web", Port: 3000, Host: "${main_domain}", Path: "/"}},
	}, "my-service", nil)
	if len(mounts) != 0 {
		t.Fatalf("resolved template mounts = %+v", mounts)
	}
	if len(configured) != 2 || configured[0].Value == "" || configured[1].Value == "" {
		t.Fatalf("resolved template environment = %+v", configured)
	}
	if len(configured[1].Value) != 16 {
		t.Fatalf("resolved secret length = %d", len(configured[1].Value))
	}
	if len(domains) != 1 || domains[0].Host != "main_domain.example.com" {
		t.Fatalf("resolved template domains = %+v", domains)
	}
}

func TestMaterializeTemplateIngressUsesServiceAliases(t *testing.T) {
	app := &domain.ComposeApp{ID: uuid.MustParse("12345678-1234-1234-1234-123456789abc"), Compose: `services:
  web:
    image: nginx:1.27
  worker:
    image: busybox:1.36
`}
	service := &Templates{IngressNetwork: "aether-ingress"}
	content, err := service.materializeTemplateIngress(app, []domain.TemplateDomain{{ServiceName: "web", Port: 3000, Host: "web.example.com", Path: "/"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := composeengine.ValidatePolicy(content); err != nil {
		t.Fatalf("materialized ingress compose rejected: %v\n%s", err, content)
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		t.Fatal(err)
	}
	services := document["services"].(map[string]any)
	web := services["web"].(map[string]any)
	networks := web["networks"].(map[string]any)
	if _, ok := networks["aether-ingress"]; !ok {
		t.Fatalf("web service is not attached to ingress: %v", networks)
	}
	if _, ok := services["worker"].(map[string]any)["networks"]; ok {
		t.Fatal("worker without a template domain must not be attached to ingress")
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
	if !strings.Contains(content, "restart: no") {
		t.Fatalf("restart policy was not normalized: %s", content)
	}
}

func TestInterpolateComposeVariableReferences(t *testing.T) {
	content, err := interpolateComposeVariables("services:\n  app:\n    environment:\n      PASSWORD: ${APP_PASSWORD}\n      URL: https://${APP_HOST:-localhost}\n", map[string]string{"APP_PASSWORD": "secret", "APP_HOST": "example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, "${APP_PASSWORD}") || strings.Contains(content, "${APP_HOST") || !strings.Contains(content, "PASSWORD: secret") || !strings.Contains(content, "URL: https://example.com") {
		t.Fatalf("compose variables were not interpolated: %s", content)
	}
}

func TestInjectComposeEnvironmentAddsResolvedVariables(t *testing.T) {
	content, err := injectComposeEnvironment(`services:
  api:
    image: example/api
  worker:
    image: example/worker
    environment:
      EXISTING:
`, map[string]string{"API_KEY": "secret", "EXISTING": "value"})
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		t.Fatal(err)
	}
	services := document["services"].(map[string]any)
	apiEnvironment := services["api"].(map[string]any)["environment"].(map[string]any)
	workerEnvironment := services["worker"].(map[string]any)["environment"].(map[string]any)
	if apiEnvironment["API_KEY"] != "secret" || workerEnvironment["API_KEY"] != "secret" || workerEnvironment["EXISTING"] != "value" {
		t.Fatalf("resolved variables were not injected: %s", content)
	}
}

func TestRustFSComposeAdmission(t *testing.T) {
	content := `version: "3.8"
services:
  rustfs:
    image: rustfs/rustfs@sha256:41fe89380f4120a337790c02af192c3fe7bb55c3edc2e6e9357b487b47c6ab21
    ports:
      - "9000:9000"
      - "9001:9001"
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
	if !hasPort || port != 9000 {
		t.Fatalf("RustFS published port = %d, has=%v; want 9000", port, hasPort)
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

func TestTemplateEnvironmentVariablesSkipsReferences(t *testing.T) {
	variables := templateEnvironmentVariables(`services:
  app:
    environment:
      PASSWORD: ${APP_PASSWORD}
      APP_PASSWORD:
`)
	if len(variables) != 1 || variables[0].Name != "APP_PASSWORD" {
		t.Fatalf("template references were treated as independent variables: %v", variables)
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
	if _, err := compose.runCompose(context.Background(), app, false, "down"); err != nil {
		t.Fatal(err)
	}
	if executor.project.File != filepath.Join(dir, "compose", appID.String(), "compose.generated.yml") {
		t.Fatalf("down project file = %q", executor.project.File)
	}
}
