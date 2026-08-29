package domain

import "strings"

type Kind string

const (
	KindApp      Kind = "app"
	KindCompose  Kind = "compose"
	KindDatabase Kind = "database"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusDeploying Status = "deploying"
	StatusRunning   Status = "running"
	StatusDegraded  Status = "degraded"
	StatusStopped   Status = "stopped"
	StatusFailed    Status = "failed"
	StatusUnknown   Status = "unknown"
)

type Capabilities struct {
	CanDeploy            bool `json:"can_deploy"`
	CanRestart           bool `json:"can_restart"`
	CanStop              bool `json:"can_stop"`
	CanStart             bool `json:"can_start"`
	CanOpenTerminal      bool `json:"can_open_terminal"`
	CanViewLogs          bool `json:"can_view_logs"`
	CanViewMetrics       bool `json:"can_view_metrics"`
	CanManageDomains     bool `json:"can_manage_domains"`
	CanManageEnvironment bool `json:"can_manage_environment"`
	CanBuild             bool `json:"can_build"`
	CanEditCompose       bool `json:"can_edit_compose"`
	CanManageBackups     bool `json:"can_manage_backups"`
	CanRestore           bool `json:"can_restore"`
	CanManageSchedules   bool `json:"can_manage_schedules"`
	CanManageSource      bool `json:"can_manage_source"`
}

type ContainerState struct {
	ID      string
	Name    string
	Status  string
	Healthy *bool
}

func CapabilitiesFor(kind Kind) Capabilities {
	capabilities := Capabilities{
		CanRestart:           true,
		CanStop:              true,
		CanStart:             true,
		CanOpenTerminal:      true,
		CanViewLogs:          true,
		CanViewMetrics:       true,
		CanManageDomains:     true,
		CanManageEnvironment: true,
	}
	switch kind {
	case KindApp:
		capabilities.CanDeploy = true
		capabilities.CanBuild = true
		capabilities.CanManageSchedules = true
		capabilities.CanManageSource = true
	case KindCompose:
		capabilities.CanDeploy = true
		capabilities.CanEditCompose = true
		capabilities.CanManageSource = true
	case KindDatabase:
		capabilities.CanDeploy = true
		capabilities.CanManageBackups = true
		capabilities.CanRestore = true
	}
	return capabilities
}

func NormalizeApp(deployment, container string, healthy *bool) Status {
	if isTransition(deployment) {
		return StatusDeploying
	}
	if container == "" {
		return statusFromDeployment(deployment)
	}
	return normalizeContainer(container, healthy)
}

func NormalizeDatabase(container string, healthy *bool) Status {
	if container == "" {
		return StatusUnknown
	}
	if healthy != nil && !*healthy {
		return StatusDegraded
	}
	return normalizeContainer(container, healthy)
}

func NormalizeCompose(states []ContainerState, deploying bool) Status {
	if deploying {
		return StatusDeploying
	}
	if len(states) == 0 {
		return StatusUnknown
	}
	running := 0
	failed := 0
	stopped := 0
	for _, state := range states {
		normalized := normalizeContainer(state.Status, state.Healthy)
		switch normalized {
		case StatusRunning:
			running++
		case StatusStopped:
			if isIntentionalStop(state.Status) {
				stopped++
			} else {
				failed++
			}
		case StatusFailed, StatusDegraded:
			failed++
		default:
			failed++
		}
	}
	if running == len(states) {
		return StatusRunning
	}
	if running > 0 && running < len(states) {
		return StatusDegraded
	}
	if failed > 0 {
		return StatusFailed
	}
	if stopped == len(states) {
		return StatusStopped
	}
	return StatusUnknown
}

func isIntentionalStop(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "created", "stopped", "paused":
		return true
	default:
		return false
	}
}

func normalizeContainer(raw string, healthy *bool) Status {
	if healthy != nil && !*healthy {
		return StatusDegraded
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "running", "healthy":
		return StatusRunning
	case "created", "configured", "exited", "stopped", "paused":
		return StatusStopped
	case "restarting", "starting":
		return StatusDeploying
	case "unhealthy", "failed", "dead":
		return StatusFailed
	default:
		return StatusUnknown
	}
}

func statusFromDeployment(raw string) Status {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "queued", "pending", "building", "deploying", "starting":
		return StatusDeploying
	case "success", "succeeded", "ready", "running":
		return StatusUnknown
	case "failed", "error":
		return StatusFailed
	case "stopped", "cancelled", "canceled":
		return StatusStopped
	default:
		return StatusUnknown
	}
}

func isTransition(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "queued", "pending", "building", "deploying", "starting", "restarting":
		return true
	default:
		return false
	}
}
