package worker

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	types "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/container"
	dockerevents "github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

type DockerRuntime struct {
	client *client.Client
}

func NewDockerRuntime(host string) (*DockerRuntime, error) {
	options := []client.Opt{client.WithAPIVersionNegotiation()}
	if strings.TrimSpace(host) != "" {
		options = append(options, client.WithHost(host))
	} else {
		options = append(options, client.FromEnv)
	}
	engine, err := client.NewClientWithOpts(options...)
	if err != nil {
		return nil, fmt.Errorf("create Docker Engine client: %w", err)
	}
	return &DockerRuntime{client: engine}, nil
}

func (r *DockerRuntime) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

func (r *DockerRuntime) Pull(ctx context.Context, imageRef string) (string, error) {
	response, err := r.client.ImagePull(ctx, imageRef, image.PullOptions{})
	if err != nil {
		return "", runtimeError("pull image", err)
	}
	defer response.Close()
	output, readErr := readEngineOutput(response)
	if readErr != nil {
		return output, runtimeError("read image pull output", readErr)
	}
	return output, nil
}

func (r *DockerRuntime) Push(ctx context.Context, imageRef string) (string, error) {
	response, err := r.client.ImagePush(ctx, imageRef, image.PushOptions{})
	if err != nil {
		return "", runtimeError("push image", err)
	}
	defer response.Close()
	output, readErr := readEngineOutput(response)
	if readErr != nil {
		return output, runtimeError("read image push output", readErr)
	}
	return output, nil
}

func (r *DockerRuntime) Tag(ctx context.Context, source, target string) error {
	if err := r.client.ImageTag(ctx, source, target); err != nil {
		return runtimeError("tag image", err)
	}
	return nil
}

func (r *DockerRuntime) Build(ctx context.Context, dir, dockerfile, tag string) (string, error) {
	return r.BuildStream(ctx, dir, dockerfile, tag, nil)
}

func (r *DockerRuntime) BuildStream(ctx context.Context, dir, dockerfile, tag string, onLine func(string)) (string, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return "", runtimeError("resolve build context", err)
	}
	dockerfilePath, err := filepath.Abs(dockerfile)
	if err != nil {
		return "", runtimeError("resolve Dockerfile", err)
	}
	contextArchive, err := dockerContext(root)
	if err != nil {
		return "", runtimeError("create build context", err)
	}
	defer contextArchive.Close()
	relativeDockerfile, err := filepath.Rel(root, dockerfilePath)
	if err != nil {
		return "", runtimeError("resolve Dockerfile relative path", err)
	}
	if relativeDockerfile == ".." || strings.HasPrefix(relativeDockerfile, ".."+string(filepath.Separator)) {
		return "", errors.New("Dockerfile must be inside build context")
	}
	response, err := r.client.ImageBuild(ctx, contextArchive, build.ImageBuildOptions{
		Tags:        []string{tag},
		Dockerfile:  filepath.ToSlash(relativeDockerfile),
		PullParent:  true,
		Remove:      true,
		ForceRemove: true,
	})
	if err != nil {
		return "", runtimeError("build image", err)
	}
	defer response.Body.Close()
	output, readErr := readEngineOutputWithCallback(response.Body, onLine)
	if readErr != nil {
		return output, runtimeError("read image build output", readErr)
	}
	return output, nil
}

func (r *DockerRuntime) ExposedPort(ctx context.Context, imageRef string) (int, error) {
	inspected, _, err := r.client.ImageInspectWithRaw(ctx, imageRef)
	if err != nil {
		return 0, runtimeError("inspect image", err)
	}
	for exposed := range inspected.Config.ExposedPorts {
		port, err := strconv.Atoi(strings.TrimSuffix(string(exposed), "/tcp"))
		if err == nil && port > 0 {
			return port, nil
		}
	}
	return 0, errors.New("image has no exposed port")
}

func (r *DockerRuntime) ListImages(ctx context.Context) ([]ImageInfo, error) {
	items, err := r.client.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		return nil, runtimeError("list images", err)
	}
	result := make([]ImageInfo, 0, len(items))
	for _, item := range items {
		result = append(result, ImageInfo{Names: item.RepoTags, ID: item.ID, Size: item.Size, Created: item.Created})
	}
	return result, nil
}

func (r *DockerRuntime) RemoveImage(ctx context.Context, imageRef string) error {
	if _, err := r.client.ImageRemove(ctx, imageRef, image.RemoveOptions{Force: true, PruneChildren: true}); err != nil {
		return runtimeError("remove image", err)
	}
	return nil
}

func (r *DockerRuntime) FollowLogs(ctx context.Context, containerID string, writer io.Writer) error {
	logs, err := r.client.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true, ShowStderr: true, Follow: true, Tail: "100"})
	if err != nil {
		return containerError("follow container logs", err)
	}
	defer logs.Close()
	if _, err := stdcopy.StdCopy(writer, writer, logs); err != nil {
		return runtimeError("read container logs", err)
	}
	return nil
}

func (r *DockerRuntime) LogTail(ctx context.Context, containerID string, lines int) ([]string, error) {
	if lines < 0 {
		lines = 0
	}
	logs, err := r.client.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true, ShowStderr: true, Tail: strconv.Itoa(lines)})
	if err != nil {
		return nil, containerError("read container logs", err)
	}
	defer logs.Close()
	var output bytes.Buffer
	if _, err := stdcopy.StdCopy(&output, &output, logs); err != nil {
		return nil, runtimeError("read container logs", err)
	}
	trimmed := strings.TrimRight(output.String(), "\n")
	if trimmed == "" {
		return []string{}, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

func (r *DockerRuntime) ContainerState(ctx context.Context, containerID string) (string, error) {
	inspected, err := r.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", containerError("inspect container", err)
	}
	if inspected.State == nil {
		return "", errors.New("container state unavailable")
	}
	return string(inspected.State.Status), nil
}

func (r *DockerRuntime) Start(ctx context.Context, containerID string) error {
	if err := r.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return containerError("start container", err)
	}
	return nil
}

func (r *DockerRuntime) Stop(ctx context.Context, containerID string) error {
	if err := r.client.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
		return containerError("stop container", err)
	}
	return nil
}

func (r *DockerRuntime) Restart(ctx context.Context, containerID string) error {
	if err := r.client.ContainerRestart(ctx, containerID, container.StopOptions{}); err != nil {
		return containerError("restart container", err)
	}
	return nil
}

func (r *DockerRuntime) Stats(ctx context.Context, containerID string) (ContainerStats, error) {
	response, err := r.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return ContainerStats{}, containerError("read container stats", err)
	}
	defer response.Body.Close()
	var raw container.StatsResponse
	if err := json.NewDecoder(response.Body).Decode(&raw); err != nil {
		return ContainerStats{}, runtimeError("decode container stats", err)
	}
	return normalizeStats(raw), nil
}

func (r *DockerRuntime) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	return r.listContainers(ctx, true)
}

func (r *DockerRuntime) ListContainerMetadata(ctx context.Context) ([]ContainerInfo, error) {
	return r.listContainers(ctx, false)
}

func (r *DockerRuntime) listContainers(ctx context.Context, includeStats bool) ([]ContainerInfo, error) {
	containers, err := r.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, runtimeError("list containers", err)
	}
	out := make([]ContainerInfo, 0, len(containers))
	for _, item := range containers {
		name := ""
		if len(item.Names) > 0 {
			name = strings.TrimPrefix(item.Names[0], "/")
		}
		info := ContainerInfo{ID: item.ID, Name: name, State: string(item.State), Labels: item.Labels}
		if inspected, inspectErr := r.client.ContainerInspect(ctx, item.ID); inspectErr == nil && inspected.State != nil && inspected.State.Health != nil {
			switch inspected.State.Health.Status {
			case "healthy":
				healthy := true
				info.Healthy = &healthy
			case "unhealthy":
				healthy := false
				info.Healthy = &healthy
			}
		}
		if includeStats && (item.State == container.StateRunning || item.State == container.StateRestarting) {
			if stats, statsErr := r.Stats(ctx, item.ID); statsErr == nil {
				info.Stats = stats
				info.HasStats = true
			}
		}
		out = append(out, info)
	}
	return out, nil
}

type dockerRuntimeEventSubscription struct {
	events <-chan RuntimeEvent
	errors <-chan error
	cancel context.CancelFunc
}

func (s *dockerRuntimeEventSubscription) Events() <-chan RuntimeEvent { return s.events }
func (s *dockerRuntimeEventSubscription) Errors() <-chan error        { return s.errors }
func (s *dockerRuntimeEventSubscription) Close() error {
	s.cancel()
	return nil
}

func (r *DockerRuntime) SubscribeEvents(ctx context.Context, eventFilters map[string]string) (RuntimeEventSubscription, error) {
	filtersArgs := filters.NewArgs(filters.Arg("type", "container"))
	for key, value := range eventFilters {
		filtersArgs.Add(key, value)
	}
	eventCtx, cancel := context.WithCancel(ctx)
	rawEvents, rawErrors := r.client.Events(eventCtx, dockerevents.ListOptions{Filters: filtersArgs})
	eventsOut := make(chan RuntimeEvent, 32)
	errorsOut := make(chan error, 1)
	go func() {
		defer close(eventsOut)
		defer close(errorsOut)
		for {
			select {
			case raw, ok := <-rawEvents:
				if !ok {
					return
				}
				select {
				case eventsOut <- normalizeRuntimeEvent(raw):
				case <-eventCtx.Done():
					return
				}
			case err, ok := <-rawErrors:
				if ok && err != nil && !errors.Is(err, context.Canceled) {
					errorsOut <- runtimeError("read Docker events", err)
				}
				return
			case <-eventCtx.Done():
				return
			}
		}
	}()
	return &dockerRuntimeEventSubscription{events: eventsOut, errors: errorsOut, cancel: cancel}, nil
}

func normalizeRuntimeEvent(raw dockerevents.Message) RuntimeEvent {
	action := string(raw.Action)
	health := ""
	if strings.HasPrefix(action, "health_status:") {
		parts := strings.SplitN(action, ":", 2)
		health = strings.TrimSpace(parts[1])
	}
	name := raw.Actor.Attributes["name"]
	if name == "" {
		name = raw.Actor.Attributes["com.docker.compose.service"]
	}
	at := time.Unix(raw.Time, 0).UTC()
	if raw.TimeNano > 0 {
		at = time.Unix(0, raw.TimeNano).UTC()
	}
	return RuntimeEvent{ID: raw.Actor.ID, Action: action, ContainerID: raw.Actor.ID, Name: name, Status: raw.Status, Health: health, Labels: raw.Actor.Attributes, OccurredAt: at}
}

func (r *DockerRuntime) StorageUsage(ctx context.Context) (map[string]uint64, error) {
	usage, err := r.client.DiskUsage(ctx, types.DiskUsageOptions{Types: []types.DiskUsageObject{types.ContainerObject}})
	if err != nil {
		return nil, runtimeError("read container storage usage", err)
	}
	result := make(map[string]uint64, len(usage.Containers))
	for _, item := range usage.Containers {
		if item == nil {
			continue
		}
		name := ""
		if len(item.Names) > 0 {
			name = strings.TrimPrefix(item.Names[0], "/")
		}
		if item.SizeRw > 0 {
			result[name] = uint64(item.SizeRw)
		}
	}
	return result, nil
}

func (r *DockerRuntime) Exec(ctx context.Context, containerID string, env []string, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	err := r.execAttached(ctx, containerID, "", env, nil, &stdout, &stderr, args...)
	return stdout.String(), stderr.String(), err
}

func (r *DockerRuntime) ExecStream(ctx context.Context, containerID string, env []string, stdout io.Writer, stderr io.Writer, args ...string) error {
	return r.execAttached(ctx, containerID, "", env, nil, stdout, stderr, args...)
}

func (r *DockerRuntime) ExecIn(ctx context.Context, containerID string, env []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, args ...string) error {
	return r.execAttached(ctx, containerID, "", env, stdin, stdout, stderr, args...)
}

func (r *DockerRuntime) ExecAs(ctx context.Context, containerID string, user string, env []string, stdin io.Reader, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	err := r.execAttached(ctx, containerID, user, env, stdin, &stdout, &stderr, args...)
	return stdout.String(), stderr.String(), err
}

type dockerInteractiveSession struct {
	client *client.Client
	execID string
	stream types.HijackedResponse
}

func (r *DockerRuntime) OpenInteractive(ctx context.Context, containerID string, args ...string) (InteractiveSession, error) {
	created, err := r.client.ContainerExecCreate(ctx, containerID, container.ExecOptions{AttachStdin: true, AttachStdout: true, AttachStderr: true, Tty: true, Cmd: args})
	if err != nil {
		return nil, containerError("create interactive exec", err)
	}
	stream, err := r.client.ContainerExecAttach(ctx, created.ID, container.ExecStartOptions{Tty: true})
	if err != nil {
		return nil, containerError("attach interactive exec", err)
	}
	if err := r.client.ContainerExecStart(ctx, created.ID, container.ExecStartOptions{Tty: true}); err != nil {
		stream.Close()
		return nil, containerError("start interactive exec", err)
	}
	return &dockerInteractiveSession{client: r.client, execID: created.ID, stream: stream}, nil
}

func (s *dockerInteractiveSession) Read(p []byte) (int, error)  { return s.stream.Reader.Read(p) }
func (s *dockerInteractiveSession) Write(p []byte) (int, error) { return s.stream.Conn.Write(p) }
func (s *dockerInteractiveSession) Close() error {
	s.stream.Close()
	return nil
}
func (s *dockerInteractiveSession) Resize(ctx context.Context, cols, rows uint16) error {
	return s.client.ContainerExecResize(ctx, s.execID, container.ResizeOptions{Width: uint(cols), Height: uint(rows)})
}

func (r *DockerRuntime) execAttached(ctx context.Context, containerID, user string, env []string, stdin io.Reader, stdout, stderr io.Writer, args ...string) error {
	execResponse, err := r.client.ContainerExecCreate(ctx, containerID, container.ExecOptions{AttachStdin: stdin != nil, AttachStdout: true, AttachStderr: true, Env: env, Cmd: args, User: user})
	if err != nil {
		return containerError("create container exec", err)
	}
	attached, err := r.client.ContainerExecAttach(ctx, execResponse.ID, container.ExecStartOptions{Tty: false})
	if err != nil {
		return containerError("attach container exec", err)
	}
	defer attached.Close()
	if err := r.client.ContainerExecStart(ctx, execResponse.ID, container.ExecStartOptions{Tty: false}); err != nil {
		return containerError("start container exec", err)
	}
	if stdin != nil {
		if _, err := io.Copy(attached.Conn, stdin); err != nil {
			return runtimeError("write container exec input", err)
		}
		_ = attached.CloseWrite()
	}
	if _, err := stdcopy.StdCopy(stdout, stderr, attached.Reader); err != nil {
		return runtimeError("read container exec", err)
	}
	inspect, err := r.client.ContainerExecInspect(ctx, execResponse.ID)
	if err != nil {
		return containerError("inspect container exec", err)
	}
	if inspect.ExitCode != 0 {
		return fmt.Errorf("container exec exited with code %d", inspect.ExitCode)
	}
	return nil
}

func (r *DockerRuntime) Run(ctx context.Context, spec RunSpec) (string, error) {
	config := &container.Config{Image: spec.Image, Env: spec.Env, Labels: spec.Labels, Cmd: spec.Command}
	hostConfig := &container.HostConfig{}
	for _, mount := range spec.Mounts {
		mode := "rw"
		if mount.ReadOnly {
			mode = "ro"
		}
		hostConfig.Binds = append(hostConfig.Binds, mount.Source+":"+mount.Target+":"+mode)
	}
	var networking *network.NetworkingConfig
	if spec.Network != "" {
		endpoint := &network.EndpointSettings{}
		if spec.NetworkAlias != "" {
			endpoint.Aliases = []string{spec.NetworkAlias}
		}
		networking = &network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{spec.Network: endpoint}}
	}
	if spec.MemMB > 0 {
		hostConfig.Memory = int64(spec.MemMB) * 1024 * 1024
	}
	if spec.CPUs != "" {
		cpus, err := strconv.ParseFloat(spec.CPUs, 64)
		if err != nil || cpus <= 0 {
			return "", fmt.Errorf("invalid CPU limit %q", spec.CPUs)
		}
		hostConfig.NanoCPUs = int64(cpus * 1_000_000_000)
	}
	if spec.StorageMB > 0 {
		hostConfig.StorageOpt = map[string]string{"size": fmt.Sprintf("%dm", spec.StorageMB)}
	}
	if spec.ContainerPort > 0 {
		port, err := nat.NewPort("tcp", strconv.Itoa(spec.ContainerPort))
		if err != nil {
			return "", runtimeError("configure container port", err)
		}
		config.ExposedPorts = nat.PortSet{port: struct{}{}}
		publicPort := ""
		if spec.Port > 0 {
			publicPort = strconv.Itoa(spec.Port)
		}
		hostConfig.PortBindings = nat.PortMap{port: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: publicPort}}}
	}
	for _, binding := range spec.Ports {
		if binding.ContainerPort <= 0 {
			continue
		}
		port, err := nat.NewPort("tcp", strconv.Itoa(binding.ContainerPort))
		if err != nil {
			return "", runtimeError("configure container port", err)
		}
		if config.ExposedPorts == nil {
			config.ExposedPorts = nat.PortSet{}
		}
		config.ExposedPorts[port] = struct{}{}
		hostPort := ""
		if binding.HostPort > 0 {
			hostPort = strconv.Itoa(binding.HostPort)
		}
		if hostConfig.PortBindings == nil {
			hostConfig.PortBindings = nat.PortMap{}
		}
		hostConfig.PortBindings[port] = append(hostConfig.PortBindings[port], nat.PortBinding{HostIP: "0.0.0.0", HostPort: hostPort})
	}
	created, err := r.client.ContainerCreate(ctx, config, hostConfig, networking, nil, spec.Name)
	if err != nil {
		if spec.StorageMB > 0 && strings.Contains(strings.ToLower(err.Error()), "storage-opt") {
			return "", fmt.Errorf("create container: %w: %v", ErrStorageLimitUnsupported, err)
		}
		return "", containerError("create container", err)
	}
	if err := r.client.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		_ = r.client.ContainerRemove(context.Background(), created.ID, container.RemoveOptions{Force: true})
		return "", containerError("start container", err)
	}
	return created.ID, nil
}

func (r *DockerRuntime) Wait(ctx context.Context, containerID string) (int64, error) {
	statusChannel, errorChannel := r.client.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case waitErr := <-errorChannel:
		if waitErr != nil {
			return 0, runtimeError("wait for container", waitErr)
		}
	case <-ctx.Done():
		return 0, ctx.Err()
	case response := <-statusChannel:
		return response.StatusCode, nil
	}
	return 0, nil
}

func (r *DockerRuntime) RunCommand(ctx context.Context, name, imageRef, command string, env []string, remove bool) (string, error) {
	created, err := r.client.ContainerCreate(ctx, &container.Config{Image: imageRef, Env: env, Cmd: []string{"sh", "-c", command}}, &container.HostConfig{}, nil, nil, name)
	if err != nil {
		return "", containerError("create command container", err)
	}
	if remove {
		defer func() {
			_ = r.client.ContainerRemove(context.Background(), created.ID, container.RemoveOptions{Force: true})
		}()
	}
	if err := r.client.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		_ = r.client.ContainerRemove(context.Background(), created.ID, container.RemoveOptions{Force: true})
		return "", containerError("start command container", err)
	}
	if !remove {
		return created.ID, nil
	}
	statusChannel, errorChannel := r.client.ContainerWait(ctx, created.ID, container.WaitConditionNotRunning)
	select {
	case waitErr := <-errorChannel:
		if waitErr != nil {
			return "", runtimeError("wait for command container", waitErr)
		}
	case <-ctx.Done():
		return "", ctx.Err()
	case <-statusChannel:
	}
	logs, err := r.client.ContainerLogs(ctx, created.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", containerError("read command container logs", err)
	}
	defer logs.Close()
	var output bytes.Buffer
	if _, err := stdcopy.StdCopy(&output, &output, logs); err != nil {
		return output.String(), runtimeError("read command container logs", err)
	}
	return output.String(), nil
}

func (r *DockerRuntime) Port(ctx context.Context, containerID string) (string, error) {
	inspected, err := r.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", containerError("inspect container ports", err)
	}
	if inspected.NetworkSettings == nil {
		return "", errors.New("container network settings unavailable")
	}
	for _, bindings := range inspected.NetworkSettings.Ports {
		for _, binding := range bindings {
			if binding.HostPort != "" {
				return binding.HostPort, nil
			}
		}
	}
	return "", errors.New("no published port")
}

func (r *DockerRuntime) Remove(ctx context.Context, containerID string) error {
	if err := r.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		return containerError("remove container", err)
	}
	return nil
}

func (r *DockerRuntime) RemoveByLabel(ctx context.Context, label string) error {
	items, err := r.client.ContainerList(ctx, container.ListOptions{All: true, Filters: filters.NewArgs(filters.Arg("label", label))})
	if err != nil {
		return runtimeError("find containers by label", err)
	}
	for _, item := range items {
		if err := r.Remove(ctx, item.ID); err != nil {
			return err
		}
	}
	return nil
}

func (r *DockerRuntime) EnsureNetwork(ctx context.Context, name string, labels map[string]string) error {
	if _, err := r.client.NetworkInspect(ctx, name, network.InspectOptions{}); err == nil {
		return nil
	} else if !client.IsErrNotFound(err) {
		return runtimeError("inspect network", err)
	}
	_, err := r.client.NetworkCreate(ctx, name, network.CreateOptions{Driver: "bridge", Labels: labels})
	if err != nil {
		return runtimeError("create network", err)
	}
	return nil
}

func (r *DockerRuntime) RemoveNetwork(ctx context.Context, name string) error {
	if err := r.client.NetworkRemove(ctx, name); err != nil {
		return runtimeError("remove network", err)
	}
	return nil
}

func (r *DockerRuntime) CreateVolume(ctx context.Context, name string, labels map[string]string) error {
	_, err := r.client.VolumeCreate(ctx, volume.CreateOptions{Name: name, Driver: "local", Labels: labels})
	if err != nil {
		return runtimeError("create volume", err)
	}
	return nil
}

func (r *DockerRuntime) RemoveVolume(ctx context.Context, name string, force bool) error {
	if err := r.client.VolumeRemove(ctx, name, force); err != nil {
		return runtimeError("remove volume", err)
	}
	return nil
}

func (r *DockerRuntime) HealthCheck(ctx context.Context, hostPort, path string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	var lastErr error
	for _, host := range []string{"host.docker.internal", "127.0.0.1"} {
		url := "http://" + host + ":" + hostPort + path
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		response, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if response.StatusCode < http.StatusInternalServerError {
			response.Body.Close()
			return nil
		}
		response.Body.Close()
		lastErr = fmt.Errorf("health check returned HTTP %d", response.StatusCode)
	}
	if lastErr == nil {
		lastErr = errors.New("health check failed")
	}
	return lastErr
}

func normalizeStats(raw container.StatsResponse) ContainerStats {
	cpuDelta := float64(raw.CPUStats.CPUUsage.TotalUsage - raw.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(raw.CPUStats.SystemUsage - raw.PreCPUStats.SystemUsage)
	onlineCPUs := raw.CPUStats.OnlineCPUs
	if onlineCPUs == 0 {
		onlineCPUs = uint32(len(raw.CPUStats.CPUUsage.PercpuUsage))
	}
	cpuPercent := 0.0
	if cpuDelta > 0 && systemDelta > 0 && onlineCPUs > 0 {
		cpuPercent = cpuDelta / systemDelta * float64(onlineCPUs) * 100
	}
	var netInput, netOutput uint64
	for _, stats := range raw.Networks {
		netInput += stats.RxBytes
		netOutput += stats.TxBytes
	}
	var blockInput, blockOutput uint64
	for _, entry := range raw.BlkioStats.IoServiceBytesRecursive {
		switch strings.ToLower(entry.Op) {
		case "read":
			blockInput += entry.Value
		case "write":
			blockOutput += entry.Value
		}
	}
	memBytes := raw.MemoryStats.Usage
	memLimit := raw.MemoryStats.Limit
	memPercent := 0.0
	if memLimit > 0 {
		memPercent = float64(memBytes) / float64(memLimit) * 100
	}
	return ContainerStats{CPUPercent: cpuPercent, MemUsage: memBytes, MemBytes: memBytes, MemLimit: memLimit, MemPercent: memPercent, NetInput: netInput, NetOutput: netOutput, BlockInput: blockInput, BlockOutput: blockOutput}
}

func readEngineOutput(reader io.Reader) (string, error) {
	return readEngineOutputWithCallback(reader, nil)
}

func readEngineOutputWithCallback(reader io.Reader, onLine func(string)) (string, error) {
	const maxEngineOutputBytes int64 = 16 << 20
	var output strings.Builder
	var pendingLine string
	limitedReader := &io.LimitedReader{R: reader, N: maxEngineOutputBytes + 1}
	decoder := json.NewDecoder(limitedReader)
	for {
		var event struct {
			Stream      string `json:"stream"`
			Error       string `json:"error"`
			ErrorDetail struct {
				Message string `json:"message"`
			} `json:"errorDetail"`
		}
		if err := decoder.Decode(&event); err != nil {
			if errors.Is(err, io.EOF) {
				if onLine != nil && strings.TrimSpace(pendingLine) != "" {
					onLine(strings.TrimRight(pendingLine, "\r"))
				}
				if limitedReader.N == 0 {
					return output.String(), fmt.Errorf("engine response exceeds %d bytes", maxEngineOutputBytes)
				}
				return output.String(), nil
			}
			if limitedReader.N == 0 {
				return output.String(), fmt.Errorf("engine response exceeds %d bytes", maxEngineOutputBytes)
			}
			return output.String(), err
		}
		if event.Stream != "" {
			output.WriteString(event.Stream)
			if onLine != nil {
				emitOutputLines(&pendingLine, event.Stream, onLine)
			}
		}
		if event.Error != "" {
			return output.String(), errors.New(event.Error)
		}
		if event.ErrorDetail.Message != "" {
			return output.String(), errors.New(event.ErrorDetail.Message)
		}
	}
}

func emitOutputLines(pending *string, chunk string, onLine func(string)) {
	value := strings.ReplaceAll(*pending+chunk, "\r\n", "\n")
	parts := strings.Split(value, "\n")
	*pending = parts[len(parts)-1]
	for _, line := range parts[:len(parts)-1] {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) != "" {
			onLine(line)
		}
	}
}

func dockerContext(dir string) (io.ReadCloser, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("build context is not a directory")
	}
	reader, writer := io.Pipe()
	go func() {
		_ = writer.CloseWithError(writeTar(root, writer))
	}()
	return reader, nil
}

func writeTar(root string, writer *io.PipeWriter) error {
	archive := tar.NewWriter(writer)
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relative)
		if err := archive.WriteHeader(header); err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(archive, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	closeErr := archive.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func runtimeError(operation string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return fmt.Errorf("%s: %w: %v", operation, ErrRuntimeTimeout, err)
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "permission denied") || strings.Contains(message, "forbidden") || strings.Contains(message, "unauthorized") {
		return fmt.Errorf("%s: %w: %v", operation, ErrRuntimePermission, err)
	}
	if client.IsErrConnectionFailed(err) {
		return fmt.Errorf("%s: %w: %v", operation, ErrRuntimeUnavailable, err)
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func containerError(operation string, err error) error {
	if client.IsErrNotFound(err) {
		return fmt.Errorf("%s: %w: %v", operation, ErrContainerNotFound, err)
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "is not running") || strings.Contains(message, "container is stopped") || strings.Contains(message, "container has stopped") {
		return fmt.Errorf("%s: %w: %v", operation, ErrContainerStopped, err)
	}
	return runtimeError(operation, err)
}

var _ Runtime = (*DockerRuntime)(nil)
var _ EventRuntime = (*DockerRuntime)(nil)
var _ CommandRuntime = (*DockerRuntime)(nil)
var _ WaitRuntime = (*DockerRuntime)(nil)
var _ InteractiveRuntime = (*DockerRuntime)(nil)
var _ StorageRuntime = (*DockerRuntime)(nil)
var _ NetworkRuntime = (*DockerRuntime)(nil)
var _ VolumeRuntime = (*DockerRuntime)(nil)
