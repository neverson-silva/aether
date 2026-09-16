package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aether/internal/platform/config"
	"aether/internal/platform/worker"
)

const ingressConfigVersion = "2"

func ensureIngressWithRetry(ctx context.Context, cfg *config.Config, runtime worker.Runtime) error {
	var lastErr error
	for attempt := 0; attempt < 30; attempt++ {
		if err := ensureIngress(ctx, cfg, runtime); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if attempt == 29 {
			break
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return lastErr
}

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

	traefikYml := filepath.Join(dir, "traefik.yml")
	if err := os.WriteFile(traefikYml, []byte(staticTraefikConfig(cfg)), 0o644); err != nil {
		return fmt.Errorf("write ingress configuration: %w", err)
	}
	mountSource := cfg.TraefikMountSource
	mountTarget := "/etc/traefik"
	configFile := "/etc/traefik/traefik.yml"
	if mountSource != "" {
		mountTarget = "/var/lib/aether"
		configFile = "/var/lib/aether/traefik/traefik.yml"
	}

	for _, item := range mustListContainers(ctx, runtime) {
		if item.Name != "aether-traefik" {
			continue
		}
		if item.State == "running" && item.Labels["io.aether.ingress-config"] == ingressConfigVersion {
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
		Labels:             map[string]string{"io.aether.component": "traefik", "io.aether.managed": "true", "io.aether.ingress-config": ingressConfigVersion},
		Command:            []string{"--configFile=" + configFile},
		Mounts:             []worker.MountSpec{{Source: firstNonEmpty(mountSource, dir), Target: mountTarget}},
		Ports:              []worker.PortSpec{{HostPort: 80, ContainerPort: 80, HostIP: "0.0.0.0"}, {HostPort: 443, ContainerPort: 443, HostIP: "0.0.0.0"}},
	}); err != nil {
		return fmt.Errorf("start ingress: %w", err)
	}
	return nil
}

func firstNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func mustListContainers(ctx context.Context, runtime worker.Runtime) []worker.ContainerInfo {
	items, err := runtime.ListContainers(ctx)
	if err != nil {
		return nil
	}
	return items
}

func staticTraefikConfig(cfg *config.Config) string {
	traefikRoot := "/etc/traefik"
	if cfg.TraefikMountSource != "" {
		traefikRoot = "/var/lib/aether/traefik"
	}
	var sb strings.Builder
	sb.WriteString("log:\n  level: INFO\n")
	sb.WriteString("entryPoints:\n")
	sb.WriteString("  web:\n    address: \":80\"\n")
	sb.WriteString("  websecure:\n    address: \":443\"\n")
	sb.WriteString("providers:\n  file:\n    directory: " + traefikRoot + "/dynamic\n    watch: true\n")
	if cfg.CertEmail != "" {
		sb.WriteString("certificatesResolvers:\n  letsencrypt:\n    acme:\n      email: " + cfg.CertEmail + "\n      storage: " + traefikRoot + "/acme/acme.json\n      httpChallenge:\n        entryPoint: web\n")
	}
	return sb.String()
}
