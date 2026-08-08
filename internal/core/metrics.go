package core

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"aether/internal/domain"
	"time"
)

func (c *Core) MetricsText(ctx context.Context) string {
	var b strings.Builder
	b.WriteString("# HELP aether_info Aether platform info.\n# TYPE aether_info gauge\n")
	b.WriteString(fmt.Sprintf("aether_info{version=%q} 1\n", "1.0.0"))

	orgs, _ := c.Store.ListOrgs()
	var apps []domain.App
	for _, o := range orgs {
		list, _ := c.Store.ListApps(o.ID)
		apps = append(apps, list...)
	}
	b.WriteString("# HELP aether_apps_total Total apps.\n# TYPE aether_apps_total gauge\n")
	b.WriteString(fmt.Sprintf("aether_apps_total %d\n", len(apps)))

	projs := 0
	for _, o := range orgs {
		if ps, err := c.Store.ListProjects(o.ID); err == nil {
			projs += len(ps)
		}
	}
	b.WriteString("# HELP aether_projects_total Total projects.\n# TYPE aether_projects_total gauge\n")
	b.WriteString(fmt.Sprintf("aether_projects_total %d\n", projs))

	b.WriteString("# HELP aether_deployments_total Deployments per status.\n# TYPE aether_deployments_total gauge\n")
	statuses := map[string]int{}
	for _, a := range apps {
		deploys, _ := c.Store.ListDeployments(a.ID, 100000)
		for _, d := range deploys {
			statuses[string(d.Status)]++
		}
	}
	for _, st := range []string{"queued", "building", "starting", "health_checking", "ready", "failed", "canceled"} {
		b.WriteString(fmt.Sprintf("aether_deployments_total{status=%q} %d\n", st, statuses[st]))
	}

	up := 0
	b.WriteString("# HELP aether_app_usage Per-app resource usage.\n# TYPE aether_app_usage gauge\n")
	for _, a := range apps {
		cid := ""
		if deploys, err := c.Store.ListDeployments(a.ID, 1); err == nil && len(deploys) > 0 {
			cid = deploys[0].ContainerID
		}
		if cid == "" {
			continue
		}
		st, err := c.Driver.Stats(ctx, cid)
		if err != nil || st.MemBytes == 0 {
			continue
		}
		up++
		labels := fmt.Sprintf("app=%q,project=%q", a.Name, a.ProjectID)
		b.WriteString(fmt.Sprintf("aether_app_usage{metric=\"cpu_percent\",%s} %g\n", labels, st.CPUPercent))
		b.WriteString(fmt.Sprintf("aether_app_usage{metric=\"mem_bytes\",%s} %d\n", labels, st.MemBytes))
		b.WriteString(fmt.Sprintf("aether_app_usage{metric=\"net_rx_bytes\",%s} %d\n", labels, st.NetRxBytes))
		b.WriteString(fmt.Sprintf("aether_app_usage{metric=\"net_tx_bytes\",%s} %d\n", labels, st.NetTxBytes))
		b.WriteString(fmt.Sprintf("aether_app_usage{metric=\"io_read_bytes\",%s} %d\n", labels, st.IOReadBytes))
		b.WriteString(fmt.Sprintf("aether_app_usage{metric=\"io_write_bytes\",%s} %d\n", labels, st.IOWriteBytes))
	}
	b.WriteString("# HELP aether_apps_up Apps reporting metrics.\n# TYPE aether_apps_up gauge\n")
	b.WriteString(fmt.Sprintf("aether_apps_up %d\n", up))

	if d, err := c.Driver.Info(ctx); err == nil {
		b.WriteString("# HELP aether_driver_info Runtime driver.\n# TYPE aether_driver_info gauge\n")
		b.WriteString(fmt.Sprintf("aether_driver_info{driver=%q,version=%q} 1\n", d.Driver, d.Version))
	}
	b.WriteString("# HELP aether_uptime_seconds Platform uptime.\n# TYPE aether_uptime_seconds gauge\n")
	b.WriteString(fmt.Sprintf("aether_uptime_seconds %d\n", int64(time.Since(c.startedAt).Seconds())))
	return b.String()
}

func (c *Core) MetricsJSON(ctx context.Context) string {
	orgs, _ := c.Store.ListOrgs()
	total := 0
	for _, o := range orgs {
		list, _ := c.Store.ListApps(o.ID)
		total += len(list)
	}
	return "{\"apps_total\":" + strconv.Itoa(total) + ",\"generated_at\":\"" + time.Now().UTC().Format(time.RFC3339) + "\"}\n"
}
