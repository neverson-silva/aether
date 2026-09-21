package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"aether/internal/modules/settings/domain"
)

type ServerDomainSettings struct {
	WebDomain string `json:"web_domain"`
	APIDomain string `json:"api_domain"`
	HTTPS     bool   `json:"https"`
}

type ServerDomains struct {
	TraefikDir string
}

var serverDomainPattern = regexp.MustCompile(`^(?i)([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,}$`)

func (s *ServerDomains) Get() (*ServerDomainSettings, error) {
	settings := &ServerDomainSettings{HTTPS: true}
	raw, err := os.ReadFile(s.settingsPath())
	if errors.Is(err, os.ErrNotExist) {
		return settings, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *ServerDomains) PublicURL(ctx context.Context) string {
	_ = ctx
	settings, err := s.Get()
	if err != nil {
		return ""
	}
	host := settings.WebDomain
	if host == "" {
		host = settings.APIDomain
	}
	if host == "" {
		return ""
	}
	scheme := "http"
	if settings.HTTPS {
		scheme = "https"
	}
	return scheme + "://" + host
}

func (s *ServerDomains) Save(settings *ServerDomainSettings) (*ServerDomainSettings, error) {
	normalized := &ServerDomainSettings{
		WebDomain: strings.ToLower(strings.TrimSpace(settings.WebDomain)),
		APIDomain: strings.ToLower(strings.TrimSpace(settings.APIDomain)),
		HTTPS:     settings.HTTPS,
	}
	if normalized.WebDomain != "" && !serverDomainPattern.MatchString(normalized.WebDomain) {
		return nil, domain.ErrValidation
	}
	if normalized.APIDomain != "" && !serverDomainPattern.MatchString(normalized.APIDomain) {
		return nil, domain.ErrValidation
	}
	if normalized.WebDomain != "" && normalized.WebDomain == normalized.APIDomain {
		return nil, domain.ErrConflict
	}
	if err := os.MkdirAll(filepath.Join(s.TraefikDir, "dynamic"), 0o755); err != nil {
		return nil, err
	}
	if err := writeAtomic(s.settingsPath(), normalized); err != nil {
		return nil, err
	}
	if err := os.WriteFile(s.dynamicPath(), []byte(s.dynamicConfig(normalized)), 0o644); err != nil {
		return nil, err
	}
	return normalized, nil
}

func (s *ServerDomains) settingsPath() string {
	return filepath.Join(s.TraefikDir, "server-domains.json")
}

func (s *ServerDomains) dynamicPath() string {
	return filepath.Join(s.TraefikDir, "dynamic", "aether-platform-domains.yml")
}

func writeAtomic(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, raw, 0o600); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}

func (s *ServerDomains) dynamicConfig(settings *ServerDomainSettings) string {
	var b strings.Builder
	b.WriteString("http:\n")
	if settings.WebDomain == "" && settings.APIDomain == "" {
		b.WriteString("  routers: {}\n  services: {}\n")
		return b.String()
	}
	b.WriteString("  middlewares:\n")
	if settings.HTTPS {
		b.WriteString("    aether-platform-https-redirect:\n      redirectScheme:\n        scheme: https\n        permanent: true\n")
	}
	b.WriteString("  routers:\n")
	if settings.WebDomain != "" {
		s.writeTarget(&b, "web", settings.WebDomain, settings.HTTPS)
	}
	if settings.APIDomain != "" {
		s.writeTarget(&b, "api", settings.APIDomain, settings.HTTPS)
	}
	b.WriteString("  services:\n")
	if settings.WebDomain != "" {
		b.WriteString("    aether-platform-web:\n      loadBalancer:\n        servers:\n          - url: \"http://aether-web:4000/\"\n")
	}
	if settings.APIDomain != "" {
		b.WriteString("    aether-platform-api:\n      loadBalancer:\n        servers:\n          - url: \"http://aether-api:8080/\"\n")
	}
	return b.String()
}

func (s *ServerDomains) writeTarget(b *strings.Builder, name, host string, https bool) {
	rule := fmt.Sprintf("Host(`%s`)", host)
	if https {
		b.WriteString("    aether-platform-" + name + "-http:\n")
		b.WriteString("      rule: \"" + rule + "\"\n      entryPoints:\n        - web\n      middlewares:\n        - aether-platform-https-redirect\n      service: aether-platform-" + name + "\n")
		b.WriteString("    aether-platform-" + name + "-https:\n")
		b.WriteString("      rule: \"" + rule + "\"\n      entryPoints:\n        - websecure\n      service: aether-platform-" + name + "\n      tls:\n        certResolver: letsencrypt\n")
		return
	}
	b.WriteString("    aether-platform-" + name + ":\n")
	b.WriteString("      rule: \"" + rule + "\"\n      entryPoints:\n        - web\n      service: aether-platform-" + name + "\n")
}
