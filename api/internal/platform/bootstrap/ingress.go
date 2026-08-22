package bootstrap

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"aether/internal/platform/config"
)

func ensureIngress(ctx context.Context, cfg *config.Config) {
	if _, err := exec.CommandContext(ctx, "podman", "network", "exists", cfg.IngressNetwork).CombinedOutput(); err != nil {
		_ = exec.CommandContext(ctx, "podman", "network", "create", cfg.IngressNetwork).Run()
	}

	dir := filepath.Join(cfg.StateDir, "traefik")
	_ = os.MkdirAll(filepath.Join(dir, "dynamic"), 0o755)
	_ = os.MkdirAll(filepath.Join(dir, "acme"), 0o700)

	traefikYml := filepath.Join(dir, "traefik.yml")
	_ = os.WriteFile(traefikYml, []byte(staticTraefikConfig(cfg)), 0o644)

	out, err := exec.CommandContext(ctx, "podman", "ps", "-a", "--filter", "name=aether-traefik", "--format", "{{.Names}}").CombinedOutput()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		return
	}
	args := []string{
		"run", "-d", "--name", "aether-traefik",
		"--network", cfg.IngressNetwork,
		"--label", "io.aether.component=traefik",
		"-p", "80:80", "-p", "443:443",
		"-v", "aether-traefik:/etc/traefik",
		cfg.TraefikImage,
		"--configFile=/etc/traefik/traefik.yml",
	}
	_ = exec.CommandContext(ctx, "podman", args...).Run()
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
