package infra

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"aether/internal/platform/worker"
)

type DockerExecutor struct {
	OutputDir string
	Image     string
	Runtime   worker.Runtime
}

func (e DockerExecutor) Delete(_ context.Context, path string) error {
	if filepath.Dir(path) != e.OutputDir {
		return fmt.Errorf("invalid snapshot path")
	}
	return os.Remove(path)
}

func (e DockerExecutor) Create(ctx context.Context, volume, name string) (string, int64, error) {
	if !safeVolume(volume) || !safeName(name) {
		return "", 0, fmt.Errorf("invalid snapshot source")
	}
	if e.Image == "" {
		e.Image = "docker.io/library/alpine:3.20"
	}
	if e.OutputDir == "" || e.Runtime == nil {
		return "", 0, fmt.Errorf("snapshot runtime and output directory are required")
	}
	waiter, ok := e.Runtime.(worker.WaitRuntime)
	if !ok {
		return "", 0, fmt.Errorf("snapshot wait runtime is unavailable")
	}
	if err := os.MkdirAll(e.OutputDir, 0o750); err != nil {
		return "", 0, err
	}
	filename := fmt.Sprintf("%s-%d.tar.gz", name, time.Now().UnixNano())
	path := filepath.Join(e.OutputDir, filename)
	containerName := "aether-snapshot-" + strings.ToLower(name[:min(len(name), 12)])
	containerID, err := e.Runtime.Run(ctx, worker.RunSpec{
		Name:    containerName,
		Image:   e.Image,
		Command: []string{"tar", "-czf", "/out/" + filename, "-C", "/source", "."},
		Mounts: []worker.MountSpec{
			{Source: volume, Target: "/source", ReadOnly: true},
			{Source: e.OutputDir, Target: "/out"},
		},
	})
	if err != nil {
		return "", 0, fmt.Errorf("snapshot export: %w", err)
	}
	status, err := waiter.Wait(ctx, containerID)
	_ = e.Runtime.Remove(context.Background(), containerID)
	if err != nil {
		return "", 0, fmt.Errorf("snapshot export: %w", err)
	}
	if status != 0 {
		return "", 0, fmt.Errorf("snapshot export exited with code %d", status)
	}
	stat, err := os.Stat(path)
	if err != nil {
		return "", 0, err
	}
	return path, stat.Size(), nil
}

var safeVolumePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)
var safeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,63}$`)

func safeVolume(value string) bool { return safeVolumePattern.MatchString(value) }
func safeName(value string) bool   { return safeNamePattern.MatchString(value) }
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
