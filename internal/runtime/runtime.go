package runtime

import (
	"context"
	"io"
	"time"
)

type PortBinding struct {
	HostPort      string
	ContainerPort string
}

type VolumeMount struct {
	Source string
	Target string
}

type RunSpec struct {
	Name         string
	Image        string
	Cmd          []string
	Env          []string
	Ports        []PortBinding
	Networks     []string
	NetworkAlias string
	Volumes      []VolumeMount
	MemMB        int64
	CPUs         string
	PidsLimit    int64
	Restart      string
	ReadOnly     bool
	Labels       map[string]string
}

type ContainerInfo struct {
	ID        string
	Name      string
	State     string
	StartedAt time.Time
}

type Stats struct {
	CPUPercent   float64
	MemBytes     uint64
	MemLimit     uint64
	Pids         int64
	NetRxBytes   uint64
	NetTxBytes   uint64
	IOReadBytes  uint64
	IOWriteBytes uint64
}

type Info struct {
	Driver        string
	Version       string
	StorageDriver string
	Rootless      bool
	Capabilities  []string
}

type ExecRequest struct {
	Command []string
	Stdin   io.Reader
	Env     []string
	TTY     bool
}

type ExecResult struct {
	Stdout   []byte
	ExitCode int
}

type Driver interface {
	Name() string
	Info(ctx context.Context) (Info, error)
	Pull(ctx context.Context, image string) error
	Exists(ctx context.Context, image string) (bool, error)
	Run(ctx context.Context, spec RunSpec) (string, error)
	Start(ctx context.Context, id string) error
	Stop(ctx context.Context, id string) error
	Restart(ctx context.Context, id string) error
	Remove(ctx context.Context, id string, force bool) error
	Inspect(ctx context.Context, id string) (ContainerInfo, error)
	Ports(ctx context.Context, id string) (map[string]string, error)
	Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error)
	Stats(ctx context.Context, id string) (Stats, error)
	Build(ctx context.Context, dir, dockerfile, tag string) (string, error)
	BuildWithWriter(ctx context.Context, dir, dockerfile, tag string, w io.Writer) (string, error)
	UpdateResources(ctx context.Context, id string, memMB int64, cpus string) error
	NetworkCreate(ctx context.Context, name string) error
	NetworkRemove(ctx context.Context, name string) error
	VolumeCreate(ctx context.Context, name string, sizeMB int64) error
	VolumeRemove(ctx context.Context, name string) error
	Exec(ctx context.Context, id string, req ExecRequest) (*ExecResult, error)
	ExecStream(ctx context.Context, id string, req ExecRequest) (ExecStream, error)
}

type ExecStream interface {
	io.WriteCloser
	Stdout() io.Reader
	Wait() (int, error)
	Resize(cols, rows uint16) error
}
