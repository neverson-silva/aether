package compose

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type Project struct {
	Directory string
	File      string
	EnvFile   string
	Name      string
}

type Executor interface {
	Execute(context.Context, Project, ...string) (string, error)
}

type StreamingExecutor interface {
	ExecuteWithLogs(context.Context, Project, func(string), ...string) (string, error)
}

type Docker struct {
	Binary string
	Host   string
}

func NewDocker(host string) *Docker {
	return &Docker{Binary: "docker", Host: host}
}

func (d *Docker) Execute(ctx context.Context, project Project, args ...string) (string, error) {
	return d.execute(ctx, project, nil, args...)
}

func (d *Docker) ExecuteWithLogs(ctx context.Context, project Project, sink func(string), args ...string) (string, error) {
	return d.execute(ctx, project, sink, args...)
}

func (d *Docker) execute(ctx context.Context, project Project, sink func(string), args ...string) (string, error) {
	if strings.TrimSpace(project.Directory) == "" || strings.TrimSpace(project.File) == "" {
		return "", fmt.Errorf("compose project is incomplete")
	}
	binary := d.Binary
	if binary == "" {
		binary = "docker"
	}
	composeFile, cleanup, err := d.prepareNetworks(ctx, project)
	if err != nil {
		return "", err
	}
	defer cleanup()
	commandArgs := []string{"compose", "--project-directory", project.Directory, "--project-name", project.Name}
	if project.EnvFile != "" {
		commandArgs = append(commandArgs, "--env-file", project.EnvFile)
	}
	commandArgs = append(commandArgs, "-f", composeFile)
	commandArgs = append(commandArgs, args...)
	cmd := exec.CommandContext(ctx, binary, commandArgs...)
	cmd.Dir = project.Directory
	cmd.Env = os.Environ()
	if d.Host != "" {
		cmd.Env = append(cmd.Env, "DOCKER_HOST="+d.Host)
	}
	var outputBuffer bytes.Buffer
	if sink == nil {
		cmd.Stdout = &outputBuffer
		cmd.Stderr = &outputBuffer
	} else {
		writer := &streamWriter{buffer: &outputBuffer, sink: sink}
		cmd.Stdout = writer
		cmd.Stderr = writer
	}
	err = cmd.Run()
	output := outputBuffer.String()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return "", fmt.Errorf("docker compose %s: %w", strings.Join(args, " "), err)
		}
		return string(output), fmt.Errorf("docker compose %s: %s: %w", strings.Join(args, " "), message, err)
	}
	return string(output), nil
}

func (d *Docker) prepareNetworks(ctx context.Context, project Project) (string, func(), error) {
	content, err := os.ReadFile(project.File)
	if err != nil {
		return "", func() {}, err
	}
	updated, changed, err := markExistingNetworks(string(content), func(name string) bool {
		binary := d.Binary
		if binary == "" {
			binary = "docker"
		}
		command := exec.CommandContext(ctx, binary, "network", "inspect", name)
		if d.Host != "" {
			command.Env = append(os.Environ(), "DOCKER_HOST="+d.Host)
		}
		return command.Run() == nil
	})
	if err != nil {
		return "", func() {}, err
	}
	if !changed {
		return project.File, func() {}, nil
	}
	overlay, err := os.CreateTemp(project.Directory, ".compose-network-*.yml")
	if err != nil {
		return "", func() {}, err
	}
	name := overlay.Name()
	if _, err := overlay.WriteString(updated); err != nil {
		_ = overlay.Close()
		_ = os.Remove(name)
		return "", func() {}, err
	}
	if err := overlay.Close(); err != nil {
		_ = os.Remove(name)
		return "", func() {}, err
	}
	return name, func() { _ = os.Remove(filepath.Clean(name)) }, nil
}

func markExistingNetworks(content string, exists func(string) bool) (string, bool, error) {
	var document map[string]any
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", false, err
	}
	rawNetworks, ok := document["networks"].(map[string]any)
	if !ok {
		return content, false, nil
	}
	changed := false
	for key, raw := range rawNetworks {
		config, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if external, ok := config["external"].(bool); ok && external {
			continue
		}
		name, ok := config["name"].(string)
		if !ok || strings.TrimSpace(name) == "" || !exists(name) {
			continue
		}
		config["external"] = true
		rawNetworks[key] = config
		changed = true
	}
	if !changed {
		return content, false, nil
	}
	updated, err := yaml.Marshal(document)
	if err != nil {
		return "", false, err
	}
	return string(updated), true, nil
}

type streamWriter struct {
	buffer *bytes.Buffer
	sink   func(string)
	mu     sync.Mutex
}

func (w *streamWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.sink != nil {
		line := strings.TrimSpace(string(data))
		if line != "" {
			w.sink(line)
		}
	}
	return w.buffer.Write(data)
}

var _ io.Writer = (*streamWriter)(nil)
