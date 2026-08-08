package core

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"aether/internal/domain"
	"aether/internal/runtime/compose"
)

// AppToSpec converte um domain.App (mais envs/domains) em um DeploymentSpec
// tipado — a fonte de verdade independente de runtime.
func (c *Core) AppToSpec(app *domain.App, depNumber int64) (*compose.DeploymentSpec, error) {
	if app == nil {
		return nil, fmt.Errorf("app vazio")
	}
	spec := &compose.DeploymentSpec{
		Service:  compose.ServiceName(app.Name),
		Image:    app.Image,
		Restart:  "unless-stopped",
		Runtime:  "compose",
		Strategy: "recreate",
	}
	// source git/upload → build via Dockerfile
	if app.SourceType == domain.SourceGit {
		spec.Build = &compose.BuildSpec{
			Context:    ".",
			Dockerfile: app.Dockerfile,
		}
	}
	// porta interna (container) + exposição pública (host)
	internalPort := c.internalPortStr(app, depNumber)
	hostPort := strconv.Itoa(app.Port)
	if app.Port > 0 {
		spec.Ports = append(spec.Ports, compose.PortMapping{Host: hostPort, Container: internalPort})
	} else {
		spec.Expose = append(spec.Expose, internalPort)
	}

	// environment
	envs, _ := c.EnsureAppEnv(app.ID)
	spec.Environment = map[string]string{}
	for _, line := range envs {
		if i := strings.Index(line, "="); i > 0 {
			spec.Environment[line[:i]] = line[i+1:]
		}
	}

	// volumes
	for _, v := range app.Volumes {
		spec.Volumes = append(spec.Volumes, compose.VolumeSpec{Source: "aether-" + app.Name + "-" + v.Name, Target: v.MountPath})
	}

	// networks
	spec.Networks = []string{"aether-" + app.ProjectID}

	// labels
	spec.Labels = map[string]string{
		"aether.app":  app.ID,
		"aether.name": app.Name,
		"aether.org":  app.OrgID,
		"aether.proj": app.ProjectID,
	}

	// healthcheck
	if app.HealthCheck.Enabled {
		spec.Healthcheck = &compose.HealthcheckSpec{
			Test:     []string{"CMD-SHELL", healthTest(app)},
			Interval: msToCompose(app.HealthCheck.IntervalMS),
			Timeout:  msToCompose(app.HealthCheck.TimeoutMS),
			Retries:  app.HealthCheck.Retries,
		}
	}

	// resources
	if app.Resources.MemMB > 0 || app.Resources.CPUs != "" {
		spec.Resources = &compose.ResourcesSpec{Memory: mbToMem(app.Resources.MemMB)}
		if f, err := strconv.ParseFloat(app.Resources.CPUs, 64); err == nil && f > 0 {
			spec.Resources.CPUs = f
		}
	}

	// domínios
	domains, _ := c.Store.ListDomains(app.ID)
	for _, d := range domains {
		spec.Domains = append(spec.Domains, d.Host)
	}
	return spec, nil
}

// GenerateCompose gera o docker-compose.yml atual (ao vivo) para o app.
func (c *Core) GenerateCompose(app *domain.App) (string, error) {
	var num int64
	if deps, err := c.Store.ListDeployments(app.ID, 1); err == nil && len(deps) > 0 {
		num = deps[0].Number
	}
	spec, err := c.AppToSpec(app, num)
	if err != nil {
		return "", err
	}
	return compose.Generate(spec)
}

// InternalHost retorna o hostname interno estável do serviço na rede
// container-to-container (outros serviços alcançam por este nome).
func (c *Core) InternalHost(app *domain.App) string {
	return compose.ServiceName(app.Name)
}

// InternalNetwork retorna a rede interna do projeto (todos os serviços do
// projeto compartilham a mesma rede e podem se comunicar).
func (c *Core) InternalNetwork(app *domain.App) string {
	return "aether-" + app.ProjectID
}

// ComposeHash calcula um hash estável do compose (para histórico/diff).
func (c *Core) ComposeHash(composeYAML string) string {
	return sha256Hex(composeYAML)
}

func (c *Core) internalPortStr(app *domain.App, depNumber int64) string {
	if plan, err := c.Store.GetDeploymentPlan(app.ID); err == nil && plan.WebServer == "nginx" {
		return "80"
	}
	if depNumber > 0 {
		srcDir := filepath.Join(c.Cfg.BuildsDir, "sources", app.Name, strconv.FormatInt(depNumber, 10))
		if p := readExposePort(filepath.Join(srcDir, app.Dockerfile)); p != "" {
			return p
		}
	}
	return strconv.Itoa(app.Port)
}

func healthTest(app *domain.App) string {
	return "curl -f http://localhost:" + strconv.Itoa(app.Port) + app.HealthCheck.Path + " || exit 1"
}

func msToCompose(ms int) string {
	if ms <= 0 {
		return ""
	}
	return fmt.Sprintf("%dms", ms)
}

func mbToMem(mb int64) string {
	if mb <= 0 {
		return ""
	}
	return fmt.Sprintf("%dM", mb)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum)
}
