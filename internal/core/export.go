package core

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"aether/internal/domain"
)

type exportProject struct {
	Name      string      `yaml:"name"`
	Apps      []exportApp `yaml:"apps,omitempty"`
	Databases []exportDB  `yaml:"databases,omitempty"`
}

type exportApp struct {
	Name          string            `yaml:"name"`
	SourceType    string            `yaml:"source_type"`
	Image         string            `yaml:"image,omitempty"`
	GitURL        string            `yaml:"git_url,omitempty"`
	GitBranch     string            `yaml:"git_branch,omitempty"`
	Dockerfile    string            `yaml:"dockerfile,omitempty"`
	BuildType     string            `yaml:"build_type,omitempty"`
	PreviewDomain string            `yaml:"preview_domain,omitempty"`
	Port          int               `yaml:"port,omitempty"`
	Resources     map[string]any    `yaml:"resources,omitempty"`
	HealthCheck   map[string]any    `yaml:"health_check,omitempty"`
	Env           map[string]string `yaml:"env,omitempty"`
	Domains       []string          `yaml:"domains,omitempty"`
}

type exportDB struct {
	Name    string `yaml:"name"`
	Engine  string `yaml:"engine"`
	Version string `yaml:"version,omitempty"`
	MemMB   int64  `yaml:"mem_mb,omitempty"`
}

type exportDoc struct {
	Version  string          `yaml:"version"`
	Projects []exportProject `yaml:"projects"`
}

func (c *Core) ExportOrg(orgID string) ([]byte, error) {
	projects, err := c.Store.ListProjects(orgID)
	if err != nil {
		return nil, err
	}
	doc := exportDoc{Version: "1"}
	for _, p := range projects {
		ep := exportProject{Name: p.Name}
		apps, err := c.Store.ListApps(orgID)
		if err != nil {
			return nil, err
		}
		for _, a := range apps {
			if a.ProjectID != p.ID {
				continue
			}
			ea := exportApp{
				Name:          a.Name,
				SourceType:    string(a.SourceType),
				Image:         a.Image,
				GitURL:        a.GitURL,
				GitBranch:     a.GitBranch,
				Dockerfile:    a.Dockerfile,
				BuildType:     a.BuildType,
				PreviewDomain: a.PreviewDomain,
				Port:          a.Port,
				Resources: map[string]any{
					"cpus": a.Resources.CPUs, "mem_mb": a.Resources.MemMB,
				},
				HealthCheck: map[string]any{
					"enabled": a.HealthCheck.Enabled, "path": a.HealthCheck.Path,
				},
				Env: map[string]string{},
			}
			envs, err := c.Store.ListEnvVars(a.ID)
			if err == nil {
				for _, e := range envs {
					if e.Secret {
						if v, err := c.Secrets.DecryptString(string(e.Value)); err == nil {
							ea.Env[e.Name] = v
						}
					} else {
						ea.Env[e.Name] = string(e.Value)
					}
				}
			}
			domains, err := c.Store.ListDomains(a.ID)
			if err == nil {
				for _, d := range domains {
					ea.Domains = append(ea.Domains, d.Host)
				}
			}
			ep.Apps = append(ep.Apps, ea)
		}
		dbs, err := c.Store.ListDatabases(orgID)
		if err == nil {
			for _, d := range dbs {
				if d.ProjectID != p.ID {
					continue
				}
				ep.Databases = append(ep.Databases, exportDB{Name: d.Name, Engine: string(d.Engine), Version: d.Version, MemMB: d.MemMB})
			}
		}
		doc.Projects = append(doc.Projects, ep)
	}
	return yaml.Marshal(doc)
}

func (c *Core) ImportOrg(orgID string, data []byte) error {
	var doc exportDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("yaml inválido: %w", err)
	}
	for _, p := range doc.Projects {
		project, err := c.CreateProject(orgID, p.Name)
		if err != nil {
			return err
		}
		for _, ea := range p.Apps {
			app := &domain.App{
				ID:            domain.NewID(),
				ProjectID:     project.ID,
				Name:          ea.Name,
				SourceType:    domain.SourceType(ea.SourceType),
				Image:         ea.Image,
				GitURL:        ea.GitURL,
				GitBranch:     ea.GitBranch,
				Dockerfile:    ea.Dockerfile,
				BuildType:     ea.BuildType,
				PreviewDomain: ea.PreviewDomain,
				Port:          ea.Port,
				HealthCheck:   domain.HealthCheck{Enabled: true, Path: "/"},
			}
			if app.SourceType == "" {
				app.SourceType = domain.SourceImage
			}
			if app.GitBranch == "" {
				app.GitBranch = "main"
			}
			if app.Dockerfile == "" {
				app.Dockerfile = "Dockerfile"
			}
			if app.Port == 0 {
				app.Port = 80
			}
			if err := c.CreateApp(orgID, app); err != nil {
				return err
			}
			for k, v := range ea.Env {
				if err := c.SetAppEnv(app.ID, k, v, false); err != nil {
					return err
				}
			}
			for _, host := range ea.Domains {
				if _, err := c.Store.GetDomainByHost(host); err == nil {
					continue
				}
				if err := c.CreateDomain(app.ID, host, false); err != nil {
					return err
				}
			}
		}
		for _, ed := range p.Databases {
			if _, err := c.CreateDatabase(orgID, project.ID, ed.Name, domain.DBEngine(ed.Engine), ed.Version, ed.MemMB, 0); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Core) ExportToFile(orgID, path string) error {
	data, err := c.ExportOrg(orgID)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o640)
}

func (c *Core) ImportFromFile(orgID, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return c.ImportOrg(orgID, data)
}
