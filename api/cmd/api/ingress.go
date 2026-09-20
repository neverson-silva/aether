package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	if err := networkRuntime.EnsureNetwork(ctx, cfg.PublishedNetwork, map[string]string{"io.aether.component": "workload-published"}); err != nil {
		return fmt.Errorf("ensure published network: %w", err)
	}

	dir := filepath.Join(cfg.StateDir, "traefik")
	if err := os.MkdirAll(filepath.Join(dir, "dynamic"), 0o755); err != nil {
		return fmt.Errorf("prepare ingress dynamic directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "acme"), 0o700); err != nil {
		return fmt.Errorf("prepare ingress certificate directory: %w", err)
	}
	acmeStorage := filepath.Join(dir, "acme", "acme.json")
	if _, err := os.Stat(acmeStorage); os.IsNotExist(err) {
		if err := os.WriteFile(acmeStorage, []byte("{}\n"), 0o600); err != nil {
			return fmt.Errorf("prepare ingress certificate storage: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("inspect ingress certificate storage: %w", err)
	} else if err := os.Chmod(acmeStorage, 0o600); err != nil {
		return fmt.Errorf("secure ingress certificate storage: %w", err)
	}

	traefikYml := filepath.Join(dir, "traefik.yml")
	if err := os.WriteFile(traefikYml, []byte(staticTraefikConfig(cfg)), 0o644); err != nil {
		return fmt.Errorf("write ingress configuration: %w", err)
	}

	for _, item := range mustListContainers(ctx, runtime) {
		if item.Name != "aether-traefik" {
			continue
		}
		if item.State == "running" {
			if _, err := runtime.Port(ctx, item.ID); err == nil {
				return nil
			}
		}
		if err := runtime.Remove(ctx, item.ID); err != nil {
			return fmt.Errorf("remove stale ingress: %w", err)
		}
		break
	}
	if _, err := runtime.Pull(ctx, cfg.TraefikImage); err != nil {
		return fmt.Errorf("pull ingress image %q: %w", cfg.TraefikImage, err)
	}
	if _, err := runtime.Run(ctx, worker.RunSpec{
		Name: "aether-traefik", Image: cfg.TraefikImage, Network: cfg.IngressNetwork,
		AdditionalNetworks: []string{cfg.PublishedNetwork},
		NetworkAlias:       "traefik",
		Labels:             map[string]string{"io.aether.component": "traefik", "io.aether.managed": "true"},
		Command:            []string{"--configFile=/etc/traefik/traefik.yml"},
		Mounts:             []worker.MountSpec{{Source: dir, Target: "/etc/traefik"}},
		Ports:              []worker.PortSpec{{HostPort: 80, ContainerPort: 80, HostIP: "0.0.0.0"}, {HostPort: 443, ContainerPort: 443, HostIP: "0.0.0.0"}},
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
	sb.WriteString("  websecure:\n    address: \":443\"\n    http:\n      tls:\n        certResolver: letsencrypt\n")
	sb.WriteString("providers:\n  file:\n    directory: /etc/traefik/dynamic\n    watch: true\n")
	email := cfg.CertEmail
	if email == "" {
		email = "test@localhost.com"
	}
	sb.WriteString("certificatesResolvers:\n  letsencrypt:\n    acme:\n      email: " + strconv.Quote(email) + "\n")
	if cfg.ACMEDirectory != "" {
		sb.WriteString("      caServer: " + strconv.Quote(cfg.ACMEDirectory) + "\n")
	}
	sb.WriteString("      storage: /etc/traefik/acme/acme.json\n      httpChallenge:\n        entryPoint: web\n")
	return sb.String()
}
