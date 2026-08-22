package domain

import "time"

const (
	OwnerAether  = "aether"
	OwnerUser    = "user"
	OwnerSystem  = "system"
	OwnerUnknown = "unknown"
)

// Resource is a single container exposed to the UI with ownership metadata.
// No Docker internals (env, ports, paths) are exposed.
type Resource struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Owner        string  `json:"owner"`
	ServiceType  string  `json:"service_type"`
	ServiceID    string  `json:"service_id"`
	ProjectID    string  `json:"project_id"`
	State        string  `json:"state"`
	Active       bool    `json:"active"`
	HasStats     bool    `json:"has_stats"`
	CPUPercent   float64 `json:"cpu_percent"` // % of a single core (raw, can exceed 100)
	CPUOfHost    float64 `json:"cpu_of_host"` // % of total host capacity
	MemUsage     uint64  `json:"mem_usage"`
	MemLimit     uint64  `json:"mem_limit"`
	MemPercent   float64 `json:"mem_percent"`
	NetInput     uint64  `json:"net_input"`   // cumulative bytes since container start
	NetOutput    uint64  `json:"net_output"`  // cumulative bytes since container start
	NetRxRate    float64 `json:"net_rx_rate"` // bytes/sec
	NetTxRate    float64 `json:"net_tx_rate"` // bytes/sec
	HasNetRate   bool    `json:"has_net_rate"`
	BlockInput   uint64  `json:"block_input"`
	BlockOutput  uint64  `json:"block_output"`
	BlockRxRate  float64 `json:"block_rx_rate"` // bytes/sec (disk I/O)
	BlockTxRate  float64 `json:"block_tx_rate"` // bytes/sec (disk I/O)
	HasBlockRate bool    `json:"has_block_rate"`
	Storage      *uint64 `json:"storage"` // nil = unavailable
}

type Aggregate struct {
	CPUOfHost    float64 `json:"cpu_of_host"` // % of total host capacity
	MemUsage     uint64  `json:"mem_usage"`
	MemPercent   float64 `json:"mem_percent"`
	NetRxRate    float64 `json:"net_rx_rate"`
	NetTxRate    float64 `json:"net_tx_rate"`
	StorageUsage uint64  `json:"storage_usage"`
	Count        int     `json:"count"`
	RunningCount int     `json:"running_count"`
	Available    bool    `json:"available"`
}

type Host struct {
	CPUPercent   float64   `json:"cpu_percent"`
	CPUCores     int       `json:"cpu_cores"`
	RuntimeCores int       `json:"runtime_cores"`
	MemTotal     uint64    `json:"mem_total"`
	MemUsed      uint64    `json:"mem_used"`
	MemPercent   float64   `json:"mem_percent"`
	DiskTotal    uint64    `json:"disk_total"`
	DiskUsed     uint64    `json:"disk_used"`
	DiskPercent  float64   `json:"disk_percent"`
	NetRxRate    float64   `json:"net_rx_rate"`
	NetTxRate    float64   `json:"net_tx_rate"`
	Load         []float64 `json:"load"`
	Uptime       uint64    `json:"uptime"`
	Hostname     string    `json:"hostname"`
	OS           string    `json:"os"`
	Source       string    `json:"source"`
}

type CollectorStats struct {
	CollectCount  int64   `json:"collect_count"`
	ErrorCount    int64   `json:"error_count"`
	LastCollectMS float64 `json:"last_collect_ms"`
	Resources     int     `json:"resources"`
	WithStats     int     `json:"with_stats"`
	LastError     string  `json:"last_error,omitempty"`
	UpSince       string  `json:"up_since"`
}

type Snapshot struct {
	TS        time.Time      `json:"ts"`
	Host      Host           `json:"host"`
	Aether    Aggregate      `json:"aether"`
	User      Aggregate      `json:"user"`
	System    Aggregate      `json:"system"`
	Resources []Resource     `json:"resources"`
	Collector CollectorStats `json:"collector"`
}

type HistoryPoint struct {
	TS           int64   `json:"ts"`
	HostCPU      float64 `json:"host_cpu"`
	HostMem      float64 `json:"host_mem"`
	AetherCPU    float64 `json:"aether_cpu"`
	AetherMem    uint64  `json:"aether_mem"`
	AetherMemPct float64 `json:"aether_mem_pct"`
	UserCPU      float64 `json:"user_cpu"`
	UserMem      uint64  `json:"user_mem"`
	UserMemPct   float64 `json:"user_mem_pct"`
	NetRx        float64 `json:"net_rx"`
	NetTx        float64 `json:"net_tx"`
}

type ResourcePoint struct {
	TS    int64   `json:"ts"`
	CPU   float64 `json:"cpu"` // % of host
	Mem   uint64  `json:"mem"`
	NetRx float64 `json:"net_rx"`
	NetTx float64 `json:"net_tx"`
}
