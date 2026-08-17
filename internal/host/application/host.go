package application

import (
	"bufio"
	"context"
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
}

func (h *Host) Stats(ctx context.Context) domain.Stats {
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
