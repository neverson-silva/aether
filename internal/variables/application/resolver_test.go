package application

import (
	"context"
	"testing"

	"github.com/google/uuid"

	appsdomain "aether/internal/apps/domain"
	appsInfra "aether/internal/apps/infra"
	"aether/internal/variables/domain"
	"aether/internal/variables/infra"
)

type resolverEnv struct {
	ctx       context.Context
	orgID     uuid.UUID
	projID    uuid.UUID
	envID     uuid.UUID
	appsStore *appsInfra.Store
	vars      domain.Store
}

func newResolverEnv(t *testing.T) *resolverEnv {
	t.Helper()
	pool := testPool(t)
	appsStore := appsInfra.NewStore(pool)
	vars := infra.NewStore(pool)
	e := &resolverEnv{ctx: context.Background(), orgID: uuid.New(), appsStore: appsStore, vars: vars}
	t.Cleanup(func() {
		_ = appsStore.Close()
		_ = vars.Close()
	})
	if _, err := pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, e.orgID, "R Org", "r-org"); err != nil {
		t.Fatalf("criar org: %v", err)
	}
	p, err := appsStore.CreateProject(e.ctx, e.orgID, "RProj", "r-proj", "", "")
	if err != nil {
		t.Fatalf("criar project: %v", err)
	}
	e.projID = p.ID
	env, err := appsStore.CreateEnvironment(e.ctx, p.ID, "prod", "prod", "", "", true)
	if err != nil {
		t.Fatalf("criar env: %v", err)
	}
	e.envID = env.ID
	return e
}

func (e *resolverEnv) newApp(t *testing.T, name string) *appsdomain.App {
	t.Helper()
	return e.newAppInEnv(t, name, e.envID)
}

func (e *resolverEnv) newAppInEnv(t *testing.T, name string, envID uuid.UUID) *appsdomain.App {
	t.Helper()
	app, err := e.appsStore.CreateApp(e.ctx, &appsdomain.App{
		OrgID: e.orgID, ProjectID: e.projID, EnvironmentID: &envID,
		Name: name, SourceType: "image", Image: "nginx", Port: 80,
	})
	if err != nil {
		t.Fatalf("criar app: %v", err)
	}
	return app
}

func (e *resolverEnv) newEnvironment(t *testing.T, name string) uuid.UUID {
	t.Helper()
	env, err := e.appsStore.CreateEnvironment(e.ctx, e.projID, name, name, "", "", false)
	if err != nil {
		t.Fatalf("criar env: %v", err)
	}
	return env.ID
}

func (e *resolverEnv) setEnvIn(envID uuid.UUID, key, val string) {
	if _, err := e.vars.UpsertVariable(e.ctx, &domain.Variable{ProjectID: e.projID, EnvironmentID: envID, Key: key, Value: val}); err != nil {
		panic(err)
	}
}

func (e *resolverEnv) setProject(key, val string) {
	if _, err := e.vars.UpsertVariable(e.ctx, &domain.Variable{ProjectID: e.projID, EnvironmentID: uuid.Nil, Key: key, Value: val}); err != nil {
		panic(err)
	}
}

func (e *resolverEnv) setEnv(key, val string) {
	if _, err := e.vars.UpsertVariable(e.ctx, &domain.Variable{ProjectID: e.projID, EnvironmentID: e.envID, Key: key, Value: val}); err != nil {
		panic(err)
	}
}

func (e *resolverEnv) setService(appID uuid.UUID, key, val string) {
	if err := e.appsStore.UpsertEnvVar(e.ctx, appID, key, val, false); err != nil {
		panic(err)
	}
}

func (e *resolverEnv) effective(appID uuid.UUID) map[string]string {
	r := &Resolver{Vars: e.vars, Apps: e.appsStore}
	m, err := r.Effective(e.ctx, appID, e.orgID)
	if err != nil {
		panic(err)
	}
	return m
}

func TestResolverPrecedence(t *testing.T) {
	e := newResolverEnv(t)
	app := e.newApp(t, "api")
	e.setProject("PORT", "3000")
	e.setProject("LOG_LEVEL", "info")
	e.setProject("REGION", "sa-east-1")
	e.setEnv("PORT", "4000")
	e.setEnv("LOG_LEVEL", "debug")
	e.setService(app.ID, "PORT", "5000")

	got := e.effective(app.ID)
	if got["PORT"] != "5000" || got["LOG_LEVEL"] != "debug" || got["REGION"] != "sa-east-1" {
		t.Fatalf("precedência errada: %+v", got)
	}
}

func TestResolverProjectOnly(t *testing.T) {
	e := newResolverEnv(t)
	app := e.newApp(t, "api")
	e.setProject("A", "1")
	got := e.effective(app.ID)
	if got["A"] != "1" {
		t.Fatalf("project var não aplicada: %+v", got)
	}
}

func TestResolverServiceOnly(t *testing.T) {
	e := newResolverEnv(t)
	appA := e.newApp(t, "svc-a")
	appB := e.newApp(t, "svc-b")
	e.setService(appA.ID, "SECRET_A", "foo")
	e.setService(appB.ID, "SECRET_B", "bar")
	gotA := e.effective(appA.ID)
	gotB := e.effective(appB.ID)
	if gotA["SECRET_A"] != "foo" || gotA["SECRET_B"] != "" {
		t.Fatalf("service A vazou: %+v", gotA)
	}
	if gotB["SECRET_B"] != "bar" || gotB["SECRET_A"] != "" {
		t.Fatalf("service B vazou: %+v", gotB)
	}
}

func TestResolverPlaceholder(t *testing.T) {
	e := newResolverEnv(t)
	app := e.newApp(t, "api")
	e.setProject("HOST", "project.example.com")
	e.setProject("PORT", "3000")
	e.setProject("URL", "https://${HOST}:${PORT}")
	e.setEnv("HOST", "staging.example.com")
	e.setService(app.ID, "PORT", "8080")

	got := e.effective(app.ID)
	if got["HOST"] != "staging.example.com" {
		t.Fatalf("env override não aplicado: %+v", got)
	}
	if got["PORT"] != "8080" {
		t.Fatalf("service override não aplicado: %+v", got)
	}
	if got["URL"] != "https://staging.example.com:8080" {
		t.Fatalf("placeholder não usou efetiva: %+v", got)
	}
}

func TestResolverPlaceholderChain(t *testing.T) {
	e := newResolverEnv(t)
	app := e.newApp(t, "api")
	e.setProject("DOMAIN", "example.com")
	e.setEnv("SUBDOMAIN", "api")
	e.setService(app.ID, "HOST", "${SUBDOMAIN}.${DOMAIN}")
	e.setService(app.ID, "URL", "https://${HOST}")

	got := e.effective(app.ID)
	if got["HOST"] != "api.example.com" || got["URL"] != "https://api.example.com" {
		t.Fatalf("chain de placeholder errado: %+v", got)
	}
}

func TestResolverCircular(t *testing.T) {
	e := newResolverEnv(t)
	app := e.newApp(t, "api")
	e.setProject("A", "${B}")
	e.setProject("B", "${A}")

	r := &Resolver{Vars: e.vars, Apps: e.appsStore}
	if _, err := r.Effective(e.ctx, app.ID, e.orgID); err == nil {
		t.Fatalf("ciclo deveria ser detectado")
	}
}

func TestResolverMissingPlaceholderStaysLiteral(t *testing.T) {
	e := newResolverEnv(t)
	app := e.newApp(t, "api")
	e.setProject("URL", "https://${MISSING_HOST}")
	got := e.effective(app.ID)
	if got["URL"] != "https://${MISSING_HOST}" {
		t.Fatalf("placeholder desconhecido deveria permanecer literal: %+v", got)
	}
}

func TestResolverEnvironmentIsolation(t *testing.T) {
	e := newResolverEnv(t)
	envB := e.newEnvironment(t, "staging")
	appA := e.newAppInEnv(t, "svc-a", e.envID)
	appB := e.newAppInEnv(t, "svc-b", envB)
	e.setEnvIn(e.envID, "SECRET_A", "123")
	e.setEnvIn(envB, "SECRET_B", "456")

	gotA := e.effective(appA.ID)
	gotB := e.effective(appB.ID)
	if gotA["SECRET_A"] != "123" || gotA["SECRET_B"] != "" {
		t.Fatalf("env A vazou: %+v", gotA)
	}
	if gotB["SECRET_B"] != "456" || gotB["SECRET_A"] != "" {
		t.Fatalf("env B vazou: %+v", gotB)
	}
}

func TestResolverProjectSharedAcrossEnvs(t *testing.T) {
	e := newResolverEnv(t)
	envB := e.newEnvironment(t, "staging")
	appA := e.newAppInEnv(t, "svc-a", e.envID)
	appB := e.newAppInEnv(t, "svc-b", envB)
	e.setProject("GLOBAL_REGION", "sa-east-1")
	e.setEnvIn(e.envID, "A", "value-a")
	e.setEnvIn(envB, "B", "value-b")

	gotA := e.effective(appA.ID)
	gotB := e.effective(appB.ID)
	if gotA["GLOBAL_REGION"] != "sa-east-1" || gotA["A"] != "value-a" || gotA["B"] != "" {
		t.Fatalf("service A errado: %+v", gotA)
	}
	if gotB["GLOBAL_REGION"] != "sa-east-1" || gotB["B"] != "value-b" || gotB["A"] != "" {
		t.Fatalf("service B errado: %+v", gotB)
	}
}

func TestResolverDifferentEnvOverrides(t *testing.T) {
	e := newResolverEnv(t)
	envX := e.newEnvironment(t, "x")
	envY := e.newEnvironment(t, "y")
	appX := e.newAppInEnv(t, "svc-x", envX)
	appY := e.newAppInEnv(t, "svc-y", envY)
	e.setProject("A", "1")
	e.setEnvIn(envX, "A", "2")
	e.setEnvIn(envY, "A", "3")

	if e.effective(appX.ID)["A"] != "2" || e.effective(appY.ID)["A"] != "3" {
		t.Fatalf("overrides por env errados")
	}
}

func TestResolverEnvironmentOverride(t *testing.T) {
	e := newResolverEnv(t)
	app := e.newApp(t, "api")
	e.setProject("A", "1")
	e.setEnv("A", "2")
	if got := e.effective(app.ID)["A"]; got != "2" {
		t.Fatalf("env deveria sobrescrever project: %s", got)
	}
}

func TestResolverResolvedSource(t *testing.T) {
	e := newResolverEnv(t)
	app := e.newApp(t, "api")
	e.setProject("REGION", "sa-east-1")
	e.setEnv("API_URL", "https://api.internal")
	e.setService(app.ID, "PORT", "3000")

	r := &Resolver{Vars: e.vars, Apps: e.appsStore}
	resolved, err := r.Resolved(e.ctx, app.ID, e.orgID)
	if err != nil {
		t.Fatalf("resolved: %v", err)
	}
	src := map[string]string{}
	for _, v := range resolved {
		src[v.Key] = v.Source
	}
	if src["REGION"] != "project" || src["API_URL"] != "environment" || src["PORT"] != "service" {
		t.Fatalf("fontes erradas: %+v", src)
	}
}

func TestResolverCrossOrg(t *testing.T) {
	e := newResolverEnv(t)
	app := e.newApp(t, "api")
	e.setProject("A", "1")

	r := &Resolver{Vars: e.vars, Apps: e.appsStore}
	if _, err := r.Effective(e.ctx, app.ID, uuid.New()); err == nil {
		t.Fatalf("cross-org effective deveria falhar")
	}
}

func TestResolverOverrideRemoval(t *testing.T) {
	e := newResolverEnv(t)
	app := e.newApp(t, "api")
	e.setProject("PORT", "3000")
	e.setEnv("PORT", "4000")
	e.setService(app.ID, "PORT", "5000")

	if got := e.effective(app.ID)["PORT"]; got != "5000" {
		t.Fatalf("service override esperado 5000, got %s", got)
	}
	// remove service override → volta para env (4000)
	if err := e.appsStore.DeleteEnvVar(e.ctx, app.ID, "PORT"); err != nil {
		t.Fatalf("delete service: %v", err)
	}
	if got := e.effective(app.ID)["PORT"]; got != "4000" {
		t.Fatalf("após delete service esperado 4000, got %s", got)
	}
	// remove env override → volta para project (3000)
	if err := e.vars.DeleteVariable(e.ctx, e.projID, e.envID, "PORT"); err != nil {
		t.Fatalf("delete env: %v", err)
	}
	if got := e.effective(app.ID)["PORT"]; got != "3000" {
		t.Fatalf("após delete env esperado 3000, got %s", got)
	}
	// remove project → ausente
	if err := e.vars.DeleteVariable(e.ctx, e.projID, uuid.Nil, "PORT"); err != nil {
		t.Fatalf("delete project: %v", err)
	}
	if got := e.effective(app.ID)["PORT"]; got != "" {
		t.Fatalf("após delete project esperado ausente, got %s", got)
	}
}

func TestResolverScopedPlaceholder(t *testing.T) {
	e := newResolverEnv(t)
	app := e.newApp(t, "api")
	e.setProject("APP_NAME", "myapp")
	e.setService(app.ID, "APP_NAME", "${{project.APP_NAME}}")
	got := e.effective(app.ID)
	if got["APP_NAME"] != "myapp" {
		t.Fatalf("${{project.X}} deveria resolver no escopo project: %q", got["APP_NAME"])
	}
}

func TestResolverScopedPlaceholderMissing(t *testing.T) {
	e := newResolverEnv(t)
	app := e.newApp(t, "api")
	e.setService(app.ID, "APP_NAME", "${{project.APP_NAME}}")
	got := e.effective(app.ID)
	if got["APP_NAME"] != "${{project.APP_NAME}}" {
		t.Fatalf("placeholder ausente deveria permanecer literal: %q", got["APP_NAME"])
	}
}

func TestResolverScopedPlaceholderNotCircular(t *testing.T) {
	e := newResolverEnv(t)
	app := e.newApp(t, "api")
	// service APP_NAME referencia project APP_NAME; project não tem → NÃO deve ser ciclo
	e.setService(app.ID, "APP_NAME", "${{project.APP_NAME}}")
	r := &Resolver{Vars: e.vars, Apps: e.appsStore}
	if _, err := r.Effective(e.ctx, app.ID, e.orgID); err != nil {
		t.Fatalf("não deveria ser circular: %v", err)
	}
}

func TestResolverSecretEncryption(t *testing.T) {
	pool := testPool(t)
	vars := infra.NewStore(pool)
	apps := appsInfra.NewStore(pool)
	t.Cleanup(func() { _ = apps.Close(); _ = vars.Close() })

	cipher, err := appsInfra.NewSecretCipher([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	vars.Cipher = cipher
	apps.Cipher = cipher

	ctx := context.Background()
	orgID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1,$2,$3,NULL)`, orgID, "Enc Org", "enc-org"); err != nil {
		t.Fatalf("org: %v", err)
	}
	proj, err := apps.CreateProject(ctx, orgID, "EncProj", "enc-proj", "", "")
	if err != nil {
		t.Fatalf("proj: %v", err)
	}
	env, err := apps.CreateEnvironment(ctx, proj.ID, "prod", "prod", "", "", true)
	if err != nil {
		t.Fatalf("env: %v", err)
	}
	app, err := apps.CreateApp(ctx, &appsdomain.App{OrgID: orgID, ProjectID: proj.ID, EnvironmentID: &env.ID, Name: "api", SourceType: "image", Image: "nginx", Port: 80})
	if err != nil {
		t.Fatalf("app: %v", err)
	}

	if _, err := vars.UpsertVariable(ctx, &domain.Variable{ProjectID: proj.ID, EnvironmentID: uuid.Nil, Key: "PASS", Value: "supersecret", IsSecret: true}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	raw, _ := vars.ListVariables(ctx, proj.ID, uuid.Nil)
	if len(raw) != 1 || raw[0].Value == "supersecret" {
		t.Fatalf("secret deveria estar criptografado no banco: %q", raw[0].Value)
	}

	r := &Resolver{Vars: vars, Apps: apps, Cipher: cipher}
	eff, err := r.Effective(ctx, app.ID, orgID)
	if err != nil {
		t.Fatalf("effective: %v", err)
	}
	if eff["PASS"] != "supersecret" {
		t.Fatalf("resolver deveria descriptografar: %q", eff["PASS"])
	}
}
