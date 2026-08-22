package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"
)

func TestVerifyWebhook(t *testing.T) {
	provider := &Provider{WebhookKey: []byte("secret")}
	body := []byte(`{"ref":"refs/heads/main"}`)
	mac := hmac.New(sha256.New, provider.WebhookKey)
	_, _ = mac.Write(body)
	headers := http.Header{"X-Hub-Signature-256": []string{"sha256=" + hex.EncodeToString(mac.Sum(nil))}}
	if err := provider.VerifyWebhook(headers, body); err != nil {
		t.Fatalf("verify webhook: %v", err)
	}
	headers.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString([]byte("invalid")))
	if err := provider.VerifyWebhook(headers, body); err == nil {
		t.Fatal("invalid signature was accepted")
	}
}

func TestParsePushWebhook(t *testing.T) {
	provider := &Provider{}
	headers := http.Header{}
	headers.Set("X-GitHub-Event", "push")
	headers.Set("X-GitHub-Delivery", "delivery-1")
	body := []byte(`{"ref":"refs/heads/main","before":"before","after":"after","repository":{"id":42,"name":"web","full_name":"acme/web","default_branch":"main","owner":{"login":"acme"}},"installation":{"id":7},"head_commit":{"id":"after","message":"update","author":{"name":"Aether"}}}`)
	event, err := provider.ParsePushWebhook(headers, body)
	if err != nil {
		t.Fatalf("parse push: %v", err)
	}
	if event.Branch != "main" || event.Repository.ID != "42" || event.InstallationID != "7" || event.AfterSHA != "after" {
		t.Fatalf("unexpected event: %+v", event)
	}
}
