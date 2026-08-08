package runtime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type UnitManager struct {
	User string
	Dir  string
}

func (u *UnitManager) WriteUnit(name string, spec RunSpec, networkName string) (string, error) {
	if err := os.MkdirAll(u.Dir, 0o750); err != nil {
		return "", err
	}
	path := filepath.Join(u.Dir, name+".container")
	var b strings.Builder
	b.WriteString("[Unit]\n")
	b.WriteString("Description=Aether app unit " + name + "\n")
	b.WriteString("\n[Container]\n")
	b.WriteString("Image=" + spec.Image + "\n")
	if len(spec.Cmd) > 0 {
		b.WriteString("Exec=" + strings.Join(spec.Cmd, " ") + "\n")
	}
	for _, e := range spec.Env {
		b.WriteString("Environment=" + e + "\n")
	}
	for _, p := range spec.Ports {
		if p.HostPort != "" {
			b.WriteString("PublishPort=127.0.0.1:" + p.HostPort + ":" + p.ContainerPort + "\n")
		} else {
			b.WriteString("PublishPort=127.0.0.1::" + p.ContainerPort + "\n")
		}
	}
	if networkName != "" {
		b.WriteString("Network=" + networkName + "\n")
	}
	for _, v := range spec.Volumes {
		b.WriteString("Volume=" + v.Source + ":" + v.Target + "\n")
	}
	b.WriteString("\n[Service]\n")
	if spec.MemMB > 0 {
		b.WriteString(fmt.Sprintf("MemoryMax=%dm\n", spec.MemMB))
	}
	if spec.CPUs != "" {
		b.WriteString("CPUWeight=100\n")
	}
	if spec.PidsLimit > 0 {
		b.WriteString(fmt.Sprintf("PidsMax=%d\n", spec.PidsLimit))
	}
	switch spec.Restart {
	case "always":
		b.WriteString("Restart=always\n")
	case "on-failure":
		b.WriteString("Restart=on-failure\n")
	default:
		b.WriteString("Restart=no\n")
	}
	b.WriteString("\n[Install]\nWantedBy=default.target\n")
	return path, os.WriteFile(path, []byte(b.String()), 0o640)
}

func (u *UnitManager) machineArgs() []string {
	return []string{"--user", "--machine", u.User + "@.host"}
}

func (u *UnitManager) daemonReload(ctx context.Context) error {
	return exec.CommandContext(ctx, "systemctl", append(u.machineArgs(), "daemon-reload")...).Run()
}

func (u *UnitManager) Start(ctx context.Context, service string) error {
	if err := u.daemonReload(ctx); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "systemctl", append(u.machineArgs(), "start", service)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl start %s: %s", service, strings.TrimSpace(string(out)))
	}
	return nil
}

func (u *UnitManager) Stop(ctx context.Context, service string) error {
	cmd := exec.CommandContext(ctx, "systemctl", append(u.machineArgs(), "stop", service)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl stop %s: %s", service, strings.TrimSpace(string(out)))
	}
	return nil
}

func (u *UnitManager) Enable(ctx context.Context, service string) error {
	cmd := exec.CommandContext(ctx, "systemctl", append(u.machineArgs(), "enable", "--now", service)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl enable %s: %s", service, strings.TrimSpace(string(out)))
	}
	return nil
}

func (u *UnitManager) Disable(ctx context.Context, service string) error {
	cmd := exec.CommandContext(ctx, "systemctl", append(u.machineArgs(), "disable", "--now", service)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl disable %s: %s", service, strings.TrimSpace(string(out)))
	}
	return nil
}

func (u *UnitManager) Status(ctx context.Context, service string) (string, error) {
	out, err := exec.CommandContext(ctx, "systemctl", append(u.machineArgs(), "is-active", service)...).Output()
	if err != nil {
		return "", fmt.Errorf("systemctl is-active %s: %s", service, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}
