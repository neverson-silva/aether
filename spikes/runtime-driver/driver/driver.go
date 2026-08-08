// Package driver define a porta RuntimeDriver (RFC-0006) e a implementação
// DockerDriver usada para validar a semântica da interface em F0.
//
// A interface é neutra OCI; o Docker aqui é um STAND-IN da semântica do driver
// (o driver real de produção é o PodmanDriver). Se a interface expressa bem o
// Docker, expressa bem Podman (mesmo modelo OCI/CLI).
package driver

import (
	"context"
	"io"
	"time"
)

// ---- Tipos (espelho da RFC-0006) ----

type ImageRef struct {
	Registry string
	Repo     string
	Tag      string
	Digest   string
}

func (i ImageRef) String() string {
	ref := i.Registry
	if ref != "" {
		ref += "/"
	}
	ref += i.Repo
	if i.Tag != "" {
		ref += ":" + i.Tag
	} else if i.Digest != "" {
		ref += "@" + i.Digest
	}
	return ref
}

type RegistryAuth struct {
	Username string
	Password string
}

type ContainerSpec struct {
	Name      string
	Image     ImageRef
	Command   []string
	Env       []string
	Ports     []PortBinding
	Networks  []string
	Volumes   []VolumeMount
	Resources ResourceSpec
	ReadOnly  bool
}

type PortBinding struct {
	ContainerPort string
	HostPort      string
}

type VolumeMount struct {
	Source string
	Target string
}

type ResourceSpec struct {
	CPUs    float64
	MemMB   int64
	Pids    int64
	Devices []string
}

type VolumeSpec struct {
	Name     string
	Driver   string
	SizeMB   int64
	ReadOnly bool
}

type NetworkSpec struct {
	Name     string
	Subnet   string
	Internal bool
	Driver   string
}

type ExecRequest struct {
	Command []string
	Stdin   io.Reader
	Env     []string
}

type ExecResult struct {
	Stdout   []byte
	ExitCode int
}

type LogRequest struct {
	Since  time.Time
	Follow bool
	Tail   int
}

type LogStream struct {
	Reader io.ReadCloser
	Close  func() error
}

type ContainerInfo struct {
	ID        string
	Name      string
	Image     string
	State     string
	Status    string
	Networks  []string
	Ports     []string
	StartedAt time.Time
}

type ContainerStats struct {
	CpuPercent   float64
	MemBytes     uint64
	MemLimit     uint64
	Pids         int64
	NetRxBytes   uint64
	NetTxBytes   uint64
	IOReadBytes  uint64
	IOWriteBytes uint64
}

type RuntimeInfo struct {
	Driver        string
	Version       string
	StorageDriver string
	Rootless      bool
	Capabilities  []string
}

type ImageInfo struct {
	Ref     string
	Size    int64
	Created time.Time
	Layers  int
	Digest  string
}

type ContainerHandle struct {
	ID   string
	Name string
}

type NetworkInfo struct {
	ID       string
	Name     string
	Driver   string
	Subnet   string
	Internal bool
}

type VolumeInfo struct {
	ID     string
	Name   string
	Driver string
	SizeMB int64
}

type PrunePolicy struct {
	UnusedOnly   bool
	OlderThan    time.Duration
	KeepReferred int
}

type PruneReport struct {
	ImagesRemoved  int
	BytesReclaimed int64
	VolumesRemoved int
}

type GCRequest struct {
	Images PrunePolicy
}

type GCReport struct {
	Prune PruneReport
}

// ---- Erros normalizados (RFC-0006) ----

type ErrorCode string

const (
	ErrImageNotFound     ErrorCode = "image_not_found"
	ErrContainerNotFound ErrorCode = "container_not_found"
	ErrNetworkNotFound   ErrorCode = "network_not_found"
	ErrVolumeNotFound    ErrorCode = "volume_not_found"
	ErrPermissionDenied  ErrorCode = "permission_denied"
	ErrInsufficientRes   ErrorCode = "insufficient_resources"
	ErrTimeout           ErrorCode = "timeout"
	ErrDriver            ErrorCode = "driver_error"
	ErrConflict          ErrorCode = "conflict"
)

type DriverError struct {
	Code ErrorCode
	Msg  string
}

func (e *DriverError) Error() string { return string(e.Code) + ": " + e.Msg }

// ---- A PORTA (RFC-0006) ----

type RuntimeDriver interface {
	// Imagens
	Pull(ctx context.Context, ref ImageRef, auth *RegistryAuth) (string, error)
	Push(ctx context.Context, ref ImageRef, auth *RegistryAuth) error
	InspectImage(ctx context.Context, ref ImageRef) (*ImageInfo, error)
	ListImages(ctx context.Context) ([]ImageInfo, error)
	PruneImages(ctx context.Context, policy PrunePolicy) (*PruneReport, error)

	// Containers
	Run(ctx context.Context, spec ContainerSpec) (*ContainerHandle, error)
	Start(ctx context.Context, id string) error
	Stop(ctx context.Context, id string, timeout time.Duration) error
	Restart(ctx context.Context, id string, timeout time.Duration) error
	Remove(ctx context.Context, id string, force bool) error
	Inspect(ctx context.Context, id string) (*ContainerInfo, error)
	List(ctx context.Context) ([]ContainerInfo, error)
	Exec(ctx context.Context, id string, req ExecRequest) (*ExecResult, error)
	Logs(ctx context.Context, id string, req LogRequest) (*LogStream, error)
	Stats(ctx context.Context, id string) (*ContainerStats, error)

	// Rede
	NetworkCreate(ctx context.Context, spec NetworkSpec) (string, error)
	NetworkRemove(ctx context.Context, id string) error
	NetworkInspect(ctx context.Context, id string) (*NetworkInfo, error)
	NetworkConnect(ctx context.Context, netID, containerID string) error
	NetworkDisconnect(ctx context.Context, netID, containerID string) error

	// Volumes
	VolumeCreate(ctx context.Context, spec VolumeSpec) (string, error)
	VolumeRemove(ctx context.Context, id string, force bool) error
	VolumeInspect(ctx context.Context, id string) (*VolumeInfo, error)

	// Sistema
	Info(ctx context.Context) (*RuntimeInfo, error)
	GC(ctx context.Context, req GCRequest) (*GCReport, error)
}
