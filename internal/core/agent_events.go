package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"aether/internal/domain"
)

func (c *Core) applyAgentEvent(serverID string, ev agentEvent) {
	switch ev.Type {
	case "deploy.log":
		depID, _ := ev.Payload["deployment_id"].(string)
		line, _ := ev.Payload["line"].(string)
		c.writeRemoteLog(depID, line)
	case "deploy.ready":
		depID, _ := ev.Payload["deployment_id"].(string)
		containerID, _ := ev.Payload["container_id"].(string)
		dep, err := c.Store.GetDeployment(depID)
		if err != nil {
			return
		}
		dep.Status = domain.DeploymentReady
		dep.ContainerID = containerID
		dep.FinishedAt = time.Now().UTC()
		_ = c.Store.UpdateDeployment(dep)
		app, aerr := c.Store.GetApp(dep.AppID)
		if aerr == nil {
			c.Bus.Publish(context.Background(), "app", app.ID, "app.deployed", map[string]any{
				"deployment_id": dep.ID, "number": dep.Number, "image": dep.ImageRef, "server": serverID,
			}, nil)
		}
		c.writeRemoteLog(depID, "[deploy] pronto no servidor "+serverID+"\n")
		c.FireWebhookEvent(context.Background(), appOrgID(c, dep.AppID), EvDeployReady, map[string]any{
			"app": appName(c, dep.AppID), "app_id": dep.AppID, "build": dep.Number, "server": serverID,
		})
	case "deploy.failed":
		depID, _ := ev.Payload["deployment_id"].(string)
		msg, _ := ev.Payload["error"].(string)
		dep, err := c.Store.GetDeployment(depID)
		if err != nil {
			return
		}
		dep.Status = domain.DeploymentFailed
		dep.Error = msg
		dep.FinishedAt = time.Now().UTC()
		_ = c.Store.UpdateDeployment(dep)
		c.writeRemoteLog(depID, "[deploy] falhou no servidor "+serverID+": "+msg+"\n")
		app, aerr := c.Store.GetApp(dep.AppID)
		if aerr == nil {
			c.NotifyOrg(app.OrgID, "Deploy failed: "+app.Name, fmt.Sprintf("#%d: %s", dep.Number, msg))
			c.FireWebhookEvent(context.Background(), app.OrgID, EvDeployFailed, map[string]any{
				"app": app.Name, "app_id": app.ID, "build": dep.Number, "error": msg,
			})
		}
	}
}

func (c *Core) writeRemoteLog(depID, line string) {
	dep, err := c.Store.GetDeployment(depID)
	if err != nil {
		return
	}
	app, err := c.Store.GetApp(dep.AppID)
	if err != nil {
		return
	}
	dir := filepath.Join(c.Cfg.LogsDir, "apps", app.Name)
	_ = os.MkdirAll(dir, 0o750)
	path := filepath.Join(dir, strconv.FormatInt(dep.Number, 10)+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(line)
}

func appOrgID(c *Core, appID string) string {
	app, err := c.Store.GetApp(appID)
	if err != nil {
		return ""
	}
	return app.OrgID
}

func appName(c *Core, appID string) string {
	app, err := c.Store.GetApp(appID)
	if err != nil {
		return ""
	}
	return app.Name
}
