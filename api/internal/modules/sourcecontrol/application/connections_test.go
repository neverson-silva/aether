package application

import (
	"context"
	"testing"
)

func TestResolvePublicURLPrefersConfiguredFrontendDomain(t *testing.T) {
	connections := &Connections{
		PublicURL: "http://203.0.113.10:8080",
		PublicURLResolver: func(context.Context) string {
			return "https://console.example.com/"
		},
	}

	if got := connections.ResolvePublicURL(context.Background()); got != "https://console.example.com" {
		t.Fatalf("public URL = %q, want frontend domain", got)
	}
}

func TestResolvePublicURLFallsBackToConfiguredURL(t *testing.T) {
	connections := &Connections{
		PublicURL: " http://203.0.113.10:8080/ ",
		PublicURLResolver: func(context.Context) string {
			return " "
		},
	}

	if got := connections.ResolvePublicURL(context.Background()); got != "http://203.0.113.10:8080" {
		t.Fatalf("public URL = %q, want configured fallback", got)
	}
}
