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
		"aether-service-unavailable@file",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("Traefik config does not contain %q:\n%s", expected, content)
		}
	}
}

func TestGlobalErrorConfigUsesWebGateway(t *testing.T) {
	content := globalErrorConfig()

	for _, expected := range []string{
		"aether-service-unavailable:",
		"status:\n        - \"502-504\"",
		"service: aether-error-page",
		"url: \"http://aether-web:4000\"",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("Traefik error configuration does not contain %q:\n%s", expected, content)
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
