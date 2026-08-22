package infra

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiscover(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"issuer":"https://issuer","authorization_endpoint":"https://issuer/authorize","token_endpoint":"https://issuer/token","userinfo_endpoint":"https://issuer/userinfo"}`))
	}))
	defer srv.Close()
	d := NewOIDCDiscoverer("http://127.0.0.1:8080")
	disc, err := d.Discover(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if disc.AuthorizationEndpoint != "https://issuer/authorize" {
		t.Fatalf("endpoint: %s", disc.AuthorizationEndpoint)
	}
}

func TestAuthURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"authorization_endpoint":"https://issuer/authorize","token_endpoint":"https://issuer/token","userinfo_endpoint":"https://issuer/userinfo"}`))
	}))
	defer srv.Close()
	d := NewOIDCDiscoverer("https://aether.example")
	url, err := d.AuthURL(context.Background(), srv.URL, "client123", "openid email", "oidc-1")
	if err != nil {
		t.Fatalf("auth url: %v", err)
	}
	if !strings.Contains(url, "client_id=client123") || !strings.Contains(url, "redirect_uri=") {
		t.Fatalf("url: %s", url)
	}
	if !strings.Contains(url, "sso%2Foidc-1%2Fcallback") {
		t.Fatalf("callback: %s", url)
	}
}

func TestCallbackURLDefault(t *testing.T) {
	d := NewOIDCDiscoverer("")
	if got := d.CallbackURL("oidc-1"); !strings.Contains(got, "127.0.0.1:8080") {
		t.Fatalf("callback: %s", got)
	}
}
