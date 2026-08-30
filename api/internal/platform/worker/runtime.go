package worker

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
)

var (
	ErrRuntimeUnavailable      = errors.New("container runtime unavailable")
	ErrRuntimePermission       = errors.New("container runtime permission denied")
	ErrRuntimeTimeout          = errors.New("container runtime timeout")
	ErrContainerNotFound       = errors.New("container not found")
	ErrContainerStopped        = errors.New("container stopped")
	ErrStorageLimitUnsupported = errors.New("container storage limit unsupported")
)

type RunSpec struct {
	Name          string
	Image         string
	Env           []string
	Port          int
	ContainerPort int
	Network       string
	NetworkAlias  string
	MemMB         int
	CPUs          string
	StorageMB     int
	Labels        map[string]string
	Command       []string
	Mounts        []MountSpec
	Ports         []PortSpec
}

type MountSpec struct {
	Source   string
	Target   string
	ReadOnly bool
}

type PortSpec struct {
	HostPort      int
	ContainerPort int
}

type LifecycleRuntime interface {
	Run(ctx context.Context, spec RunSpec) (containerID string, err error)
	Port(ctx context.Context, containerID string) (hostPort string, err error)
	HealthCheck(ctx context.Context, hostPort, path string) error
	Remove(ctx context.Context, containerID string) error
	RemoveByLabel(ctx context.Context, label string) error
	Start(ctx context.Context, containerID string) error
	Stop(ctx context.Context, containerID string) error
	Restart(ctx context.Context, containerID string) error
}

type ImageRuntime interface {
	Pull(ctx context.Context, image string) (output string, err error)
	ExposedPort(ctx context.Context, image string) (int, error)
}

type ImageBuildRuntime interface {
	Build(ctx context.Context, dir, dockerfile, tag string) (output string, err error)
}

type ImageRegistryRuntime interface {
	Pull(ctx context.Context, image string) (output string, err error)
	Push(ctx context.Context, image string) (output string, err error)
	Tag(ctx context.Context, source, target string) error
	ListImages(ctx context.Context) ([]ImageInfo, error)
	RemoveImage(ctx context.Context, image string) error
}

type ImageInfo struct {
	Names   []string `json:"names"`
	ID      string   `json:"id"`
	Size    int64    `json:"size"`
	Created int64    `json:"created"`
}

type LogsRuntime interface {
	FollowLogs(ctx context.Context, containerID string, writer io.Writer) error
	LogTail(ctx context.Context, containerID string, lines int) ([]string, error)
}

type StatsRuntime interface {
	ContainerState(ctx context.Context, containerID string) (string, error)
	Stats(ctx context.Context, containerID string) (ContainerStats, error)
	ListContainers(ctx context.Context) ([]ContainerInfo, error)
}

type ContainerMetadataRuntime interface {
	ListContainerMetadata(ctx context.Context) ([]ContainerInfo, error)
}

type RuntimeEvent struct {
	ID          string
	Action      string
	ContainerID string
	Name        string
	Status      string
	Health      string
	Labels      map[string]string
	OccurredAt  time.Time
}

type RuntimeEventSubscription interface {
	Events() <-chan RuntimeEvent
	Errors() <-chan error
	Close() error
}

type EventRuntime interface {
	SubscribeEvents(ctx context.Context, filters map[string]string) (RuntimeEventSubscription, error)
}

type RuntimeServiceTarget struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	Kind             string
	Status           string
	EverDeployed     bool
	ActiveDeployment bool
}

type ServiceStateStore interface {
	ListRuntimeServiceTargets(ctx context.Context) ([]RuntimeServiceTarget, error)
	UpdateRuntimeStatus(ctx context.Context, serviceID uuid.UUID, status string) (bool, error)
}

type ExecRuntime interface {
	Exec(ctx context.Context, containerID string, env []string, args ...string) (stdout string, stderr string, err error)
}

type CommandRuntime interface {
	RunCommand(ctx context.Context, name, image, command string, env []string, remove bool) (string, error)
}

type WaitRuntime interface {
	Wait(ctx context.Context, containerID string) (int64, error)
}

type StorageRuntime interface {
	StorageUsage(ctx context.Context) (map[string]uint64, error)
}

type InteractiveSession interface {
	io.ReadWriteCloser
	Resize(ctx context.Context, cols, rows uint16) error
}

type InteractiveRuntime interface {
	OpenInteractive(ctx context.Context, containerID string, args ...string) (InteractiveSession, error)
}

type Runtime interface {
	LifecycleRuntime
	ImageRuntime
	ImageBuildRuntime
	LogsRuntime
	StatsRuntime
	ExecRuntime
}

type NetworkRuntime interface {
	EnsureNetwork(ctx context.Context, name string, labels map[string]string) error
	RemoveNetwork(ctx context.Context, name string) error
}

type VolumeRuntime interface {
	CreateVolume(ctx context.Context, name string, labels map[string]string) error
	RemoveVolume(ctx context.Context, name string, force bool) error
}

type ContainerStats struct {
	CPUPercent  float64 `json:"cpu_percent"`
	MemUsage    uint64  `json:"mem_usage"`
	MemBytes    uint64  `json:"mem_bytes"`
	MemLimit    uint64  `json:"mem_limit"`
	MemPercent  float64 `json:"mem_percent"`
	NetInput    uint64  `json:"net_input"`
	NetOutput   uint64  `json:"net_output"`
	BlockInput  uint64  `json:"block_input"`
	BlockOutput uint64  `json:"block_output"`
}

type ContainerInfo struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	State    string            `json:"state"`
	Labels   map[string]string `json:"labels"`
	Healthy  *bool             `json:"healthy,omitempty"`
	Stats    ContainerStats    `json:"stats"`
	HasStats bool              `json:"has_stats"`
}
