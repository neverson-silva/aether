package worker

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	dockerevents "github.com/docker/docker/api/types/events"
)

func TestNewDockerRuntimeUsesConfiguredHost(t *testing.T) {
	server := httptest.NewServer(nil)
	defer server.Close()
	runtime, err := NewDockerRuntime(server.URL)
	if err != nil {
		t.Fatalf("create Docker runtime: %v", err)
	}
	defer runtime.Close()
	if got := runtime.client.DaemonHost(); got != server.URL {
		t.Fatalf("daemon host = %q, want %q", got, server.URL)
	}
}

func TestDockerRuntimeNormalizesMissingContainer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"No such container"}`))
	}))
	defer server.Close()
	runtime, err := NewDockerRuntime(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	_, err = runtime.ContainerState(context.Background(), "missing")
	if !errors.Is(err, ErrContainerNotFound) {
		t.Fatalf("error = %v, want ErrContainerNotFound", err)
	}
}

func TestNormalizeStats(t *testing.T) {
	stats := normalizeStats(container.StatsResponse{
		CPUStats:    container.CPUStats{CPUUsage: container.CPUUsage{TotalUsage: 300}, SystemUsage: 1_000, OnlineCPUs: 2},
		PreCPUStats: container.CPUStats{CPUUsage: container.CPUUsage{TotalUsage: 100}, SystemUsage: 500},
		MemoryStats: container.MemoryStats{Usage: 256, Limit: 1_024},
		Networks: map[string]container.NetworkStats{
			"bridge": {RxBytes: 10, TxBytes: 20},
			"aether": {RxBytes: 30, TxBytes: 40},
		},
		BlkioStats: container.BlkioStats{IoServiceBytesRecursive: []container.BlkioStatEntry{
			{Op: "Read", Value: 50}, {Op: "Write", Value: 70},
		}},
	})
	if stats.CPUPercent != 80 {
		t.Fatalf("CPU percent = %f, want 80", stats.CPUPercent)
	}
	if stats.MemBytes != 256 || stats.MemLimit != 1_024 || stats.MemPercent != 25 {
		t.Fatalf("memory stats = %+v", stats)
	}
	if stats.NetInput != 40 || stats.NetOutput != 60 || stats.BlockInput != 50 || stats.BlockOutput != 70 {
		t.Fatalf("IO stats = %+v", stats)
	}
}

func TestNormalizeRuntimeEvent(t *testing.T) {
	raw := dockerevents.Message{
		Status:   "running",
		Action:   "health_status: healthy",
		Time:     10,
		TimeNano: 10_000_000_001,
		Actor: dockerevents.Actor{
			ID: "container-1",
			Attributes: map[string]string{
				"name":                       "web",
				"aether.service-id":          "service-1",
				"com.docker.compose.service": "web",
			},
		},
	}
	event := normalizeRuntimeEvent(raw)
	if event.ContainerID != "container-1" || event.Name != "web" || event.Health != "healthy" || event.Status != "running" {
		t.Fatalf("normalized event = %+v", event)
	}
	if event.Labels["aether.service-id"] != "service-1" || event.OccurredAt.UnixNano() != 10_000_000_001 {
		t.Fatalf("normalized event metadata = %+v", event)
	}
}

func TestDockerContextContainsRelativeFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Dockerfile"), []byte("FROM alpine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "app.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	archive, err := dockerContext(root)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	entries := map[string]string{}
	reader := tar.NewReader(archive)
	for {
		header, err := reader.Next()
		if err != nil {
			break
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		entries[header.Name] = string(data)
	}
	if entries["Dockerfile"] != "FROM alpine\n" || entries["nested/app.txt"] != "ok" {
		t.Fatalf("archive entries = %+v", entries)
	}
}

func TestDockerContextRejectsMissingDirectory(t *testing.T) {
	_, err := dockerContext(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected missing build context error")
	}
}

func TestDockerRuntimeCloseIsIdempotent(t *testing.T) {
	runtime, err := NewDockerRuntime("")
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDockerRuntimeNormalizesOperationalErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "timeout", err: context.DeadlineExceeded, want: ErrRuntimeTimeout},
		{name: "permission", err: errors.New("permission denied"), want: ErrRuntimePermission},
		{name: "stopped", err: errors.New("container is not running"), want: ErrContainerStopped},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got error
			if test.name == "stopped" {
				got = containerError("inspect", test.err)
			} else {
				got = runtimeError("inspect", test.err)
			}
			if !errors.Is(got, test.want) {
				t.Fatalf("error = %v, want %v", got, test.want)
			}
		})
	}
}

func TestReadEngineOutputBoundsResponse(t *testing.T) {
	large := fmt.Sprintf(`{"stream":"%s"}`, strings.Repeat("x", 17<<20))
	if _, err := readEngineOutput(strings.NewReader(large)); err == nil {
		t.Fatal("expected oversized engine response error")
	}
}

func TestDockerRuntimeRealEngine(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	runtime, err := NewDockerRuntime("")
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	if _, err := runtime.client.Ping(ctx); err != nil {
		t.Skipf("Docker Engine unavailable: %v", err)
	}

	if _, err := runtime.Pull(ctx, "nginx:alpine"); err != nil {
		t.Fatalf("pull image: %v", err)
	}
	tag := "aether-test-runtime:" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	if err := runtime.Tag(ctx, "nginx:alpine", tag); err != nil {
		t.Fatalf("tag image: %v", err)
	}
	defer runtime.RemoveImage(context.Background(), tag)
	if port, err := runtime.ExposedPort(ctx, "nginx:alpine"); err != nil || port != 80 {
		t.Fatalf("exposed port = %d, error = %v", port, err)
	}

	buildDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(buildDir, "Dockerfile"), []byte("FROM alpine:3.20\nEXPOSE 8081\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	builtTag := "aether-test-build:" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	if _, err := runtime.Build(ctx, buildDir, filepath.Join(buildDir, "Dockerfile"), builtTag); err != nil {
		t.Fatalf("build image: %v", err)
	}
	defer runtime.RemoveImage(context.Background(), builtTag)
	if port, err := runtime.ExposedPort(ctx, builtTag); err != nil || port != 8081 {
		t.Fatalf("built exposed port = %d, error = %v", port, err)
	}

	networkName := "aether-test-network-" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	if err := runtime.EnsureNetwork(ctx, networkName, map[string]string{"aether.test": "true"}); err != nil {
		t.Fatalf("create network: %v", err)
	}
	defer runtime.RemoveNetwork(context.Background(), networkName)
	volumeName := "aether-test-volume-" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	if err := runtime.CreateVolume(ctx, volumeName, map[string]string{"aether.test": "true"}); err != nil {
		t.Fatalf("create volume: %v", err)
	}
	defer runtime.RemoveVolume(context.Background(), volumeName, true)

	containerName := "aether-test-runtime-" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	containerID, err := runtime.Run(ctx, RunSpec{
		Name:          containerName,
		Image:         "nginx:alpine",
		ContainerPort: 80,
		Network:       networkName,
		NetworkAlias:  "runtime-test",
		Labels:        map[string]string{"aether.test": "true"},
	})
	if err != nil {
		t.Fatalf("run container: %v", err)
	}
	defer runtime.Remove(context.Background(), containerID)

	if state, err := runtime.ContainerState(ctx, containerID); err != nil || state != "running" {
		t.Fatalf("state = %q, error = %v", state, err)
	}
	if hostPort, err := runtime.Port(ctx, containerID); err != nil || hostPort == "" {
		t.Fatalf("host port = %q, error = %v", hostPort, err)
	}
	containers, err := runtime.ListContainers(ctx)
	if err != nil {
		t.Fatalf("list containers: %v", err)
	}
	found := false
	for _, item := range containers {
		if item.ID == containerID {
			found = true
			if item.Labels["aether.test"] != "true" || !item.HasStats {
				t.Fatalf("container normalization = %+v", item)
			}
		}
	}
	if !found {
		t.Fatalf("container %s was not listed", containerID)
	}
	if _, err := runtime.Stats(ctx, containerID); err != nil {
		t.Fatalf("stats: %v", err)
	}
	stdout, stderr, err := runtime.Exec(ctx, containerID, nil, "sh", "-c", "printf exec-output")
	if err != nil || stdout != "exec-output" || stderr != "" {
		t.Fatalf("exec stdout=%q stderr=%q error=%v", stdout, stderr, err)
	}
	if _, err := runtime.LogTail(ctx, containerID, 10); err != nil {
		t.Fatalf("log tail: %v", err)
	}
	if err := runtime.Stop(ctx, containerID); err != nil {
		t.Fatalf("stop: %v", err)
	}
	var logs bytes.Buffer
	if err := runtime.FollowLogs(ctx, containerID, &logs); err != nil {
		t.Fatalf("follow logs: %v", err)
	}
	if state, err := runtime.ContainerState(ctx, containerID); err != nil || state != "exited" {
		t.Fatalf("stopped state = %q, error = %v", state, err)
	}
	if err := runtime.Start(ctx, containerID); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := runtime.Restart(ctx, containerID); err != nil {
		t.Fatalf("restart: %v", err)
	}
	if err := runtime.Stop(ctx, containerID); err != nil {
		t.Fatalf("stop before wait: %v", err)
	}
	if _, err := runtime.Wait(ctx, containerID); err != nil {
		t.Fatalf("wait: %v", err)
	}

	commandOutput, err := runtime.RunCommand(ctx, "", "nginx:alpine", "printf command-output", nil, true)
	if err != nil || commandOutput != "command-output" {
		t.Fatalf("run command output=%q error=%v", commandOutput, err)
	}
}
