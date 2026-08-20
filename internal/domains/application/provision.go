package application

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"aether/internal/domains/domain"
	"aether/internal/hostinfo"
)

type Provisioner struct {
	TraefikDir         string
	FreeDomainBase     string
	FreeDomainProvider string
	TraefikBin         string
}

var hostPattern = regexp.MustCompile(`^(?i)([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,}$`)

var wildcardDNSProviders = map[string]bool{
	"nip.io": true, "sslip.io": true, "traefik.me": true,
}

func (p *Provisioner) EffectiveBase() string {
	if base := strings.Trim(strings.ToLower(p.FreeDomainBase), "."); base != "" {
		return base
	}
	provider := strings.ToLower(strings.TrimSpace(p.FreeDomainProvider))
	if wildcardDNSProviders[provider] {
		return "apps." + hostinfo.PublicIPDashed() + "." + provider
	}
	return ""
}

func (p *Provisioner) IsPublicBase() bool {
	base := strings.ToLower(strings.TrimSpace(p.FreeDomainBase))
	if base == "" {
		return true
	}
	for _, suffix := range []string{".local", ".localhost", ".internal", ".lan", ".home.arpa"} {
		if strings.HasSuffix(base, suffix) {
			return false
		}
	}
	return true
}

func (p *Provisioner) ValidateHost(host string) error {
	host = strings.TrimSpace(host)
	if len(host) > 253 || !hostPattern.MatchString(host) {
		return domain.ErrValidation
	}
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasPrefix(lower, "127.") || lower == "0.0.0.0" ||
		strings.HasPrefix(lower, "169.254.") || strings.HasPrefix(lower, "10.") ||
		strings.HasPrefix(lower, "192.168.") || strings.HasPrefix(lower, "172.") {
		return domain.ErrValidation
	}
	return nil
}

func (p *Provisioner) GenerateFreeDomain(slug string, id uuid.UUID) string {
	return fmt.Sprintf("%s-%s.%s", slugify(slug), id.String()[:8], p.EffectiveBase())
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func serverKey(serverID string) string {
	if serverID == "" || serverID == "00000000-0000-0000-0000-000000000000" {
		return "local"
	}
	return "server-" + serverID[:8]
}

func (p *Provisioner) dynamicDir(serverID string) string {
	if serverKey(serverID) == "local" {
		return filepath.Join(p.TraefikDir, "dynamic")
	}
	return filepath.Join(p.TraefikDir, serverKey(serverID), "dynamic")
}

// WriteDomainConfig grava (idempotente) a config dinâmica do domínio para que o
// Traefik do server detecte a rota sem redeploy.
func (p *Provisioner) WriteDomainConfig(d *domain.Domain, alias string, httpsReady bool) error {
	dir := p.dynamicDir(d.ServerID.String())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	content := p.generateDynamicConfig(d, alias, httpsReady)
	return os.WriteFile(filepath.Join(dir, "domain-"+d.ID.String()+".yml"), []byte(content), 0o644)
}

func (p *Provisioner) Alias(serviceID uuid.UUID, serviceType string) string {
	prefix := "app-"
	if serviceType == ServiceTypeDB {
		prefix = "db-"
	}
	return prefix + serviceID.String()[:8]
}

func (p *Provisioner) RemoveDomainConfig(d *domain.Domain) error {
	return os.Remove(filepath.Join(p.dynamicDir(d.ServerID.String()), "domain-"+d.ID.String()+".yml"))
}

func (p *Provisioner) generateDynamicConfig(d *domain.Domain, alias string, httpsReady bool) string {
	var sb strings.Builder
	rule := "Host(`" + d.Host + "`)"
	if d.Path != "" && d.Path != "/" {
		rule += " && PathPrefix(`" + d.Path + "`)"
	}
	internal := d.InternalPath
	if internal == "" {
		internal = "/"
	}
	sb.WriteString("http:\n")
	if d.HTTPS && httpsReady {
		sb.WriteString("  middlewares:\n")
		sb.WriteString("    https-redirect-" + d.ID.String() + ":\n")
		sb.WriteString("      redirectScheme:\n        scheme: https\n        permanent: true\n")
		if d.StripPath && d.Path != "" && d.Path != "/" {
			sb.WriteString("    strip-" + d.ID.String() + ":\n")
			sb.WriteString("      stripPrefix:\n        prefixes:\n          - \"" + d.Path + "\"\n")
		}
		sb.WriteString("  routers:\n")
		sb.WriteString("    " + d.ID.String() + "-http:\n")
		sb.WriteString("      rule: \"" + rule + "\"\n")
		sb.WriteString("      entryPoints:\n      - web\n")
		sb.WriteString("      middlewares:\n      - https-redirect-" + d.ID.String() + "\n")
		sb.WriteString("      service: " + d.ID.String() + "\n")
		sb.WriteString("    " + d.ID.String() + ":\n")
		sb.WriteString("      rule: \"" + rule + "\"\n")
		sb.WriteString("      entryPoints:\n      - websecure\n")
		sb.WriteString("      service: " + d.ID.String() + "\n")
		if d.StripPath && d.Path != "" && d.Path != "/" {
			sb.WriteString("      middlewares:\n      - strip-" + d.ID.String() + "\n")
		}
		sb.WriteString("      tls:\n        certResolver: letsencrypt\n")
	} else {
		sb.WriteString("  routers:\n")
		sb.WriteString("    " + d.ID.String() + ":\n")
		sb.WriteString("      rule: \"" + rule + "\"\n")
		sb.WriteString("      entryPoints:\n      - web\n")
		sb.WriteString("      service: " + d.ID.String() + "\n")
	}
	sb.WriteString("  services:\n")
	sb.WriteString("    " + d.ID.String() + ":\n")
	sb.WriteString("      loadBalancer:\n")
	sb.WriteString("        servers:\n")
	sb.WriteString("          - url: \"http://" + alias + ":" + itoa(d.ContainerPort) + internal + "\"\n")
	return sb.String()
}

// VerifyCertificate tenta confirmar que o Traefik do server já emitiu e serve
// o certificado para o host, via HTTPS no entrypoint websecure.
func (p *Provisioner) VerifyCertificate(host string) bool {
	if p.TraefikBin == "" {
		return false
	}
	out, err := exec.Command(p.TraefikBin, "exec", "aether-traefik",
		"wget", "-q", "-O", "/dev/null", "--no-check-certificate",
		"https://localhost/", "--header=Host: "+host).CombinedOutput()
	return err == nil && strings.Contains(string(out), "200")
}

func itoa(n int) string {
	if n == 0 {
		return "80"
	}
	return fmt.Sprintf("%d", n)
}
