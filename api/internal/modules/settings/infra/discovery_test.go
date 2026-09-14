package infra

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aether/internal/platform/security"
)

func TestDiscover(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"issuer":"` + srv.URL + `","authorization_endpoint":"https://issuer/authorize","token_endpoint":"https://issuer/token","userinfo_endpoint":"https://issuer/userinfo","jwks_uri":"https://issuer/keys"}`))
	}))
	defer srv.Close()
	d := NewOIDCDiscoverer("http://127.0.0.1:8080")
	d.Client = security.NewTestHTTPClient(5 * time.Second)
	d.AllowLoopback = true
	disc, err := d.Discover(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if disc.AuthorizationEndpoint != "https://issuer/authorize" {
		t.Fatalf("endpoint: %s", disc.AuthorizationEndpoint)
	}
}

func TestAuthURL(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"issuer":"` + srv.URL + `","authorization_endpoint":"https://example.com/authorize","token_endpoint":"https://example.com/token","userinfo_endpoint":"https://example.com/userinfo","jwks_uri":"https://example.com/keys"}`))
	}))
	defer srv.Close()
	d := NewOIDCDiscoverer("https://aether.example")
	d.Client = security.NewTestHTTPClient(5 * time.Second)
	d.AllowLoopback = true
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
