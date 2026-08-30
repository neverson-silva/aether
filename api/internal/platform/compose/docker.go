package compose

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
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

type Docker struct {
	Binary string
	Host   string
}

func NewDocker(host string) *Docker {
	return &Docker{Binary: "docker", Host: host}
}

func (d *Docker) Execute(ctx context.Context, project Project, args ...string) (string, error) {
	if strings.TrimSpace(project.Directory) == "" || strings.TrimSpace(project.File) == "" {
		return "", fmt.Errorf("compose project is incomplete")
	}
	binary := d.Binary
	if binary == "" {
		binary = "docker"
	}
	commandArgs := []string{"compose", "--project-directory", project.Directory, "--project-name", project.Name}
	if project.EnvFile != "" {
		commandArgs = append(commandArgs, "--env-file", project.EnvFile)
	}
	commandArgs = append(commandArgs, "-f", project.File)
	commandArgs = append(commandArgs, args...)
	cmd := exec.CommandContext(ctx, binary, commandArgs...)
	cmd.Dir = project.Directory
	cmd.Env = os.Environ()
	if d.Host != "" {
		cmd.Env = append(cmd.Env, "DOCKER_HOST="+d.Host)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return "", fmt.Errorf("docker compose %s: %w", strings.Join(args, " "), err)
		}
		return string(output), fmt.Errorf("docker compose %s: %s: %w", strings.Join(args, " "), message, err)
	}
	return string(output), nil
}
