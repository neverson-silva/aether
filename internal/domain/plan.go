package domain

import "time"

type DeploymentPlan struct {
	ID             string    `json:"id"`
	AppID          string    `json:"app_id"`
	Framework      string    `json:"framework"`
	Library        string    `json:"library"`
	PackageManager string    `json:"package_manager"`
	Runtime        string    `json:"runtime"`
	BuildCommand   string    `json:"build_command"`
	InstallCommand string    `json:"install_command"`
	OutputDir      string    `json:"output_dir"`
	AppType        string    `json:"app_type"`
	WebServer      string    `json:"web_server"`
	ContainerPort  int       `json:"container_port"`
	SPAFallback    bool      `json:"spa_fallback"`
	IndexFile      string    `json:"index_file"`
	NginxConf      string    `json:"nginx_conf"`
	Dockerfile     string    `json:"dockerfile"`
	Warnings       []string  `json:"warnings"`
	DetectedAt     time.Time `json:"detected_at"`
	CreatedAt      time.Time `json:"created_at"`
}
