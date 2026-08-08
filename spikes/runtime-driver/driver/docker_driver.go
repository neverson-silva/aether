package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DockerDriver implementa RuntimeDriver via CLI `docker`.
// Objetivo do spike: validar a semântica da porta com um engine real
// (stand-in do PodmanDriver, mesmo modelo OCI).
type DockerDriver struct {
	Bin string
}

func NewDockerDriver() *DockerDriver { return &DockerDriver{Bin: "docker"} }

func (d *DockerDriver) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, d.Bin, args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", &DriverError{Code: ErrDriver, Msg: msg}
	}
	return strings.TrimSpace(out.String()), nil
}

func (d *DockerDriver) Info(ctx context.Context) (*RuntimeInfo, error) {
	out, err := d.run(ctx, "version", "--format", "{{.Server.Version}}")
	if err != nil {
		return nil, err
	}
	store, _ := d.run(ctx, "info", "--format", "{{.Driver}}")
	rootless := false
	return &RuntimeInfo{
		Driver:        "docker",
		Version:       out,
		StorageDriver: store,
		Rootless:      rootless,
		Capabilities:  []string{"cgroupv2", "overlay2", "user-namespaces"},
	}, nil
}

func (d *DockerDriver) Pull(ctx context.Context, ref ImageRef, auth *RegistryAuth) (string, error) {
	_, err := d.run(ctx, "pull", ref.String())
	if err != nil {
		return "", err
	}
	img, err := d.InspectImage(ctx, ref)
	if err != nil {
		return "", err
	}
	return img.Digest, nil
}

func (d *DockerDriver) Push(ctx context.Context, ref ImageRef, auth *RegistryAuth) error {
	_, err := d.run(ctx, "push", ref.String())
	return err
}

func (d *DockerDriver) InspectImage(ctx context.Context, ref ImageRef) (*ImageInfo, error) {
	raw, err := d.run(ctx, "inspect", "--format", "{{json .}}", ref.String())
	if err != nil {
		return nil, &DriverError{Code: ErrImageNotFound, Msg: ref.String()}
	}
	var img struct {
		ID      string `json:"Id"`
		Size    int64  `json:"Size"`
		Created string `json:"Created"`
		RootFS  struct {
			Layers []string `json:"Layers"`
		} `json:"RootFS"`
	}
	_ = json.Unmarshal([]byte(raw), &img)
	created, _ := time.Parse(time.RFC3339, img.Created)
	return &ImageInfo{
		Ref:     ref.String(),
		Size:    img.Size,
		Created: created,
		Layers:  len(img.RootFS.Layers),
		Digest:  img.ID,
	}, nil
}

func (d *DockerDriver) ListImages(ctx context.Context) ([]ImageInfo, error) {
	raw, err := d.run(ctx, "images", "--format", "{{.Repository}}:{{.Tag}}")
	if err != nil {
		return nil, err
	}
	var out []ImageInfo
	for _, line := range strings.Split(raw, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, ImageInfo{Ref: line})
	}
	return out, nil
}

func (d *DockerDriver) PruneImages(ctx context.Context, policy PrunePolicy) (*PruneReport, error) {
	args := []string{"image", "prune"}
	if policy.UnusedOnly {
		args = append(args, "--filter", "dangling=true")
	}
	_, err := d.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return &PruneReport{}, nil
}

func (d *DockerDriver) Run(ctx context.Context, spec ContainerSpec) (*ContainerHandle, error) {
	args := []string{"run", "-d", "--name", spec.Name}
	for _, e := range spec.Env {
		args = append(args, "-e", e)
	}
	for _, p := range spec.Ports {
		args = append(args, "-p", p.HostPort+":"+p.ContainerPort)
	}
	for _, v := range spec.Volumes {
		args = append(args, "-v", v.Source+":"+v.Target)
	}
	for _, n := range spec.Networks {
		args = append(args, "--network", n)
	}
	if spec.Resources.MemMB > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dm", spec.Resources.MemMB))
	}
	if spec.Resources.CPUs > 0 {
		args = append(args, "--cpus", fmt.Sprintf("%g", spec.Resources.CPUs))
	}
	if spec.ReadOnly {
		args = append(args, "--read-only")
	}
	args = append(args, spec.Image.String())
	args = append(args, spec.Command...)
	id, err := d.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return &ContainerHandle{ID: id, Name: spec.Name}, nil
}

func (d *DockerDriver) Start(ctx context.Context, id string) error {
	_, err := d.run(ctx, "start", id)
	return err
}

func (d *DockerDriver) Stop(ctx context.Context, id string, timeout time.Duration) error {
	args := []string{"stop"}
	if timeout > 0 {
		args = append(args, "--time", fmt.Sprintf("%d", int(timeout.Seconds())))
	}
	args = append(args, id)
	_, err := d.run(ctx, args...)
	return err
}

func (d *DockerDriver) Restart(ctx context.Context, id string, timeout time.Duration) error {
	args := []string{"restart"}
	if timeout > 0 {
		args = append(args, "--time", fmt.Sprintf("%d", int(timeout.Seconds())))
	}
	args = append(args, id)
	_, err := d.run(ctx, args...)
	return err
}

func (d *DockerDriver) Remove(ctx context.Context, id string, force bool) error {
	args := []string{"rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, id)
	_, err := d.run(ctx, args...)
	return err
}

func (d *DockerDriver) Inspect(ctx context.Context, id string) (*ContainerInfo, error) {
	raw, err := d.run(ctx, "inspect", "--format", "{{json .}}", id)
	if err != nil {
		return nil, &DriverError{Code: ErrContainerNotFound, Msg: id}
	}
	var c struct {
		ID     string `json:"Id"`
		Name   string `json:"Name"`
		Config struct {
			Image string `json:"Image"`
		} `json:"Config"`
		State struct {
			Status    string `json:"Status"`
			StartedAt string `json:"StartedAt"`
		} `json:"State"`
		NetworkSettings struct {
			Networks map[string]json.RawMessage `json:"Networks"`
		} `json:"NetworkSettings"`
	}
	_ = json.Unmarshal([]byte(raw), &c)
	started, _ := time.Parse(time.RFC3339Nano, c.State.StartedAt)
	networks := []string{}
	for k := range c.NetworkSettings.Networks {
		networks = append(networks, k)
	}
	return &ContainerInfo{
		ID:        c.ID,
		Name:      strings.TrimPrefix(c.Name, "/"),
		Image:     c.Config.Image,
		State:     c.State.Status,
		Status:    c.State.Status,
		Networks:  networks,
		StartedAt: started,
	}, nil
}

func (d *DockerDriver) List(ctx context.Context) ([]ContainerInfo, error) {
	raw, err := d.run(ctx, "ps", "-a", "--format", "{{.ID}} {{.Names}}")
	if err != nil {
		return nil, err
	}
	var out []ContainerInfo
	for _, line := range strings.Split(raw, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) == 2 {
			out = append(out, ContainerInfo{ID: f[0], Name: f[1]})
		}
	}
	return out, nil
}

func (d *DockerDriver) Exec(ctx context.Context, id string, req ExecRequest) (*ExecResult, error) {
	args := []string{"exec", id}
	args = append(args, req.Command...)
	var out, errb bytes.Buffer
	cmd := exec.CommandContext(ctx, d.Bin, args...)
	cmd.Stdout = &out
	cmd.Stderr = &errb
	code := 0
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			return nil, err
		}
	}
	res := &ExecResult{Stdout: out.Bytes(), ExitCode: code}
	if errb.Len() > 0 {
		res.Stdout = append(res.Stdout, errb.Bytes()...)
	}
	return res, nil
}

func (d *DockerDriver) Logs(ctx context.Context, id string, req LogRequest) (*LogStream, error) {
	args := []string{"logs"}
	if req.Follow {
		args = append(args, "-f")
	}
	args = append(args, id)
	cmd := exec.CommandContext(ctx, d.Bin, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &LogStream{Reader: stdout, Close: func() error { return cmd.Wait() }}, nil
}

func (d *DockerDriver) Stats(ctx context.Context, id string) (*ContainerStats, error) {
	raw, err := d.run(ctx, "stats", "--no-stream", "--format", "{{json .}}", id)
	if err != nil {
		return nil, err
	}
	var s struct {
		MemUsage string `json:"MemUsage"`
		MemPerc  string `json:"MemPerc"`
		CPUPerc  string `json:"CPUPerc"`
		PIDs     string `json:"PIDs"`
		NetIO    string `json:"NetIO"`
		BlockIO  string `json:"BlockIO"`
	}
	_ = json.Unmarshal([]byte(raw), &s)
	return &ContainerStats{CpuPercent: parsePerc(s.CPUPerc), MemBytes: parseMemUsage(s.MemUsage), Pids: parseInt(s.PIDs)}, nil
}

func (d *DockerDriver) NetworkCreate(ctx context.Context, spec NetworkSpec) (string, error) {
	args := []string{"network", "create"}
	if spec.Driver != "" {
		args = append(args, "--driver", spec.Driver)
	}
	if spec.Internal {
		args = append(args, "--internal")
	}
	if spec.Subnet != "" {
		args = append(args, "--subnet", spec.Subnet)
	}
	args = append(args, spec.Name)
	return d.run(ctx, args...)
}

func (d *DockerDriver) NetworkRemove(ctx context.Context, id string) error {
	_, err := d.run(ctx, "network", "rm", id)
	return err
}

func (d *DockerDriver) NetworkInspect(ctx context.Context, id string) (*NetworkInfo, error) {
	raw, err := d.run(ctx, "network", "inspect", "--format", "{{json .}}", id)
	if err != nil {
		return nil, &DriverError{Code: ErrNetworkNotFound, Msg: id}
	}
	var n struct {
		ID       string `json:"Id"`
		Name     string `json:"Name"`
		Driver   string `json:"Driver"`
		Internal bool   `json:"Internal"`
		IPAM     struct {
			Config []struct {
				Subnet string `json:"Subnet"`
			} `json:"Config"`
		} `json:"IPAM"`
	}
	_ = json.Unmarshal([]byte(raw), &n)
	subnet := ""
	if len(n.IPAM.Config) > 0 {
		subnet = n.IPAM.Config[0].Subnet
	}
	return &NetworkInfo{ID: n.ID, Name: n.Name, Driver: n.Driver, Internal: n.Internal, Subnet: subnet}, nil
}

func (d *DockerDriver) NetworkConnect(ctx context.Context, netID, containerID string) error {
	_, err := d.run(ctx, "network", "connect", netID, containerID)
	return err
}

func (d *DockerDriver) NetworkDisconnect(ctx context.Context, netID, containerID string) error {
	_, err := d.run(ctx, "network", "disconnect", netID, containerID)
	return err
}

func (d *DockerDriver) VolumeCreate(ctx context.Context, spec VolumeSpec) (string, error) {
	args := []string{"volume", "create"}
	if spec.Driver != "" {
		args = append(args, "--driver", spec.Driver)
	}
	args = append(args, spec.Name)
	return d.run(ctx, args...)
}

func (d *DockerDriver) VolumeRemove(ctx context.Context, id string, force bool) error {
	args := []string{"volume", "rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, id)
	_, err := d.run(ctx, args...)
	return err
}

func (d *DockerDriver) VolumeInspect(ctx context.Context, id string) (*VolumeInfo, error) {
	raw, err := d.run(ctx, "volume", "inspect", "--format", "{{json .}}", id)
	if err != nil {
		return nil, &DriverError{Code: ErrVolumeNotFound, Msg: id}
	}
	var v struct {
		Name   string `json:"Name"`
		Driver string `json:"Driver"`
	}
	_ = json.Unmarshal([]byte(raw), &v)
	return &VolumeInfo{ID: v.Name, Name: v.Name, Driver: v.Driver}, nil
}

func (d *DockerDriver) GC(ctx context.Context, req GCRequest) (*GCReport, error) {
	p, err := d.PruneImages(ctx, req.Images)
	if err != nil {
		return nil, err
	}
	return &GCReport{Prune: *p}, nil
}

// helpers
func parsePerc(s string) float64 {
	s = strings.TrimSpace(strings.TrimSuffix(s, "%"))
	var v float64
	fmt.Sscanf(s, "%f", &v)
	return v
}
func parseMemUsage(s string) uint64 {
	// "3.2MiB / 24GiB"
	part := strings.SplitN(s, "/", 2)[0]
	part = strings.TrimSpace(part)
	var v float64
	var unit string
	fmt.Sscanf(part, "%f%s", &v, &unit)
	mult := uint64(1)
	switch unit {
	case "KiB":
		mult = 1 << 10
	case "MiB":
		mult = 1 << 20
	case "GiB":
		mult = 1 << 30
	}
	return uint64(v) * mult
}
func parseInt(s string) int64 {
	var v int64
	fmt.Sscanf(strings.TrimSpace(s), "%d", &v)
	return v
}
