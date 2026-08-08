package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"

	"aether/internal/domain"
	"aether/internal/git"
)

func (c *Core) StartGitOpsWatch(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.syncAllGitOps(ctx)
			}
		}
	}()
}

func (c *Core) CreateGitOps(orgID, name, repoURL, branch, path, applyMode string) (*domain.GitOps, error) {
	g := &domain.GitOps{
		ID:         "go-" + domain.NewID(),
		OrgID:      orgID,
		Name:       name,
		RepoURL:    repoURL,
		Branch:     branch,
		Path:       path,
		ApplyMode:  applyMode,
		LastStatus: "pending",
		CreatedAt:  time.Now().UTC(),
	}
	if g.Branch == "" {
		g.Branch = "main"
	}
	if g.Path == "" {
		g.Path = "aether.yml"
	}
	if err := c.Store.CreateGitOps(g); err != nil {
		return nil, err
	}
	return g, nil
}

func (c *Core) SyncGitOps(g *domain.GitOps) error {
	dir := filepath.Join(c.Cfg.CacheDir, "gitops", g.ID)
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o750); err != nil {
		return err
	}
	if err := git.Clone(context.Background(), g.RepoURL, g.Branch, dir); err != nil {
		return fmt.Errorf("clone: %w", err)
	}
	sha, err := git.CommitHEAD(context.Background(), dir)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, g.Path)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("aether.yml não encontrado em %s: %w", g.Path, err)
	}
	var doc exportDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("yaml inválido: %w", err)
	}
	g.LastSHA = sha

	desired := map[string]bool{}
	desiredNames := map[string]bool{}
	for _, p := range doc.Projects {
		for _, ea := range p.Apps {
			desired[p.Name+"|"+ea.Name] = true
			desiredNames[ea.Name] = true
		}
	}

	var targetOrgID string
	if g.TargetOrgID == "" {
		org, err := c.ensureGitOpsOrg(g)
		if err != nil {
			return err
		}
		targetOrgID = org
		g.TargetOrgID = org
	} else {
		targetOrgID = g.TargetOrgID
	}

	added, changed, removed := 0, 0, 0
	existing := map[string]bool{}
	projects, _ := c.Store.ListProjects(targetOrgID)
	for _, p := range projects {
		apps, _ := c.Store.ListAppsByProject(p.ID)
		for _, a := range apps {
			key := p.Name + "|" + a.Name
			existing[key] = true
			if desired[key] {
				changed++
			} else if desiredNames[a.Name] {
				changed++
			} else {
				removed++
			}
		}
	}
	for key := range desired {
		if !existing[key] {
			added++
		}
	}
	g.DriftAdded, g.DriftChanged, g.DriftRemoved = added, changed, removed
	g.LastSync = time.Now().UTC().Format(time.RFC3339)
	g.LastStatus = "synced"
	if err := c.Store.UpdateGitOps(g); err != nil {
		return err
	}
	if g.ApplyMode == "auto" && (added > 0 || changed > 0) {
		if err := c.reconcileGitOps(targetOrgID, &doc); err != nil {
			log.Printf("[gitops] %s apply: %v", g.Name, err)
			g.LastStatus = "apply_failed"
			c.Store.UpdateGitOps(g)
			return err
		}
		g.LastStatus = "applied"
		g.DriftAdded, g.DriftChanged, g.DriftRemoved = 0, 0, 0
		c.Store.UpdateGitOps(g)
	}
	return nil
}

func (c *Core) reconcileGitOps(targetOrgID string, doc *exportDoc) error {
	for _, p := range doc.Projects {
		project, err := c.Store.GetProjectByOrgName(targetOrgID, p.Name)
		if err != nil {
			project, err = c.CreateProject(targetOrgID, p.Name)
			if err != nil {
				return err
			}
		}
		for _, ea := range p.Apps {
			app := &domain.App{
				ID:          domain.NewID(),
				OrgID:       targetOrgID,
				ProjectID:   project.ID,
				Name:        ea.Name,
				SourceType:  domain.SourceType(ea.SourceType),
				Image:       ea.Image,
				GitURL:      ea.GitURL,
				GitBranch:   ea.GitBranch,
				Dockerfile:  ea.Dockerfile,
				BuildType:   ea.BuildType,
				Port:        ea.Port,
				HealthCheck: domain.HealthCheck{Enabled: true, Path: "/"},
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
			existing, err := c.Store.GetAppByOrgName(targetOrgID, ea.Name)
			if err != nil {
				if err := c.CreateApp(targetOrgID, app); err != nil {
					return err
				}
				if err := c.applyGitOpsEnv(app, ea.Env); err != nil {
					return err
				}
				continue
			}
			existing.Image = app.Image
			existing.GitURL = app.GitURL
			existing.GitBranch = app.GitBranch
			existing.Dockerfile = app.Dockerfile
			existing.Port = app.Port
			if err := c.Store.UpdateApp(existing); err != nil {
				return err
			}
			if err := c.applyGitOpsEnv(existing, ea.Env); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Core) applyGitOpsEnv(app *domain.App, env map[string]string) error {
	for k, v := range env {
		if err := c.Store.SetEnvVar(app.ID, k, v, false); err != nil {
			return err
		}
	}
	return nil
}

func (c *Core) ensureGitOpsOrg(g *domain.GitOps) (string, error) {
	orgs, err := c.Store.ListOrgs()
	if err != nil {
		return "", err
	}
	for _, o := range orgs {
		if o.Name == "gitops-"+g.Name {
			return o.ID, nil
		}
	}
	o := &domain.Org{
		ID:          "org-" + domain.NewID(),
		Name:        "gitops-" + g.Name,
		OwnerUserID: g.OrgID,
		CreatedAt:   time.Now().UTC(),
	}
	if err := c.Store.CreateOrg(o); err != nil {
		return "", err
	}
	return o.ID, nil
}

func (c *Core) syncAllGitOps(ctx context.Context) {
	orgs, _ := c.Store.ListOrgs()
	for _, o := range orgs {
		list, err := c.Store.ListGitOps(o.ID)
		if err != nil {
			continue
		}
		for i := range list {
			g := &list[i]
			if err := c.SyncGitOps(g); err != nil {
				log.Printf("[gitops] %s: %v", g.Name, err)
			}
		}
	}
}

func (c *Core) DeleteGitOps(id string) error {
	return c.Store.DeleteGitOps(id)
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
