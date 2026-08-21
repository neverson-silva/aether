package container

import (
	"context"
	"io"
	"os/exec"
)

type Executor interface {
	Exec(ctx context.Context, containerID string, env []string, args ...string) (stdout string, stderr string, err error)
	ExecStream(ctx context.Context, containerID string, env []string, stdout io.Writer, stderr io.Writer, args ...string) error
	ExecIn(ctx context.Context, containerID string, env []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, args ...string) error
	ExecAs(ctx context.Context, containerID string, user string, env []string, stdin io.Reader, args ...string) (stdout string, stderr string, err error)
}

type Podman struct{}

func NewPodman() *Podman {
	return &Podman{}
}

func (p *Podman) Exec(ctx context.Context, containerID string, env []string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, "podman", p.args(containerID, env, args)...)
	outBuf := &stringWriter{}
	errBuf := &stringWriter{}
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
	err := cmd.Run()
	return outBuf.s, errBuf.s, err
}

func (p *Podman) ExecStream(ctx context.Context, containerID string, env []string, stdout io.Writer, stderr io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, "podman", p.args(containerID, env, args)...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func (p *Podman) ExecIn(ctx context.Context, containerID string, env []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, "podman", p.args(containerID, env, args)...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func (p *Podman) ExecAs(ctx context.Context, containerID string, user string, env []string, stdin io.Reader, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, "podman", p.argsAs(containerID, user, env, args)...)
	outBuf := &stringWriter{}
	errBuf := &stringWriter{}
	cmd.Stdin = stdin
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
	err := cmd.Run()
	return outBuf.s, errBuf.s, err
}

func (p *Podman) args(containerID string, env []string, args []string) []string {
	return p.argsAs(containerID, "", env, args)
}

func (p *Podman) argsAs(containerID string, user string, env []string, args []string) []string {
	out := []string{"exec"}
	if user != "" {
		out = append(out, "--user", user)
	}
	for _, e := range env {
		out = append(out, "-e", e)
	}
	out = append(out, containerID)
	return append(out, args...)
}

type stringWriter struct {
	s string
}

func (w *stringWriter) Write(b []byte) (int, error) {
	w.s += string(b)
	return len(b), nil
}