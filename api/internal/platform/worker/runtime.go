package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
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
}

type Runtime interface {
	Pull(ctx context.Context, image string) (output string, err error)
	Build(ctx context.Context, dir, dockerfile, tag string) (output string, err error)
	ExposedPort(ctx context.Context, image string) (int, error)
	Run(ctx context.Context, spec RunSpec) (containerID string, err error)
	Port(ctx context.Context, containerID string) (hostPort string, err error)
	HealthCheck(ctx context.Context, hostPort, path string) error
	Remove(ctx context.Context, containerID string) error
	RemoveByLabel(ctx context.Context, label string) error
	FollowLogs(ctx context.Context, containerID string, writer io.Writer) error
	LogTail(ctx context.Context, containerID string, lines int) ([]string, error)
	ContainerState(ctx context.Context, containerID string) (string, error)
	Start(ctx context.Context, containerID string) error
	Stop(ctx context.Context, containerID string) error
	Restart(ctx context.Context, containerID string) error
	Stats(ctx context.Context, containerID string) (ContainerStats, error)
	ListContainers(ctx context.Context) ([]ContainerInfo, error)
	Exec(ctx context.Context, containerID string, env []string, args ...string) (stdout string, stderr string, err error)
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
	Stats    ContainerStats    `json:"stats"`
	HasStats bool              `json:"has_stats"`
}

type podmanRuntime struct{}

func NewPodmanRuntime() Runtime {
	return podmanRuntime{}
}

func (podmanRuntime) Pull(ctx context.Context, image string) (string, error) {
	out, err := exec.CommandContext(ctx, "podman", "pull", image).CombinedOutput()
	return string(out), err
}

func (podmanRuntime) Build(ctx context.Context, dir, dockerfile, tag string) (string, error) {
	args := []string{"build", "-t", tag, "-f", dockerfile, "--pull", dir}
	out, err := exec.CommandContext(ctx, "podman", args...).CombinedOutput()
	return string(out), err
}

func (podmanRuntime) ExposedPort(ctx context.Context, image string) (int, error) {
	out, err := exec.CommandContext(ctx, "podman", "image", "inspect", "--format", "{{json .Config.ExposedPorts}}", image).CombinedOutput()
	if err != nil {
		return 0, err
	}
	var ports map[string]any
	if err := json.Unmarshal(out, &ports); err != nil {
		return 0, err
	}
	for k := range ports {
		n := strings.TrimSuffix(k, "/tcp")
		if p, err := strconv.Atoi(n); err == nil && p > 0 {
			return p, nil
		}
	}
	return 0, fmt.Errorf("imagem no port exposta")
}

func (podmanRuntime) FollowLogs(ctx context.Context, containerID string, writer io.Writer) error {
	cmd := exec.CommandContext(ctx, "podman", "logs", "-f", "--tail", "100", containerID)
	cmd.Stdout = writer
	cmd.Stderr = writer
	return cmd.Run()
}

func (podmanRuntime) LogTail(ctx context.Context, containerID string, lines int) ([]string, error) {
	out, err := exec.CommandContext(ctx, "podman", "logs", "--tail", strconv.Itoa(lines), containerID).CombinedOutput()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimRight(string(out), "\n"), "\n"), nil
}

func (podmanRuntime) ContainerState(ctx context.Context, containerID string) (string, error) {
	out, err := exec.CommandContext(ctx, "podman", "inspect", "--format", "{{.State.Status}}", containerID).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("inspect container %s: %s", containerID, message)
	}
	return strings.TrimSpace(string(out)), nil
}

func (podmanRuntime) Start(ctx context.Context, containerID string) error {
	_, err := exec.CommandContext(ctx, "podman", "start", containerID).CombinedOutput()
	return err
}

func (podmanRuntime) Stop(ctx context.Context, containerID string) error {
	_, err := exec.CommandContext(ctx, "podman", "stop", containerID).CombinedOutput()
	return err
}

func (podmanRuntime) Restart(ctx context.Context, containerID string) error {
	_, err := exec.CommandContext(ctx, "podman", "restart", containerID).CombinedOutput()
	return err
}

func (podmanRuntime) Stats(ctx context.Context, containerID string) (ContainerStats, error) {
	out, err := exec.CommandContext(ctx, "podman", "stats", "--no-stream", "--format", "json", containerID).CombinedOutput()
	if err != nil {
		return ContainerStats{}, err
	}
	var rows []struct {
		CPUPerc  string `json:"cpu_percent"`
		MemUsage string `json:"mem_usage"`
		MemPerc  string `json:"mem_percent"`
		NetIO    string `json:"net_io"`
		BlockIO  string `json:"block_io"`
	}
	if err := json.Unmarshal(out, &rows); err != nil || len(rows) == 0 {
		return ContainerStats{}, err
	}
	row := rows[0]
	mem, memLimit := splitBytesPair(row.MemUsage)
	netIn, netOut := splitBytesPair(row.NetIO)
	blockIn, blockOut := splitBytesPair(row.BlockIO)
	return ContainerStats{
		CPUPercent:  parsePercent(row.CPUPerc),
		MemUsage:    mem,
		MemBytes:    mem,
		MemLimit:    memLimit,
		MemPercent:  parsePercent(row.MemPerc),
		NetInput:    netIn,
		NetOutput:   netOut,
		BlockInput:  blockIn,
		BlockOutput: blockOut,
	}, nil
}

func splitBytesPair(s string) (uint64, uint64) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, 0
	}
	return parseBytes(strings.TrimSpace(parts[0])), parseBytes(strings.TrimSpace(parts[1]))
}

func parseBytes(s string) uint64 {
	s = strings.TrimSpace(s)
	mult := uint64(1)
	switch {
	case strings.HasSuffix(s, "kB"):
		mult, s = 1000, strings.TrimSuffix(s, "kB")
	case strings.HasSuffix(s, "KB"):
		mult, s = 1000, strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "MB"):
		mult, s = 1000*1000, strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "GB"):
		mult, s = 1000*1000*1000, strings.TrimSuffix(s, "GB")
	case strings.HasSuffix(s, "B"):
		mult, s = 1, strings.TrimSuffix(s, "B")
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return uint64(v * float64(mult))
}

func parsePercent(s string) float64 {
	s = strings.TrimSpace(strings.TrimSuffix(s, "%"))
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func (podmanRuntime) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	psOut, err := exec.CommandContext(ctx, "podman", "ps", "-a", "--format", "json").CombinedOutput()
	if err != nil {
		return nil, err
	}
	var psRows []struct {
		Id     string            `json:"Id"`
		Names  []string          `json:"Names"`
		State  string            `json:"State"`
		Labels map[string]string `json:"Labels"`
	}
	if err := json.Unmarshal(psOut, &psRows); err != nil {
		return nil, err
	}

	statsByID := map[string]ContainerStats{}
	if stOut, err := exec.CommandContext(ctx, "podman", "stats", "--no-stream", "--format", "json").CombinedOutput(); err == nil {
		var stRows []struct {
			ID       string `json:"id"`
			CPUPerc  string `json:"cpu_percent"`
			MemUsage string `json:"mem_usage"`
			MemLimit string `json:"mem_limit"`
			MemPerc  string `json:"mem_percent"`
			NetIO    string `json:"net_io"`
			BlockIO  string `json:"block_io"`
		}
		if json.Unmarshal(stOut, &stRows) == nil {
			for _, r := range stRows {
				mem, memLimit := splitBytesPair(r.MemUsage)
				netIn, netOut := splitBytesPair(r.NetIO)
				blockIn, blockOut := splitBytesPair(r.BlockIO)
				id := strings.TrimPrefix(r.ID, "sha256:")
				statsByID[id] = ContainerStats{
					CPUPercent:  parsePercent(r.CPUPerc),
					MemUsage:    mem,
					MemBytes:    mem,
					MemLimit:    memLimit,
					MemPercent:  parsePercent(r.MemPerc),
					NetInput:    netIn,
					NetOutput:   netOut,
					BlockInput:  blockIn,
					BlockOutput: blockOut,
				}
			}
		}
	}

	out := make([]ContainerInfo, 0, len(psRows))
	for _, r := range psRows {
		id := strings.TrimPrefix(r.Id, "sha256:")
		name := ""
		if len(r.Names) > 0 {
			name = r.Names[0]
		}
		info := ContainerInfo{ID: id, Name: name, State: r.State, Labels: r.Labels}
		// podman ps returns the full id while podman stats returns the short
		// id; match by prefix to pair them up.
		for stID, st := range statsByID {
			if strings.HasPrefix(id, stID) {
				info.Stats = st
				info.HasStats = true
				break
			}
		}
		out = append(out, info)
	}
	return out, nil
}

func (podmanRuntime) Exec(ctx context.Context, containerID string, env []string, args ...string) (string, string, error) {
	execArgs := []string{"exec"}
	for _, e := range env {
		execArgs = append(execArgs, "-e", e)
	}
	execArgs = append(execArgs, containerID)
	execArgs = append(execArgs, args...)
	cmd := exec.CommandContext(ctx, "podman", execArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func (podmanRuntime) Run(ctx context.Context, spec RunSpec) (string, error) {
	args := []string{"run", "-d", "--quiet", "--name", spec.Name}
	if spec.Network != "" {
		args = append(args, "--network", spec.Network)
	}
	if spec.NetworkAlias != "" && spec.Network != "" {
		args = append(args, "--network-alias", spec.NetworkAlias)
	}
	for _, e := range spec.Env {
		args = append(args, "-e", e)
	}
	if spec.MemMB > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dm", spec.MemMB))
	}
	if spec.CPUs != "" {
		args = append(args, "--cpus", spec.CPUs)
	}
	if spec.StorageMB > 0 {
		args = append(args, "--storage-opt", fmt.Sprintf("size=%dm", spec.StorageMB))
	}
	for k, v := range spec.Labels {
		args = append(args, "--label", k+"="+v)
	}
	if spec.Port > 0 {
		if spec.ContainerPort > 0 {
			args = append(args, "-p", fmt.Sprintf("0.0.0.0:%d:%d", spec.Port, spec.ContainerPort))
		} else {
			args = append(args, "-p", fmt.Sprintf("0.0.0.0:%d:%d", spec.Port, spec.Port))
		}
	} else if spec.ContainerPort > 0 {
		args = append(args, "-p", fmt.Sprintf("0.0.0.0::%d", spec.ContainerPort))
	}
	args = append(args, spec.Image)
	out, err := exec.CommandContext(ctx, "podman", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (podmanRuntime) Port(ctx context.Context, containerID string) (string, error) {
	out, err := exec.CommandContext(ctx, "podman", "port", containerID).CombinedOutput()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return "", fmt.Errorf("no port publicada")
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", fmt.Errorf("no port publicada")
	}
	host := fields[len(fields)-1]
	if idx := strings.LastIndex(host, ":"); idx >= 0 {
		host = host[idx+1:]
	}
	return host, nil
}

func (podmanRuntime) Remove(ctx context.Context, containerID string) error {
	_, err := exec.CommandContext(ctx, "podman", "rm", "-f", containerID).CombinedOutput()
	return err
}

func (podmanRuntime) RemoveWithVolumes(ctx context.Context, containerID string) error {
	_, err := exec.CommandContext(ctx, "podman", "rm", "-f", "--volumes", containerID).CombinedOutput()
	return err
}

func (podmanRuntime) RemoveByLabel(ctx context.Context, label string) error {
	out, err := exec.CommandContext(ctx, "podman", "ps", "-aq", "--filter", "label="+label).CombinedOutput()
	if err != nil {
		return err
	}
	for _, id := range strings.Fields(string(out)) {
		_, _ = exec.CommandContext(ctx, "podman", "rm", "-f", id).CombinedOutput()
	}
	return nil
}

func (podmanRuntime) HealthCheck(ctx context.Context, hostPort, path string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	var lastErr error
	for _, host := range []string{"host.containers.internal", "127.0.0.1"} {
		url := "http://" + host + ":" + hostPort + path
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("health status %d", resp.StatusCode)
			continue
		}
		return nil
	}
	return lastErr
}
