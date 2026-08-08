package core

import (
	"errors"
	"testing"

	"aether/internal/domain"
)

func TestProjectCreatesProductionEnv(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	_, org, err := c.CreateUserAndOrg("env@aether.local", "env", "senha-env")
	if err != nil {
		t.Fatal(err)
	}
	proj, err := c.CreateProject(org.ID, "app1")
	if err != nil {
		t.Fatal(err)
	}
	envs, err := c.ListEnvironments(proj.ID)
	if err != nil || len(envs) != 1 {
		t.Fatalf("deveria ter 1 env: %d %v", len(envs), err)
	}
	if envs[0].Name != "production" || !envs[0].IsDefault {
		t.Fatalf("env default: %+v", envs[0])
	}
}

func TestEnvironmentCRUDRules(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	_, org, err := c.CreateUserAndOrg("env2@aether.local", "env2", "senha-env2")
	if err != nil {
		t.Fatal(err)
	}
	proj, _ := c.CreateProject(org.ID, "app2")

	staging, err := c.CreateEnvironment(proj.ID, "Staging", "pre-prod", "#ff8800")
	if err != nil {
		t.Fatal(err)
	}
	if staging.Slug != "staging" || staging.IsDefault {
		t.Fatalf("staging não deveria ser default: %+v", staging)
	}

	if _, err := c.CreateEnvironment(proj.ID, "staging", "", ""); !errors.Is(err, ErrEnvExists) {
		t.Fatalf("duplicado deveria falhar: %v", err)
	}

	if err := c.SetDefaultEnvironment(proj.ID, staging.ID); err != nil {
		t.Fatal(err)
	}
	envs, _ := c.ListEnvironments(proj.ID)
	defaults := 0
	for _, e := range envs {
		if e.IsDefault {
			defaults++
		}
	}
	if defaults != 1 || envs[0].IsDefault != true || envs[0].ID != staging.ID {
		t.Fatalf("default errado: %+v", envs)
	}

	prod := envs[len(envs)-1]
	if err := c.DeleteEnvironment(proj.ID, staging.ID); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteEnvironment(proj.ID, prod.ID); !errors.Is(err, ErrEnvLast) {
		t.Fatalf("último env deveria falhar: %v", err)
	}
}

func TestDeleteEnvWithServicesBlocked(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	_, org, err := c.CreateUserAndOrg("env3@aether.local", "env3", "senha-env3")
	if err != nil {
		t.Fatal(err)
	}
	proj, _ := c.CreateProject(org.ID, "app3")
	envs, _ := c.ListEnvironments(proj.ID)
	def := envs[0]
	staging, _ := c.CreateEnvironment(proj.ID, "staging", "", "")

	app := &domain.App{
		ID:            domain.NewID(),
		OrgID:         org.ID,
		ProjectID:     proj.ID,
		EnvironmentID: staging.ID,
		Name:          "web",
	}
	if err := c.CreateApp(org.ID, app); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteEnvironment(proj.ID, staging.ID); !errors.Is(err, ErrEnvHasServices) {
		t.Fatalf("env com serviços deveria falhar: %v", err)
	}
	if err := c.DeleteEnvironment(proj.ID, def.ID); err != nil {
		t.Fatalf("production (sem serviços) deveria deletar: %v", err)
	}
	if err := c.DeleteEnvironment(proj.ID, staging.ID); !errors.Is(err, ErrEnvLast) {
		t.Fatalf("último env deveria falhar: %v", err)
	}
}

func TestDefaultEnvironmentUsedOnAppCreate(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	_, org, err := c.CreateUserAndOrg("env4@aether.local", "env4", "senha-env4")
	if err != nil {
		t.Fatal(err)
	}
	proj, _ := c.CreateProject(org.ID, "app4")
	staging, _ := c.CreateEnvironment(proj.ID, "staging", "", "")
	c.SetDefaultEnvironment(proj.ID, staging.ID)

	app := &domain.App{
		ID:        domain.NewID(),
		OrgID:     org.ID,
		ProjectID: proj.ID,
		Name:      "web",
	}
	if err := c.CreateApp(org.ID, app); err != nil {
		t.Fatal(err)
	}
	got, _ := c.Store.GetApp(app.ID)
	if got.EnvironmentID != staging.ID {
		t.Fatalf("app deveria ir para o env default (staging): %s", got.EnvironmentID)
	}
}
