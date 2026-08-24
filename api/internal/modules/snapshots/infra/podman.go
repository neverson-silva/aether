package infra

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type PodmanExecutor struct {
	OutputDir string
	Image     string
}

func (e PodmanExecutor) Delete(_ context.Context, path string) error {
	if filepath.Dir(path) != e.OutputDir {
		return fmt.Errorf("invalid snapshot path")
	}
	return os.Remove(path)
}

func (e PodmanExecutor) Create(ctx context.Context, volume, name string) (string, int64, error) {
	if !safeVolume(volume) || !safeName(name) {
		return "", 0, fmt.Errorf("invalid snapshot source")
	}
	if e.Image == "" {
		e.Image = "docker.io/library/alpine:3.20"
	}
	if e.OutputDir == "" {
		return "", 0, fmt.Errorf("snapshot output directory is required")
	}
	if err := os.MkdirAll(e.OutputDir, 0o750); err != nil {
		return "", 0, err
	}
	filename := fmt.Sprintf("%s-%d.tar.gz", name, time.Now().UnixNano())
	path := filepath.Join(e.OutputDir, filename)
	containerName := "aether-snapshot-" + strings.ToLower(name[:min(len(name), 12)])
	args := []string{"run", "--rm", "--name", containerName, "-v", volume + ":/source:ro", "-v", e.OutputDir + ":/out", e.Image, "tar", "-czf", "/out/" + filename, "-C", "/source", "."}
	if output, err := exec.CommandContext(ctx, "podman", args...).CombinedOutput(); err != nil {
		return "", 0, fmt.Errorf("snapshot export: %s: %w", strings.TrimSpace(string(output)), err)
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
