package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	appsdomain "aether/internal/apps/domain"
	dbdomain "aether/internal/databases/domain"
	domainsdomain "aether/internal/domains/domain"
	"aether/internal/orgs/domain"
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
	CPUs          string            `yaml:"cpus,omitempty"`
	MemMB         int               `yaml:"mem_mb,omitempty"`
	HealthCheck   map[string]any    `yaml:"health_check,omitempty"`
	Env           map[string]string `yaml:"env,omitempty"`
	Domains       []string          `yaml:"domains,omitempty"`
}

type exportDB struct {
	Name    string `yaml:"name"`
	Engine  string `yaml:"engine"`
	Version string `yaml:"version,omitempty"`
	MemMB   int    `yaml:"mem_mb,omitempty"`
}

type exportDoc struct {
	Version  string          `yaml:"version"`
	Projects []exportProject `yaml:"projects"`
}

type ExportAppStore interface {
	ListProjects(ctx context.Context, orgID uuid.UUID) ([]appsdomain.Project, error)
	ListAppsByProject(ctx context.Context, orgID, projectID uuid.UUID) ([]appsdomain.App, error)
	ListEnvVars(ctx context.Context, appID uuid.UUID) ([]appsdomain.EnvVar, error)
}

type DomainLister interface {
	ListDomains(ctx context.Context, appID uuid.UUID) ([]domainsdomain.Domain, error)
}

type ExportDatabaseStore interface {
	ListDatabasesByOrg(ctx context.Context, orgID uuid.UUID) ([]dbdomain.Database, error)
}

func (o *Organizations) Export(ctx context.Context, orgID uuid.UUID) ([]byte, error) {
	exportApps, ok := o.Apps.(ExportAppStore)
	if !ok {
		return nil, domain.ErrValidation
	}
	projects, err := exportApps.ListProjects(ctx, orgID)
	if err != nil {
		return nil, err
	}
	doc := exportDoc{Version: "1"}
	for _, p := range projects {
		ep := exportProject{Name: p.Name}
		apps, err := exportApps.ListAppsByProject(ctx, orgID, p.ID)
		if err != nil {
			return nil, err
		}
		for _, a := range apps {
			ea := exportApp{
				Name: a.Name, SourceType: a.SourceType, Image: a.Image, GitURL: a.GitURL,
				GitBranch: a.GitBranch, Dockerfile: a.Dockerfile, BuildType: a.BuildType,
				PreviewDomain: a.PreviewDomain, Port: a.Port, CPUs: a.CPUs, MemMB: a.MemMB,
				HealthCheck: map[string]any{
					"enabled": a.HealthCheck.Enabled, "path": a.HealthCheck.Path,
				},
				Env: map[string]string{},
			}
			envs, err := exportApps.ListEnvVars(ctx, a.ID)
			if err == nil {
				for _, e := range envs {
					ea.Env[e.Name] = e.Value
				}
			}
			if o.Domains != nil {
				domains, err := o.Domains.ListDomains(ctx, a.ID)
				if err == nil {
					for _, d := range domains {
						ea.Domains = append(ea.Domains, d.Host)
					}
				}
			}
			ep.Apps = append(ep.Apps, ea)
		}
		if o.Databases != nil {
			dbs, err := o.Databases.ListDatabasesByOrg(ctx, orgID)
			if err == nil {
				for _, d := range dbs {
					if d.ProjectID != p.ID {
						continue
					}
					ep.Databases = append(ep.Databases, exportDB{Name: d.Name, Engine: string(d.Engine), Version: d.Version, MemMB: d.MemMB})
				}
			}
		}
		doc.Projects = append(doc.Projects, ep)
	}
	return yaml.Marshal(doc)
}

func (o *Organizations) Import(ctx context.Context, orgID uuid.UUID, data []byte) error {
	var doc exportDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("invalid yaml: %w", err)
	}
	if len(doc.Projects) == 0 {
		return errors.New("no projects to import")
	}
	projectStore, ok := o.Apps.(ProjectCreator)
	if !ok {
		return domain.ErrValidation
	}
	for _, p := range doc.Projects {
		project, err := projectStore.CreateProject(ctx, orgID, p.Name, slugify(p.Name), "", "")
		if err != nil {
			return err
		}
		appStore, ok := o.Apps.(AppCreator)
		if ok {
			for _, ea := range p.Apps {
				sourceType := ea.SourceType
				if sourceType == "" {
					sourceType = "image"
				}
				gitBranch := ea.GitBranch
				if gitBranch == "" {
					gitBranch = "main"
				}
				dockerfile := ea.Dockerfile
				if dockerfile == "" {
					dockerfile = "Dockerfile"
				}
				port := ea.Port
				if port == 0 {
					port = 80
				}
				app, err := appStore.CreateApp(ctx, &appsdomain.App{
					OrgID: orgID, ProjectID: project.ID, Name: ea.Name, SourceType: sourceType, Image: ea.Image,
					GitURL: ea.GitURL, GitBranch: gitBranch, Dockerfile: dockerfile,
					BuildType: ea.BuildType, PreviewDomain: ea.PreviewDomain, Port: port,
					CPUs: ea.CPUs, MemMB: ea.MemMB,
					HealthCheck: appsdomain.HealthCheck{Enabled: true, Path: "/"},
				})
				if err != nil {
					return err
				}
				envStore, ok := o.Apps.(EnvVarSetter)
				if ok {
					for k, v := range ea.Env {
						if err := envStore.UpsertEnvVar(ctx, app.ID, k, v, false); err != nil {
							return err
						}
					}
				}
			}
		}
		if o.Databases != nil {
			dbStore, ok := o.Databases.(DatabaseCreator)
			if ok {
				for _, ed := range p.Databases {
					if _, err := dbStore.CreateDatabase(ctx, &dbdomain.Database{
						OrgID: orgID, ProjectID: project.ID, Name: ed.Name,
						Engine: dbdomain.Engine(ed.Engine), Version: ed.Version, MemMB: ed.MemMB,
					}); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

type ProjectCreator interface {
	CreateProject(ctx context.Context, orgID uuid.UUID, name, slug, description, color string) (*appsdomain.Project, error)
}

type AppCreator interface {
	CreateApp(ctx context.Context, app *appsdomain.App) (*appsdomain.App, error)
}

type EnvVarSetter interface {
	UpsertEnvVar(ctx context.Context, appID uuid.UUID, name, value string, secret bool) error
}

type DatabaseCreator interface {
	CreateDatabase(ctx context.Context, db *dbdomain.Database) (*dbdomain.Database, error)
}
