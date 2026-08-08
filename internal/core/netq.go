package core

import (
	"context"
	"net"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"aether/internal/domain"
)

type netSample struct {
	At time.Time `json:"at"`
	Ms float64   `json:"ms"`
	OK bool      `json:"ok"`
	H3 bool      `json:"h3"`
}

type netAppStat struct {
	AppID   string      `json:"app_id"`
	Name    string      `json:"name"`
	Addr    string      `json:"addr"`
	Samples []netSample `json:"samples"`
	P50     float64     `json:"p50_ms"`
	P95     float64     `json:"p95_ms"`
	Uptime  float64     `json:"uptime_pct"`
	H3      bool        `json:"http3"`
}

type netQState struct {
	mu    sync.Mutex
	stats map[string]*netAppStat
}

func (c *Core) StartNetQ(ctx context.Context) {
	c.netQ = &netQState{stats: map[string]*netAppStat{}}
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		c.netQProbe()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.netQProbe()
			}
		}
	}()
}

func (c *Core) netQProbe() {
	orgs, _ := c.Store.ListOrgs()
	for _, o := range orgs {
		projects, _ := c.Store.ListProjects(o.ID)
		for _, p := range projects {
			apps, _ := c.Store.ListAppsByProject(p.ID)
			for _, a := range apps {
				c.probeApp(&a)
			}
		}
	}
}

func (c *Core) probeApp(a *domain.App) {
	deploys, err := c.Store.ListDeployments(a.ID, 1)
	if err != nil || len(deploys) == 0 {
		return
	}
	dep := deploys[0]
	if dep.ContainerID == "" || dep.Status != "ready" {
		return
	}
	ports, err := c.Driver.Ports(context.Background(), dep.ContainerID)
	if err != nil {
		return
	}
	hostPort := ""
	for _, hp := range ports {
		hostPort = hp
		break
	}
	if hostPort == "" {
		return
	}
	host, _, _ := net.SplitHostPort(hostPort)
	if host == "0.0.0.0" || host == "" {
		host = "127.0.0.1"
	}
	addr := net.JoinHostPort(host, strconv.Itoa(portOf(hostPort)))
	start := time.Now()
	client := &http.Client{Timeout: 4 * time.Second}
	req, _ := http.NewRequest("HEAD", "http://"+addr, nil)
	resp, err := client.Do(req)
	ms := float64(time.Since(start).Microseconds()) / 1000
	s := netSample{At: time.Now(), Ms: ms, OK: err == nil}
	if err == nil {
		if resp.StatusCode >= 200 && resp.StatusCode < 500 {
			s.OK = true
		}
		s.H3 = resp.Header.Get("Alt-Svc") != ""
		resp.Body.Close()
	}
	c.netQ.mu.Lock()
	defer c.netQ.mu.Unlock()
	st, ok := c.netQ.stats[a.ID]
	if !ok {
		st = &netAppStat{AppID: a.ID, Name: a.Name, Addr: addr}
		c.netQ.stats[a.ID] = st
	}
	st.Samples = append(st.Samples, s)
	if len(st.Samples) > 120 {
		st.Samples = st.Samples[len(st.Samples)-120:]
	}
	var lats []float64
	okCount := 0
	for _, x := range st.Samples {
		if x.OK {
			lats = append(lats, x.Ms)
			okCount++
		}
		if x.H3 {
			st.H3 = true
		}
	}
	sort.Float64s(lats)
	st.P50 = percentile(lats, 50)
	st.P95 = percentile(lats, 95)
	if len(st.Samples) > 0 {
		st.Uptime = float64(okCount) / float64(len(st.Samples)) * 100
	}
}

func portOf(hostPort string) int {
	_, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		if p, perr := strconv.Atoi(hostPort); perr == nil {
			return p
		}
		return 80
	}
	p, _ := strconv.Atoi(port)
	if p == 0 {
		return 80
	}
	return p
}

func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := (len(sorted) * p) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func (c *Core) NetQStats() []netAppStat {
	c.netQ.mu.Lock()
	defer c.netQ.mu.Unlock()
	out := make([]netAppStat, 0, len(c.netQ.stats))
	for _, st := range c.netQ.stats {
		out = append(out, *st)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
