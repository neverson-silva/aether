package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

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
