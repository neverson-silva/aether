package bootstrap

import (
	"strings"
	"testing"

	"aether/internal/platform/config"
)

func TestStaticTraefikConfigIncludesACMEResolver(t *testing.T) {
	content := staticTraefikConfig(&config.Config{
		CertEmail:     "admin@example.com",
		ACMEDirectory: "https://acme-staging-v02.api.letsencrypt.org/directory",
	})

	for _, expected := range []string{
		"certResolver: letsencrypt",
		"certificatesResolvers:",
		"letsencrypt:",
		"email: \"admin@example.com\"",
		"caServer: \"https://acme-staging-v02.api.letsencrypt.org/directory\"",
		"storage: /etc/traefik/acme/acme.json",
		"entryPoint: web",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("Traefik config does not contain %q:\n%s", expected, content)
		}
	}
}

func TestStaticTraefikConfigKeepsResolverWithoutEmail(t *testing.T) {
	content := staticTraefikConfig(&config.Config{ACMEDirectory: "https://acme-v02.api.letsencrypt.org/directory"})

	for _, expected := range []string{
		"certificatesResolvers:\n  letsencrypt:\n    acme:\n",
		"email: \"test@localhost.com\"",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("Traefik config does not contain %q:\n%s", expected, content)
		}
	}
}
