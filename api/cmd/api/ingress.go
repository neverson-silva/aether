package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aether/internal/platform/config"
	"aether/internal/platform/worker"
)

func ensureIngress(ctx context.Context, cfg *config.Config, runtime worker.Runtime) error {
	if runtime == nil {
		return worker.ErrRuntimeUnavailable
	}
	networkRuntime, ok := runtime.(worker.NetworkRuntime)
	if !ok {
		return fmt.Errorf("Docker network runtime is unavailable")
	}
	if err := networkRuntime.EnsureNetwork(ctx, cfg.IngressNetwork, map[string]string{"io.aether.component": "ingress"}); err != nil {
		return fmt.Errorf("ensure ingress network: %w", err)
	}

	dir := filepath.Join(cfg.StateDir, "traefik")
	if err := os.MkdirAll(filepath.Join(dir, "dynamic"), 0o755); err != nil {
		return fmt.Errorf("prepare ingress dynamic directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "acme"), 0o700); err != nil {
		return fmt.Errorf("prepare ingress certificate directory: %w", err)
	}

	traefikYml := filepath.Join(dir, "traefik.yml")
	if err := os.WriteFile(traefikYml, []byte(staticTraefikConfig(cfg)), 0o644); err != nil {
		return fmt.Errorf("write ingress configuration: %w", err)
	}

	for _, item := range mustListContainers(ctx, runtime) {
		if item.Name == "aether-traefik" {
			return nil
		}
	}
	if _, err := runtime.Pull(ctx, cfg.TraefikImage); err != nil {
		return fmt.Errorf("pull ingress image %q: %w", cfg.TraefikImage, err)
	}
	if _, err := runtime.Run(ctx, worker.RunSpec{
		Name: "aether-traefik", Image: cfg.TraefikImage, Network: cfg.IngressNetwork,
		NetworkAlias: "traefik",
		Labels:       map[string]string{"io.aether.component": "traefik", "io.aether.managed": "true"},
		Command:      []string{"--configFile=/etc/traefik/traefik.yml"},
		Mounts:       []worker.MountSpec{{Source: dir, Target: "/etc/traefik"}},
		Ports:        []worker.PortSpec{{HostPort: 80, ContainerPort: 80}, {HostPort: 443, ContainerPort: 443}},
	}); err != nil {
		return fmt.Errorf("start ingress: %w", err)
	}
	return nil
}

func mustListContainers(ctx context.Context, runtime worker.Runtime) []worker.ContainerInfo {
	items, err := runtime.ListContainers(ctx)
	if err != nil {
		return nil
	}
	return items
}

func staticTraefikConfig(cfg *config.Config) string {
	var sb strings.Builder
	sb.WriteString("log:\n  level: INFO\n")
	sb.WriteString("entryPoints:\n")
	sb.WriteString("  web:\n    address: \":80\"\n")
	sb.WriteString("  websecure:\n    address: \":443\"\n")
	sb.WriteString("providers:\n  file:\n    directory: /etc/traefik/dynamic\n    watch: true\n")
	if cfg.CertEmail != "" {
		sb.WriteString("certificatesResolvers:\n  letsencrypt:\n    acme:\n      email: " + cfg.CertEmail + "\n      storage: /etc/traefik/acme/acme.json\n      httpChallenge:\n        entryPoint: web\n")
	}
	return sb.String()
}
