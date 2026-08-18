package application

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"

	"aether/internal/host/domain"
)

type Host struct {
	LogsDir string
	// AgentFile is the path of the host-stats.json written by the macOS host
	// agent (scripts/host-agent.sh). When present and fresh, it is the source
	// of truth for HOST metrics; otherwise local (runtime/VM) metrics are used
	// and explicitly labeled with Source.
	AgentFile string
}

type agentStats struct {
	TS           int64     `json:"ts"`
	Source       string    `json:"source"`
	CPUPercent   float64   `json:"cpu_percent"`
	CPUCores     int       `json:"cpu_cores"`
	MemTotal     uint64    `json:"mem_total"`
	MemUsed      uint64    `json:"mem_used"`
	MemPercent   float64   `json:"mem_percent"`
	DiskTotal    uint64    `json:"disk_total"`
	DiskUsed     uint64    `json:"disk_used"`
	DiskPercent  float64   `json:"disk_percent"`
	NetRxBytes   uint64    `json:"net_rx_bytes"`
	NetTxBytes   uint64    `json:"net_tx_bytes"`
	Load         []float64 `json:"load"`
	Uptime       uint64    `json:"uptime"`
	Hostname     string    `json:"hostname"`
	OS           string    `json:"os"`
}

const agentFreshWindow = 15 * time.Second

func (h *Host) Stats(ctx context.Context) domain.Stats {
	runtimeCores, _ := cpu.Counts(true)
	stats := h.localStats()
	stats.RuntimeCores = runtimeCores
	if agent, ok := h.agentStats(); ok {
		stats.CPUPercent = agent.CPUPercent
		stats.CPUCores = agent.CPUCores
		stats.MemTotal, stats.MemUsed, stats.MemPercent = agent.MemTotal, agent.MemUsed, agent.MemPercent
		stats.Disk.Total, stats.Disk.Used, stats.Disk.Percent = agent.DiskTotal, agent.DiskUsed, agent.DiskPercent
		stats.Net.RxBytes, stats.Net.TxBytes = agent.NetRxBytes, agent.NetTxBytes
		stats.Uptime = agent.Uptime
		stats.Hostname = agent.Hostname
		stats.OS = agent.OS
		stats.Source = agent.Source
		if stats.Source == "" {
			stats.Source = "host-agent"
		}
	}
	return stats
}

func (h *Host) agentStats() (agentStats, bool) {
	if h.AgentFile == "" {
		return agentStats{}, false
	}
	info, err := os.Stat(h.AgentFile)
	if err != nil || time.Since(info.ModTime()) > agentFreshWindow {
		return agentStats{}, false
	}
	raw, err := os.ReadFile(h.AgentFile)
	if err != nil {
		return agentStats{}, false
	}
	var a agentStats
	if err := json.Unmarshal(raw, &a); err != nil || a.MemTotal == 0 {
		return agentStats{}, false
	}
	return a, true
}

func (h *Host) localStats() domain.Stats {
	cpuPercent, _ := cpu.Percent(0, false)
	memStat, _ := mem.VirtualMemory()
	diskStat, _ := disk.Usage("/")
	netCounters, _ := net.IOCounters(false)
	loadAvg, _ := load.Avg()
	hostInfo, _ := host.Info()
	uptime, _ := host.Uptime()

	var rx, tx uint64
	if len(netCounters) > 0 {
		rx, tx = netCounters[0].BytesRecv, netCounters[0].BytesSent
	}
	cores, _ := cpu.Counts(true)
	stats := domain.Stats{
		CPUCores: cores,
		Net:      domain.Net{RxBytes: rx, TxBytes: tx},
		Uptime:   uptime,
		Hostname: hostInfo.Hostname,
		OS:       hostInfo.Platform + " " + hostInfo.PlatformVersion,
		Source:   "runtime",
	}
	if len(cpuPercent) > 0 {
		stats.CPUPercent = cpuPercent[0]
	}
	if memStat != nil {
		stats.MemTotal, stats.MemUsed, stats.MemPercent = memStat.Total, memStat.Used, memStat.UsedPercent
	}
	if diskStat != nil {
		stats.Disk = domain.Disk{
			Total: diskStat.Total, Used: diskStat.Used, Percent: diskStat.UsedPercent,
		}
	}
	if loadAvg != nil {
		stats.Load = []float64{loadAvg.Load1, loadAvg.Load5, loadAvg.Load15}
	}
	return stats
}

func (h *Host) Events(limit int) ([]domain.Event, error) {
	if h.LogsDir == "" {
		return []domain.Event{}, nil
	}
	file, err := os.Open(h.LogsDir + "/aether.log")
	if err != nil {
		return []domain.Event{}, nil
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > 2000 {
			lines = lines[1:]
		}
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	start := len(lines) - limit
	if start < 0 {
		start = 0
	}
	events := make([]domain.Event, 0, limit)
	for _, line := range lines[start:] {
		ts, typ, detail := parseLogLine(line)
		events = append(events, domain.Event{TS: ts, Type: typ, Detail: detail})
	}
	return events, nil
}

func (h *Host) Logs(limit int) ([]string, error) {
	if h.LogsDir == "" {
		return []string{}, nil
	}
	file, err := os.Open(h.LogsDir + "/aether.log")
	if err != nil {
		return []string{}, nil
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	return lines, nil
}

func parseLogLine(line string) (ts, typ, detail string) {
	parts := strings.SplitN(line, " ", 4)
	if len(parts) >= 4 {
		ts = parts[0] + " " + parts[1]
		typ = parts[2]
		detail = parts[3]
		return
	}
	if _, err := time.Parse(time.RFC3339, line[:min(len(line), 25)]); err == nil {
		ts = line[:25]
		detail = line[26:]
		return
	}
	detail = line
	return
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
