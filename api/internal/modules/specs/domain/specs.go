package domain

import (
	"errors"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("invalid input")
)

type HealthCheck struct {
	Enabled    bool   `json:"enabled"`
	Path       string `json:"path"`
	IntervalMS int    `json:"interval_ms"`
	TimeoutMS  int    `json:"timeout_ms"`
	Retries    int    `json:"retries"`
}

type Spec struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Port        int               `json:"port"`
	MemMB       int               `json:"mem_mb"`
	CPUs        string            `json:"cpus"`
	Env         map[string]string `json:"env"`
	HealthCheck HealthCheck       `json:"health_check"`
	Replicas    int               `json:"replicas"`
}

type DeploymentDiff struct {
	ImageA     string               `json:"image_a"`
	ImageB     string               `json:"image_b"`
	StatusA    string               `json:"status_a"`
	StatusB    string               `json:"status_b"`
	EnvAdded   []string             `json:"env_added"`
	EnvRemoved []string             `json:"env_removed"`
	EnvChanged map[string][2]string `json:"env_changed"`
	NumberA    int                  `json:"number_a"`
	NumberB    int                  `json:"number_b"`
}

type DetectResult struct {
	Framework    string `json:"framework"`
	BuildMethod  string `json:"build_method"`
	BuildCommand string `json:"build_command"`
	StartCommand string `json:"start_command"`
	OutputDir    string `json:"output_dir"`
	Port         int    `json:"port"`
	Detected     bool   `json:"detected"`
}

type Plan struct {
	Framework      string   `json:"framework"`
	Library        string   `json:"library"`
	PackageManager string   `json:"package_manager"`
	BuildCommand   string   `json:"build_command"`
	InstallCommand string   `json:"install_command"`
	OutputDir      string   `json:"output_dir"`
	AppType        string   `json:"app_type"`
	WebServer      string   `json:"web_server"`
	ContainerPort  int      `json:"container_port"`
	SPAFallback    bool     `json:"spa_fallback"`
	IndexFile      string   `json:"index_file"`
	HasLockfile    bool     `json:"has_lockfile"`
	Detected       bool     `json:"detected"`
	NginxConf      string   `json:"nginx_conf"`
	Dockerfile     string   `json:"dockerfile"`
	Warnings       []string `json:"warnings"`
}

type PlanPreview struct {
	Dockerfile string `json:"dockerfile"`
	NginxConf  string `json:"nginx_conf"`
}

type AppSummary struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	CPU    float64 `json:"cpu_pct"`
	MemPct float64 `json:"mem_pct"`
	NetRx  uint64  `json:"net_rx_bytes"`
	NetTx  uint64  `json:"net_tx_bytes"`
	IORx   uint64  `json:"io_read_bytes"`
	IOWx   uint64  `json:"io_write_bytes"`
}

type ProjectRow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Apps       int    `json:"apps"`
	Env        string `json:"env"`
	Status     string `json:"status"`
	LastDeploy string `json:"last_deploy"`
}

type SystemSummary struct {
	HealthPct    float64      `json:"health_pct"`
	Deployments  int          `json:"deployments"`
	TrafficBytes uint64       `json:"traffic_bytes"`
	IOBytes      uint64       `json:"io_bytes"`
	CPUPct       float64      `json:"cpu_pct"`
	MemPct       float64      `json:"mem_pct"`
	IOPct        float64      `json:"io_pct"`
	Apps         []AppSummary `json:"apps"`
	Projects     []ProjectRow `json:"projects"`
}
