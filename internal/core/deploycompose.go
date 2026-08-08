package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"aether/internal/domain"
	"aether/internal/runtime/compose"
)

// DeploymentCompose gera o compose específico de um deployment: usa a imagem
// já construída, nome de container fixo e a porta pública.
func (c *Core) DeploymentCompose(app *domain.App, dep *domain.Deployment) (*compose.DeploymentSpec, error) {
	spec, err := c.AppToSpec(app, dep.Number)
	if err != nil {
		return nil, err
	}
	image := dep.ImageRef
	if image == "" {
		image = app.Image
	}
	spec.Image = image
	spec.Build = nil // imagem pré-construída
	spec.Service = compose.ServiceName(app.Name)
	if c.testMode {
		if spec.Labels == nil {
			spec.Labels = map[string]string{}
		}
		spec.Labels["aether.test"] = "1"
	}
	// porta pública: usa a porta configurada se disponível; senão random na execução
	if app.Port <= 0 {
		spec.Ports = nil
		spec.Expose = []string{c.internalPortStr(app, dep.Number)}
	}
	return spec, nil
}

// firstHostPort extrai a porta pública do spec (host port do primeiro mapping).
func firstHostPort(spec *compose.DeploymentSpec) string {
	for _, p := range spec.Ports {
		if p.Host != "" {
			return p.Host
		}
	}
	return ""
}

// composeUp escreve o docker-compose.yml do deployment e executa
// `docker compose up -d`. Retorna o container ID resolvido por nome.
func (c *Core) composeUp(ctx context.Context, app *domain.App, dep *domain.Deployment, spec *compose.DeploymentSpec) (string, string, error) {
	containerName := "aether-" + app.Name + "-" + strconv.FormatInt(dep.Number, 10)
	dir := filepath.Join(c.Cfg.BuildsDir, "compose", app.Name, strconv.FormatInt(dep.Number, 10))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", "", err
	}
	// porta pública: se não configurada, atribui uma livre no host
	if len(spec.Ports) == 0 {
		spec.Ports = []compose.PortMapping{{Host: c.randomFreePort(), Container: c.internalPortStr(app, dep.Number)}}
		spec.Expose = nil
	}
	spec.Service = compose.ServiceName(app.Name)
	file := filepath.Join(dir, "docker-compose.yml")

	doc, err := composeSpecToDoc(spec, containerName)
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(file, []byte(doc), 0o640); err != nil {
		return "", "", err
	}

	// garante network externa
	network := "aether-" + app.ProjectID
	if err := c.Driver.NetworkCreate(ctx, network); err != nil && !strings.Contains(err.Error(), "exists") {
		return "", "", fmt.Errorf("network: %w", err)
	}

	// runtime OCI-genérico (podman ou docker): `podman compose up -d`
	rt := c.Driver.Name()
	cmd := exec.CommandContext(ctx, rt, "compose", "-f", file, "up", "-d")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("%s compose up: %w\n%s", rt, err, strings.TrimSpace(string(out)))
	}

	// resolve o container por nome
	id, err := c.containerIDByName(ctx, containerName)
	return id, containerName, err
}

// composeSpecToDoc serializa um spec com container_name fixo.
func composeSpecToDoc(spec *compose.DeploymentSpec, containerName string) (string, error) {
	clone := *spec
	spec2 := &clone
	return compose.GenerateWith(spec2, containerName)
}

func (c *Core) containerIDByName(ctx context.Context, name string) (string, error) {
	rt := c.Driver.Name()
	out, err := exec.CommandContext(ctx, rt, "inspect", "-f", "{{.Id}}", name).Output()
	if err != nil {
		return "", fmt.Errorf("inspect %s: %w", name, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// composeDown para e remove o stack compose de um deployment.
func (c *Core) composeDown(ctx context.Context, app *domain.App, dep *domain.Deployment) error {
	dir := filepath.Join(c.Cfg.BuildsDir, "compose", app.Name, strconv.FormatInt(dep.Number, 10))
	file := filepath.Join(dir, "docker-compose.yml")
	if _, err := os.Stat(file); err != nil {
		return nil
	}
	cmd := exec.CommandContext(ctx, c.Driver.Name(), "compose", "-f", file, "down", "-v")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compose down: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ComposeDownFor limpa o stack compose do deployment (usado no delete do app).
func (c *Core) ComposeDownFor(app *domain.App, dep *domain.Deployment) error {
	if app == nil || dep == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return c.composeDown(ctx, app, dep)
}
