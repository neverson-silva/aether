package core

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"aether/internal/domain"
	"aether/internal/obs"
)

func (c *Core) placeOnAgent(app *domain.App) (string, error) {
	servers, err := c.Store.ListServers()
	if err != nil {
		return "", err
	}
	var healthy []domain.Server
	for _, srv := range servers {
		if srv.Status == "healthy" {
			healthy = append(healthy, srv)
		}
	}
	if app.ClusterID != "" {
		var inCluster []domain.Server
		for _, srv := range healthy {
			if srv.ClusterID == app.ClusterID {
				inCluster = append(inCluster, srv)
			}
		}
		if len(inCluster) == 0 {
			return "", nil
		}
		sort.Slice(inCluster, func(i, j int) bool {
			return inCluster[i].Load < inCluster[j].Load
		})
		return inCluster[0].ID, nil
	}
	if app.ServerID != "" {
		for _, srv := range healthy {
			if srv.ID == app.ServerID {
				return app.ServerID, nil
			}
		}
		if len(healthy) == 0 {
			return "", nil
		}
		return app.ServerID, nil
	}
	if len(healthy) == 0 {
		return "", nil
	}
	sort.Slice(healthy, func(i, j int) bool {
		return healthy[i].Load < healthy[j].Load
	})
	return healthy[0].ID, nil
}

func (c *Core) runRemoteDeploy(ctx context.Context, app *domain.App, dep *domain.Deployment, serverID string, ll *obs.LiveLog) {
	defer ll.Close()
	dep.Status = domain.DeploymentBuilding
	_ = c.Store.UpdateDeployment(dep)

	if app.SourceType != domain.SourceImage {
		msg := "deploy remoto suporta apenas apps de imagem nesta versão"
		dep.Status = domain.DeploymentFailed
		dep.Error = msg
		dep.FinishedAt = time.Now().UTC()
		_ = c.Store.UpdateDeployment(dep)
		ll.Write([]byte("[deploy] " + msg + "\n"))
		c.failDeployment(dep, nil)
		return
	}
	if app.Image == "" {
		c.failDeployment(dep, nil)
		return
	}
	dep.ImageRef = app.Image
	_ = c.Store.UpdateDeployment(dep)
	ll.Write([]byte("[deploy] remoto: " + app.Image + " -> " + serverID + "\n"))

	payload := map[string]any{
		"deployment_id": dep.ID,
		"app": map[string]any{
			"id":           app.ID,
			"name":         app.Name,
			"image":        app.Image,
			"port":         app.Port,
			"project_id":   app.ProjectID,
			"env":          c.mustEnv(app.ID),
			"volumes":      app.Volumes,
			"resources":    app.Resources,
			"health_check": app.HealthCheck,
			"mem_mb":       app.Resources.MemMB,
			"cpus":         app.Resources.CPUs,
			"server_id":    serverID,
		},
	}
	raw, _ := json.Marshal(payload)
	if err := c.Store.EnqueueServerCommand(serverID, "deploy", string(raw)); err != nil {
		c.failDeployment(dep, err)
		return
	}
	ll.Write([]byte("[deploy] comando enviado ao servidor\n"))
}

func (c *Core) mustEnv(appID string) []string {
	env, err := c.EnsureAppEnv(appID)
	if err != nil {
		return nil
	}
	return env
}
