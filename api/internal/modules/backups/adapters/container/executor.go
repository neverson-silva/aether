package container

import (
	"context"
	"errors"
	"io"
)

type Executor interface {
	Exec(ctx context.Context, containerID string, env []string, args ...string) (stdout string, stderr string, err error)
	ExecStream(ctx context.Context, containerID string, env []string, stdout io.Writer, stderr io.Writer, args ...string) error
	ExecIn(ctx context.Context, containerID string, env []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, args ...string) error
	ExecAs(ctx context.Context, containerID string, user string, env []string, stdin io.Reader, args ...string) (stdout string, stderr string, err error)
}

type Unavailable struct{}

func (Unavailable) Exec(context.Context, string, []string, ...string) (string, string, error) {
	return "", "", errors.New("container executor is not configured")
}

func (Unavailable) ExecStream(context.Context, string, []string, io.Writer, io.Writer, ...string) error {
	return errors.New("container executor is not configured")
}

func (Unavailable) ExecIn(context.Context, string, []string, io.Reader, io.Writer, io.Writer, ...string) error {
	return errors.New("container executor is not configured")
}

func (Unavailable) ExecAs(context.Context, string, string, []string, io.Reader, ...string) (string, string, error) {
	return "", "", errors.New("container executor is not configured")
}
