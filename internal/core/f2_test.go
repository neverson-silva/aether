package core

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"aether/internal/domain"
)

func testCore(t *testing.T) *Core {
	cfg := testConfig(t)
	cfg.ChallengeAddr = "127.0.0.1:0"
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	c.testMode = true
	t.Cleanup(cleanupTestNetworks)
	return c
}

func cleanupTestNetworks() {
	out, err := exec.Command("podman", "ps", "-a", "--filter", "label=aether.test=1", "--format", "{{.ID}}").Output()
	if err == nil {
		for _, id := range strings.Fields(string(out)) {
			_ = exec.Command("podman", "rm", "-f", id).Run()
		}
		time.Sleep(500 * time.Millisecond)
	}
	out, err = exec.Command("podman", "network", "ls", "--format", "{{.Name}}").Output()
	if err == nil {
		for _, name := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(name, "aether-") {
				_ = exec.Command("podman", "network", "rm", name).Run()
			}
		}
	}
}

func TestTOTPFlow(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	user, _, err := c.CreateUserAndOrg("totp@aether.local", "totp", "senha-totp")
	if err != nil {
		t.Fatal(err)
	}
	enabled, err := c.TOTPEnabled(user.ID)
	if err != nil || enabled {
		t.Fatal("MFA deveria estar desabilitado")
	}
	totp, err := c.EnrollTOTP(user.ID)
	if err != nil || totp.Secret == "" {
		t.Fatalf("enroll: %v %v", totp, err)
	}
	code := totpCode(totp.Secret, time.Now().Unix())
	if code == "" {
		t.Fatal("código vazio")
	}
	ok, err := c.VerifyTOTP(user.ID, code)
	if err != nil || ok {
		t.Fatalf("MFA desabilitado deveria recusar verificação: %v %v", ok, err)
	}
	if err := c.EnableTOTP(user.ID, code); err != nil {
		t.Fatal(err)
	}
	enabled, _ = c.TOTPEnabled(user.ID)
	if !enabled {
		t.Fatal("MFA deveria estar habilitado")
	}
	ok, err = c.VerifyTOTP(user.ID, code)
	if err != nil || !ok {
		t.Fatalf("verificação deveria passar: %v %v", ok, err)
	}
	ok, _ = c.VerifyTOTP(user.ID, "000000")
	if ok {
		t.Fatal("código errado deveria falhar")
	}
	if _, _, err := c.Login("totp@aether.local", "senha-totp", ""); err != ErrMFARequired {
		t.Fatalf("login sem código deveria exigir MFA: %v", err)
	}
	if _, _, err := c.Login("totp@aether.local", "senha-totp", totpCode(totp.Secret, time.Now().Unix())); err != nil {
		t.Fatalf("login com código deveria passar: %v", err)
	}
	if err := c.DisableTOTP(user.ID); err != nil {
		t.Fatal(err)
	}
	enabled, _ = c.TOTPEnabled(user.ID)
	if enabled {
		t.Fatal("MFA deveria estar desabilitado")
	}
}

func TestCronNext(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	next := c.nextCron("*/5 * * * *", now)
	if !next.After(now) {
		t.Fatalf("next deveria ser futuro: %v", next)
	}
	if next.Minute() != 5 {
		t.Fatalf("minuto errado: %v", next.Minute())
	}
	next = c.nextCron("0 2 * * *", now)
	if next.Hour() != 2 || next.Day() != 4 {
		t.Fatalf("cron diário errado: %v", next)
	}
}

func TestExportImportRoundtrip(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	user, org, err := c.CreateUserAndOrg("exp@aether.local", "exp", "senha-exp")
	if err != nil {
		t.Fatal(err)
	}
	_ = user
	project, err := c.CreateProject(org.ID, "demo")
	if err != nil {
		t.Fatal(err)
	}
	app := &domain.App{
		ID:         domain.NewID(),
		ProjectID:  project.ID,
		Name:       "web",
		SourceType: domain.SourceImage,
		Image:      "nginx:alpine",
		Port:       80,
	}
	if err := c.CreateApp(org.ID, app); err != nil {
		t.Fatal(err)
	}
	if err := c.SetAppEnv(app.ID, "MODE", "test", false); err != nil {
		t.Fatal(err)
	}
	if err := c.CreateDomain(app.ID, "web.example.com", false); err != nil {
		t.Fatal(err)
	}

	data, err := c.ExportOrg(org.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "web.example.com") || !strings.Contains(string(data), "MODE") {
		t.Fatalf("export incompleto:\n%s", string(data))
	}

	org2 := &domain.Org{ID: domain.NewID(), Name: "imp", OwnerUserID: user.ID, CreatedAt: time.Now().UTC()}
	if err := c.Store.CreateOrg(org2); err != nil {
		t.Fatal(err)
	}
	if err := c.Store.CreateMember(&domain.Member{OrgID: org2.ID, UserID: user.ID, Role: domain.RoleOwner}); err != nil {
		t.Fatal(err)
	}
	if err := c.ImportOrg(org2.ID, data); err != nil {
		t.Fatal(err)
	}
	apps, err := c.Store.ListApps(org2.ID)
	if err != nil || len(apps) != 1 {
		t.Fatalf("apps importados: %v %v", apps, err)
	}
	if apps[0].Name != "web" || apps[0].Image != "nginx:alpine" {
		t.Fatalf("app importado errado: %+v", apps[0])
	}
	envs, _ := c.Store.ListEnvVars(apps[0].ID)
	if len(envs) != 1 || envs[0].Name != "MODE" {
		t.Fatalf("env importada errado: %+v", envs)
	}
	domains, _ := c.Store.ListDomains(apps[0].ID)
	if len(domains) != 0 {
		t.Fatalf("domínio global já existente deveria ser pulado: %+v", domains)
	}
}

func TestTemplateInstall(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	if err := c.SeedTemplates(); err != nil {
		t.Fatal(err)
	}
	templates, err := c.ListTemplates()
	if err != nil || len(templates) == 0 {
		t.Fatalf("templates: %v %v", templates, err)
	}
	tpl, err := c.GetTemplate("tpl-postgresql")
	if err != nil {
		t.Fatal(err)
	}
	user, org, err := c.CreateUserAndOrg("tpl@aether.local", "tpl", "senha-tpl")
	if err != nil {
		t.Fatal(err)
	}
	_ = user
	project, err := c.CreateProject(org.ID, "cat")
	if err != nil {
		t.Fatal(err)
	}
	stack, err := c.InstallTemplate(tpl.ID, org.ID, project.ID, "pg", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stack.ComposeYAML, "postgres:16") {
		t.Fatalf("compose do template errado:\n%s", stack.ComposeYAML)
	}
}

func TestComposeParse(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	user, org, err := c.CreateUserAndOrg("cp@aether.local", "cp", "senha-cp")
	if err != nil {
		t.Fatal(err)
	}
	_ = user
	project, err := c.CreateProject(org.ID, "st")
	if err != nil {
		t.Fatal(err)
	}
	bad := "services:"
	if _, err := c.SaveCompose(org.ID, project.ID, "bad", bad); err == nil {
		t.Fatal("compose sem serviços deveria falhar")
	}
	good := `services:
  web:
    image: nginx:alpine
    ports:
      - 8080:80
  worker:
    image: busybox
    command: sh -c "sleep 3600"
`
	stack, err := c.SaveCompose(org.ID, project.ID, "stack1", good)
	if err != nil {
		t.Fatal(err)
	}
	if stack.Status != "stopped" {
		t.Fatalf("status: %s", stack.Status)
	}
	stacks, err := c.ListCompose(org.ID)
	if err != nil || len(stacks) != 1 {
		t.Fatalf("stacks: %v %v", stacks, err)
	}
}

func timeoutCtxT() context.Context {
	return context.Background()
}
