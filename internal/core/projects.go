package core

import (
	"context"
	"errors"
	"time"

	"aether/internal/domain"
)

func (c *Core) RenameProject(orgID, projectID, name string) (*domain.Project, error) {
	project, err := c.Store.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	if project.OrgID != orgID {
		return nil, errors.New("projeto não pertence à organização")
	}
	if err := c.Store.UpdateProject(projectID, name); err != nil {
		return nil, err
	}
	project.Name = name
	return project, nil
}

func (c *Core) DeleteProject(orgID, projectID string) error {
	project, err := c.Store.GetProject(projectID)
	if err != nil {
		return err
	}
	if project.OrgID != orgID {
		return errors.New("projeto não pertence à organização")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	rows, err := c.DB.Query(`SELECT id FROM compose_apps WHERE project_id=?`, projectID)
	if err == nil {
		var ids []string
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				ids = append(ids, id)
			}
		}
		rows.Close()
		for _, id := range ids {
			_ = c.DeleteCompose(id)
		}
	}

	dbs, _ := c.Store.ListDatabases(orgID)
	for _, db := range dbs {
		if db.ProjectID == projectID {
			_ = c.DeleteDatabase(db.ID)
		}
	}

	apps, _ := c.Store.ListAppsByProject(projectID)
	for _, app := range apps {
		deploys, _ := c.Store.ListDeployments(app.ID, 1)
		if len(deploys) > 0 && deploys[0].ContainerID != "" {
			_ = c.Driver.Remove(ctx, deploys[0].ContainerID, true)
		}
		_ = c.Store.DeleteApp(app.ID)
	}

	_, _ = c.DB.Exec(`DELETE FROM environments WHERE project_id=?`, projectID)
	return c.Store.DeleteProject(projectID)
}
