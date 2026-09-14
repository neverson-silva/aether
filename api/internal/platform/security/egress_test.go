package security

import (
	"testing"
	"time"
)

func TestValidateOutboundURLRejectsInternalDestinations(t *testing.T) {
	for _, raw := range []string{
		"http://127.0.0.1:8080",
		"http://10.0.0.1",
		"http://169.254.169.254/latest/meta-data",
		"http://host.docker.internal:2375",
		"file:///etc/passwd",
		"http://user:pass@example.com",
	} {
		if err := ValidateOutboundURL(raw); err == nil {
			t.Fatalf("unsafe URL accepted: %s", raw)
		}
	}
}

func TestValidateOutboundURLAllowsPublicHTTPS(t *testing.T) {
	if err := ValidateOutboundURL("https://example.com/oidc"); err != nil {
		t.Fatalf("public URL rejected: %v", err)
	}
}

func TestTestHTTPClientAllowsExplicitLoopback(t *testing.T) {
	if err := ValidateOutboundURLForTest("http://127.0.0.1:8080"); err != nil {
		t.Fatalf("test loopback URL rejected: %v", err)
	}
	client := NewTestHTTPClient(time.Second)
	if client == nil || client.Transport == nil {
		t.Fatal("test client is incomplete")
	}
}
