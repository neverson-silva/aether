package core

import (
	"context"
	"sort"
	"strconv"
	"time"
)

type appSummary struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	CPU    float64 `json:"cpu_pct"`
	MemPct float64 `json:"mem_pct"`
	NetRx  uint64  `json:"net_rx_bytes"`
	NetTx  uint64  `json:"net_tx_bytes"`
	IORx   uint64  `json:"io_read_bytes"`
	IOWx   uint64  `json:"io_write_bytes"`
}

func (c *Core) SystemSummary(ctx context.Context, orgID string) map[string]any {
	projects, _ := c.Store.ListProjects(orgID)
	type projRow struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Apps       int    `json:"apps"`
		Env        string `json:"env"`
		Status     string `json:"status"`
		LastDeploy string `json:"last_deploy"`
	}
	projRows := []projRow{}
	apps := []appSummary{}
	var totalNet, totalIO uint64
	var cpuSum, memSum, memMax float64
	uptimeSum := 0.0
	uptimeCount := 0

	totalDeployments := 0
	for _, p := range projects {
		list, _ := c.Store.ListAppsByProject(p.ID)
		dbList, _ := c.Store.ListDatabasesByProject(p.ID)
		row := projRow{ID: p.ID, Name: p.Name, Apps: len(list) + len(dbList), Env: "—", Status: "idle"}
		var lastDeploy time.Time
		anyReady, anyBuilding, anyFailed := false, false, false
		for _, a := range list {
			if n, cerr := c.Store.CountDeployments(a.ID); cerr == nil {
				totalDeployments += int(n)
			}
			deploys, _ := c.Store.ListDeployments(a.ID, 1)
			if len(deploys) == 0 {
				continue
			}
			d := deploys[0]
			switch string(d.Status) {
			case "ready":
				anyReady = true
			case "building", "starting", "health_checking", "queued":
				anyBuilding = true
			case "failed":
				anyFailed = true
			}
			if !d.StartedAt.IsZero() && d.StartedAt.After(lastDeploy) {
				lastDeploy = d.StartedAt
			}
			cid := d.ContainerID
			if cid == "" {
				continue
			}
			if st, err := c.Driver.Stats(ctx, cid); err == nil && st.MemBytes > 0 {
				cpuSum += st.CPUPercent
				memSum += float64(st.MemBytes)
				memMax += float64(st.MemLimit)
				totalNet += st.NetRxBytes + st.NetTxBytes
				totalIO += st.IOReadBytes + st.IOWriteBytes
				apps = append(apps, appSummary{
					ID: a.ID, Name: a.Name, CPU: st.CPUPercent,
					MemPct: pctOf(st.MemBytes, st.MemLimit),
					NetRx:  st.NetRxBytes, NetTx: st.NetTxBytes,
					IORx: st.IOReadBytes, IOWx: st.IOWriteBytes,
				})
			}
		}
		switch {
		case anyFailed:
			row.Status, row.Env = "degraded", "Production"
		case anyBuilding:
			row.Status, row.Env = "syncing", "Staging"
		case anyReady:
			row.Status, row.Env = "healthy", "Production"
		}
		if !lastDeploy.IsZero() {
			row.LastDeploy = humanAgo(lastDeploy)
		} else {
			row.LastDeploy = "—"
		}
		projRows = append(projRows, row)
	}
	netq := c.NetQStats()
	for _, n := range netq {
		uptimeSum += n.Uptime
		uptimeCount++
	}
	health := 0.0
	if uptimeCount > 0 {
		health = uptimeSum / float64(uptimeCount)
	} else if len(apps) > 0 {
		health = 100
	}
	appCount := len(apps)
	cpu := 0.0
	memPct := 0.0
	ioPct := 0.0
	if appCount > 0 {
		cpu = cpuSum / float64(appCount)
	}
	if memMax > 0 {
		memPct = memSum / memMax * 100
	}
	if totalIO > 0 {
		ioPct = float64(totalIO) / float64(timeSinceStart(c, ctx)) / (1 << 20) / 10 * 100
		if ioPct > 100 {
			ioPct = 100
		}
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].Name < apps[j].Name })
	return map[string]any{
		"health_pct":    round2(health),
		"deployments":   totalDeployments,
		"traffic_bytes": totalNet,
		"io_bytes":      totalIO,
		"cpu_pct":       round2(cpu),
		"mem_pct":       round2(memPct),
		"io_pct":        round2(ioPct),
		"apps":          apps,
		"projects":      projRows,
	}
}

func pctOf(used, limit uint64) float64 {
	if limit == 0 {
		return 0
	}
	return float64(used) / float64(limit) * 100
}

func timeSinceStart(c *Core, ctx context.Context) float64 {
	if s, err := c.Driver.Stats(ctx, ""); err == nil {
		_ = s
	}
	seconds := time.Since(c.startedAt).Seconds()
	if seconds < 1 {
		return 1
	}
	return seconds
}

func humanAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmtAgo(int(d.Minutes())) + "m ago"
	case d < 24*time.Hour:
		return fmtAgo(int(d.Hours())) + "h ago"
	default:
		return fmtAgo(int(d.Hours()/24)) + "d ago"
	}
}

func fmtAgo(n int) string {
	if n == 0 {
		return "0"
	}
	return strconv.Itoa(n)
}

type AllCronJob struct {
	ID       string `json:"id"`
	AppID    string `json:"app_id"`
	AppName  string `json:"app_name"`
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
	Command  string `json:"command"`
	Enabled  bool   `json:"enabled"`
	LastRun  string `json:"last_run"`
	NextRun  string `json:"next_run"`
	Status   string `json:"status"`
}

func (c *Core) AllCronJobs(orgID string) ([]AllCronJob, error) {
	var out []AllCronJob
	projects, _ := c.Store.ListProjects(orgID)
	for _, p := range projects {
		apps, _ := c.Store.ListAppsByProject(p.ID)
		for _, a := range apps {
			jobs, _ := c.Store.ListCronJobs(a.ID)
			for _, j := range jobs {
				next := c.nextCron(j.Schedule, time.Now())
				last := ""
				if !j.LastRun.IsZero() {
					last = humanAgo(j.LastRun)
				}
				status := "idle"
				out = append(out, AllCronJob{
					ID: j.ID, AppID: a.ID, AppName: a.Name, Name: j.Name,
					Schedule: j.Schedule, Command: j.Command, Enabled: j.Enabled,
					LastRun: last, NextRun: next.Format("15:04 02/01"), Status: status,
				})
			}
		}
	}
	return out, nil
}

type CertInfo struct {
	ID        string `json:"id"`
	AppID     string `json:"app_id"`
	AppName   string `json:"app_name"`
	Host      string `json:"host"`
	HTTPS     bool   `json:"https"`
	CertState string `json:"cert_status"`
	CreatedAt string `json:"created_at"`
}

func (c *Core) Certificates(orgID string) ([]CertInfo, error) {
	var out []CertInfo
	projects, _ := c.Store.ListProjects(orgID)
	for _, p := range projects {
		apps, _ := c.Store.ListAppsByProject(p.ID)
		for _, a := range apps {
			domains, _ := c.Store.ListDomains(a.ID)
			for _, d := range domains {
				out = append(out, CertInfo{
					ID: d.ID, AppID: a.ID, AppName: a.Name, Host: d.Host,
					HTTPS: d.HTTPS, CertState: d.CertStatus,
					CreatedAt: d.CreatedAt.UTC().Format(time.RFC3339),
				})
			}
		}
	}
	return out, nil
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
