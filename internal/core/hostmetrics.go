package core

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type HostNet struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
}

type HostDisk struct {
	ReadBytes  uint64  `json:"read_bytes"`
	WriteBytes uint64  `json:"write_bytes"`
	Total      uint64  `json:"total"`
	Used       uint64  `json:"used"`
	Percent    float64 `json:"percent"`
}

type HostStats struct {
	CPUPercent float64   `json:"cpu_percent"`
	CPUCores   int       `json:"cpu_cores"`
	MemTotal   uint64    `json:"mem_total"`
	MemUsed    uint64    `json:"mem_used"`
	MemPercent float64   `json:"mem_percent"`
	Net        HostNet   `json:"net"`
	Disk       HostDisk  `json:"disk"`
	Uptime     uint64    `json:"uptime"`
	Load       []float64 `json:"load"`
	Hostname   string    `json:"hostname"`
	OS         string    `json:"os"`
}

func (c *Core) HostStats(ctx context.Context) HostStats {
	var s HostStats
	if p, err := cpu.Percent(500*time.Millisecond, false); err == nil && len(p) > 0 {
		s.CPUPercent = p[0]
	}
	if n, err := cpu.Counts(false); err == nil {
		s.CPUCores = n
	}
	if m, err := mem.VirtualMemory(); err == nil {
		s.MemTotal = m.Total
		s.MemUsed = m.Used
		s.MemPercent = m.UsedPercent
	}
	if n, err := net.IOCounters(false); err == nil && len(n) > 0 {
		s.Net.RxBytes = n[0].BytesRecv
		s.Net.TxBytes = n[0].BytesSent
	}
	if usage, err := disk.Usage("/"); err == nil {
		s.Disk.Total = usage.Total
		s.Disk.Used = usage.Used
		s.Disk.Percent = usage.UsedPercent
	}
	if io, err := disk.IOCounters(); err == nil {
		for _, v := range io {
			s.Disk.ReadBytes += v.ReadBytes
			s.Disk.WriteBytes += v.WriteBytes
		}
	}
	if h, err := host.Info(); err == nil {
		s.Uptime = h.Uptime
		s.Hostname = h.Hostname
		s.OS = h.Platform + " " + h.PlatformVersion
	}
	if l, err := load.Avg(); err == nil {
		s.Load = []float64{l.Load1, l.Load5, l.Load15}
	}
	return s
}
