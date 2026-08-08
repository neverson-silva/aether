package core

import (
	"strings"
	"testing"

	"aether/internal/domain"
)

func TestEnvVarPrecedenceAndInterpolation(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	_, org, err := c.CreateUserAndOrg("ev@aether.local", "ev", "senha-ev")
	if err != nil {
		t.Fatal(err)
	}
	proj, _ := c.CreateProject(org.ID, "evproj")
	envs, _ := c.ListEnvironments(proj.ID)
	env := envs[0]

	if _, err := c.SetEnvironmentVar(proj.ID, env.ID, "REGION", "us-east-1", false); err != nil {
		t.Fatal(err)
	}
	if _, err := c.SetEnvironmentVar(proj.ID, env.ID, "API_URL", "https://api.example.com", false); err != nil {
		t.Fatal(err)
	}
	if _, err := c.SetEnvironmentVar(proj.ID, env.ID, "JWT_SECRET", "segredo-super-secreto", true); err != nil {
		t.Fatal(err)
	}

	app := &domain.App{ID: domain.NewID(), OrgID: org.ID, ProjectID: proj.ID, EnvironmentID: env.ID, Name: "api"}
	c.CreateApp(org.ID, app)

	c.SetAppEnv(app.ID, "REGION", "sa-east-1", false)
	c.SetAppEnv(app.ID, "INTERPOLATED", "${{environment.API_URL}}", false)

	envVars, err := c.EnsureAppEnv(app.ID)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]string{}
	for _, line := range envVars {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	if m["REGION"] != "sa-east-1" {
		t.Fatalf("service deveria sobrepor environment: %s", m["REGION"])
	}
	if m["API_URL"] != "https://api.example.com" {
		t.Fatalf("environment deveria ser herdado: %s", m["API_URL"])
	}
	if m["JWT_SECRET"] != "segredo-super-secreto" {
		t.Fatalf("secret deveria ser descriptografado: %s", m["JWT_SECRET"])
	}
	if m["INTERPOLATED"] != "https://api.example.com" {
		t.Fatalf("interpolação falhou: %s", m["INTERPOLATED"])
	}
}

func TestEnvVarMaskingAndParser(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	_, org, err := c.CreateUserAndOrg("ev2@aether.local", "ev2", "senha-ev2")
	if err != nil {
		t.Fatal(err)
	}
	proj, _ := c.CreateProject(org.ID, "evproj2")
	envs, _ := c.ListEnvironments(proj.ID)
	env := envs[0]
	c.SetEnvironmentVar(proj.ID, env.ID, "TOKEN", "plain", true)

	masked, _ := c.ListEnvironmentVars(proj.ID, env.ID, false)
	if masked[0].Value != "••••••••••" {
		t.Fatalf("secret deveria estar mascarado: %s", masked[0].Value)
	}
	plain, _ := c.ListEnvironmentVars(proj.ID, env.ID, true)
	if plain[0].Value != "plain" {
		t.Fatalf("reveal deveria retornar o valor: %s", plain[0].Value)
	}

	parsed, err := ParseEnvText("A=1\n# comentário\nB=\"dois\"\n")
	if err != nil {
		t.Fatal(err)
	}
	if parsed["A"].Value != "1" || parsed["B"].Value != "dois" {
		t.Fatalf("parse: %+v", parsed)
	}
	if _, err := ParseEnvText("A=1\nA=2\n"); err == nil {
		t.Fatal("duplicada deveria falhar")
	}
	if _, err := ParseEnvText("invalida linha\n"); err == nil {
		t.Fatal("linha inválida deveria falhar")
	}
}

func TestEnvVarAuditAndCacheInvalidation(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	_, org, err := c.CreateUserAndOrg("ev3@aether.local", "ev3", "senha-ev3")
	if err != nil {
		t.Fatal(err)
	}
	proj, _ := c.CreateProject(org.ID, "evproj3")
	envs, _ := c.ListEnvironments(proj.ID)
	env := envs[0]

	c.SetEnvironmentVar(proj.ID, env.ID, "K", "v1", false)
	c.SetEnvironmentVar(proj.ID, env.ID, "K", "v2", false)
	c.DeleteEnvironmentVar(proj.ID, env.ID, "K")

	audit, err := c.Store.ListVariableAudit(env.ID, 10)
	if err != nil || len(audit) != 3 {
		t.Fatalf("audit deveria ter 3 registros: %d %v", len(audit), err)
	}
	got, _ := c.ListEnvironmentVars(proj.ID, env.ID, false)
	if len(got) != 0 {
		t.Fatalf("cache deveria ter sido invalidado: %+v", got)
	}
}

func TestEnvVarInterpolationInsideServiceVar(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	_, org, err := c.CreateUserAndOrg("ev4@aether.local", "ev4", "senha-ev4")
	if err != nil {
		t.Fatal(err)
	}
	proj, _ := c.CreateProject(org.ID, "evproj4")
	envs, _ := c.ListEnvironments(proj.ID)
	env := envs[0]
	c.SetEnvironmentVar(proj.ID, env.ID, "API_URL", "https://api.example.com", false)

	app := &domain.App{ID: domain.NewID(), OrgID: org.ID, ProjectID: proj.ID, EnvironmentID: env.ID, Name: "api"}
	c.CreateApp(org.ID, app)
	c.SetAppEnv(app.ID, "INTERNAL_URL", "${{environment.API_URL}}/internal", false)

	envVars, err := c.EnsureAppEnv(app.ID)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]string{}
	for _, line := range envVars {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	if m["INTERNAL_URL"] != "https://api.example.com/internal" {
		t.Fatalf("interpolação em variável do service falhou: %s", m["INTERNAL_URL"])
	}
}

func TestProjectVarsPrecedenceAndInterpolation(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	_, org, err := c.CreateUserAndOrg("ev5@aether.local", "ev5", "senha-ev5")
	if err != nil {
		t.Fatal(err)
	}
	proj, _ := c.CreateProject(org.ID, "evproj5")
	envs, _ := c.ListEnvironments(proj.ID)
	env := envs[0]

	c.SetProjectVar(proj.ID, "REGION", "us-east-1", false)
	c.SetProjectVar(proj.ID, "SMTP_HOST", "smtp.global.example.com", false)
	c.SetProjectVar(proj.ID, "SHARED_SECRET", "projeto-secreto", true)

	c.SetEnvironmentVar(proj.ID, env.ID, "REGION", "sa-east-1", false)
	c.SetEnvironmentVar(proj.ID, env.ID, "API_URL", "https://api.example.com", false)

	app := &domain.App{ID: domain.NewID(), OrgID: org.ID, ProjectID: proj.ID, EnvironmentID: env.ID, Name: "api"}
	c.CreateApp(org.ID, app)
	c.SetAppEnv(app.ID, "REGION", "eu-west-1", false)
	c.SetAppEnv(app.ID, "SMTP_REF", "${{project.SMTP_HOST}}", false)
	c.SetAppEnv(app.ID, "API_REF", "${{environment.API_URL}}", false)

	envVars, err := c.EnsureAppEnv(app.ID)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]string{}
	for _, line := range envVars {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	if m["REGION"] != "eu-west-1" {
		t.Fatalf("service deveria vencer: %s", m["REGION"])
	}
	if m["API_URL"] != "https://api.example.com" {
		t.Fatalf("environment deveria vencer project: %s", m["API_URL"])
	}
	if m["SMTP_HOST"] != "smtp.global.example.com" {
		t.Fatalf("project deveria ser herdado: %s", m["SMTP_HOST"])
	}
	if m["SHARED_SECRET"] != "projeto-secreto" {
		t.Fatalf("project secret deveria ser descriptografado: %s", m["SHARED_SECRET"])
	}
	if m["SMTP_REF"] != "smtp.global.example.com" {
		t.Fatalf("interpolação project falhou: %s", m["SMTP_REF"])
	}
	if m["API_REF"] != "https://api.example.com" {
		t.Fatalf("interpolação environment falhou: %s", m["API_REF"])
	}
}
