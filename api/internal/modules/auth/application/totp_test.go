package application

import (
	"strings"
	"testing"
	"time"
)

func TestTOTPCode(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	code, err := totpCode(secret, time.Unix(59, 0))
	if err != nil {
		t.Fatalf("code: %v", err)
	}
	if code != "287082" {
		t.Fatalf("esperava 287082, got %s", code)
	}
}

func TestTOTPSecretFormat(t *testing.T) {
	secret, err := generateTOTPSecret()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(secret) != 32 || strings.Contains(secret, "=") {
		t.Fatalf("secret inválida: %q", secret)
	}
}

func TestProvisioningURI(t *testing.T) {
	uri := provisioningURI("SECRET", "user@test.com", "aether")
	if !strings.HasPrefix(uri, "otpauth://totp/user@test.com") || !strings.Contains(uri, "issuer=aether") {
		t.Fatalf("uri inesperada: %s", uri)
	}
}
