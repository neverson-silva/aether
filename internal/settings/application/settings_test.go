package application

import (
	"context"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"

	"aether/internal/settings/domain"
	"aether/internal/settings/infra"
)

type env struct {
	ctx   context.Context
	svc   *Settings
	orgID uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	cipher, err := infra.NewPasswordCipher([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	e := &env{ctx: context.Background(), svc: &Settings{Store: store, Passwords: cipher}, orgID: uuid.New()}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, e.orgID, "Set Org", "set-org"); err != nil {
		t.Fatalf("criar org: %v", err)
	}
	return e
}

func TestBranding(t *testing.T) {
	e := newEnv(t)
	b, err := e.svc.GetBranding(e.ctx, e.orgID)
	if err != nil {
		t.Fatalf("get default: %v", err)
	}
	if b.Name != "" {
		t.Fatalf("default deveria ser vazio")
	}
	saved, err := e.svc.SaveBranding(e.ctx, e.orgID, &domain.Branding{Name: "Acme", PrimaryColor: "#ff0000", DarkMode: true})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if saved.Name != "Acme" || !saved.DarkMode {
		t.Fatalf("branding salvo divergente: %+v", saved)
	}
	got, _ := e.svc.GetBranding(e.ctx, e.orgID)
	if got.Name != "Acme" {
		t.Fatalf("branding relido divergente")
	}
}

func TestS3Destinations(t *testing.T) {
	e := newEnv(t)
	dest, err := e.svc.CreateS3(e.ctx, e.orgID, &domain.S3Destination{
		Name: "backups", Type: domain.TypeCustomS3, Endpoint: "s3.amazonaws.com",
		Bucket: "bucket", AccessKeyEnc: "AKIA123", SecretKeyEnc: "secret456",
	})
	if err != nil {
		t.Fatalf("create s3: %v", err)
	}
	if dest.Region != "us-east-1" {
		t.Fatalf("region default inesperada")
	}
	if dest.AccessKeyEnc == "AKIA123" || dest.AccessKeyEnc == "" {
		t.Fatalf("access key deveria estar encriptada")
	}
	if _, err := e.svc.CreateS3(e.ctx, e.orgID, &domain.S3Destination{
		Name: "", Type: domain.TypeCustomS3, Endpoint: "x", Bucket: "y",
		AccessKeyEnc: "a", SecretKeyEnc: "b",
	}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("name vazio deveria falhar: %v", err)
	}
	list, err := e.svc.ListS3(e.ctx, e.orgID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}
	if err := e.svc.DeleteS3(e.ctx, dest.ID, e.orgID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestEndpointResolution(t *testing.T) {
	cases := []struct {
		typ      domain.DestinationType
		region   string
		account  string
		userEp   string
		expected string
	}{
		{domain.TypeAWS, "sa-east-1", "", "", "https://s3.sa-east-1.amazonaws.com"},
		{domain.TypeAWS, "", "", "", "https://s3.us-east-1.amazonaws.com"},
		{domain.TypeCloudflareR2, "", "abc123", "", "https://abc123.r2.cloudflarestorage.com"},
		{domain.TypeMinIO, "", "", "http://localhost:9000", "http://localhost:9000"},
		{domain.TypeCustomS3, "", "", "https://storage.example.com", "https://storage.example.com"},
		{domain.TypeGoogleDrive, "", "", "", ""},
	}
	for _, c := range cases {
		got := ResolveEndpoint(c.typ, c.region, c.account, c.userEp)
		if got != c.expected {
			t.Fatalf("%s: got %q want %q", c.typ, got, c.expected)
		}
	}
}

func TestGoogleDriveDestination(t *testing.T) {
	e := newEnv(t)
	dest, err := e.svc.CreateS3(e.ctx, e.orgID, &domain.S3Destination{
		Name: "drive", Type: domain.TypeGoogleDrive, Bucket: "backups",
	})
	if err != nil {
		t.Fatalf("create drive: %v", err)
	}
	if dest.Endpoint != "" || dest.AccessKeyEnc != "" {
		t.Fatalf("drive não deve ter endpoint/creds: %+v", dest)
	}
	if dest.OAuthStatus != domain.OAuthNone {
		t.Fatalf("drive status inesperado: %s", dest.OAuthStatus)
	}
	got, err := e.svc.GetS3(e.ctx, dest.ID, e.orgID)
	if err != nil || got.Type != domain.TypeGoogleDrive {
		t.Fatalf("get: %v %+v", err, got)
	}
}

func TestGoogleConnectRequiresConfig(t *testing.T) {
	e := newEnv(t)
	req := httptest.NewRequest("GET", "http://localhost:5173/api/v1/s3-destinations/test/google/connect", nil)
	dest, err := e.svc.CreateS3(e.ctx, e.orgID, &domain.S3Destination{Name: "drive", Type: domain.TypeGoogleDrive})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := e.svc.GoogleConnect(e.ctx, req, e.orgID, dest.ID); !errors.Is(err, ErrGoogleNotConfigured) {
		t.Fatalf("sem config deveria falhar: %v", err)
	}
	configured, err := e.svc.CreateS3(e.ctx, e.orgID, &domain.S3Destination{
		Name: "drive2", Type: domain.TypeGoogleDrive,
		GoogleClientID: "client", GoogleClientSecretEnc: "secret",
	})
	if err != nil {
		t.Fatalf("create configurado: %v", err)
	}
	if configured.GoogleClientSecretEnc == "secret" || configured.GoogleClientSecretEnc == "" {
		t.Fatalf("secret deveria estar encriptado: %+v", configured)
	}
	authURL, err := e.svc.GoogleConnect(e.ctx, req, e.orgID, configured.ID)
	if err != nil {
		t.Fatalf("connect com config: %v", err)
	}
	if !strings.Contains(authURL, "client_id=client") {
		t.Fatalf("auth url sem client id: %s", authURL)
	}
	if !strings.Contains(authURL, url.QueryEscape("http://localhost:5173/api/v1/s3-destinations/google/callback")) {
		t.Fatalf("auth url sem redirect_uri derivado: %s", authURL)
	}
}

func TestGoogleCallbackStateValidation(t *testing.T) {
	e := newEnv(t)
	req := httptest.NewRequest("GET", "http://localhost:8080/api/v1/s3-destinations/google/callback", nil)
	dest, err := e.svc.CreateS3(e.ctx, e.orgID, &domain.S3Destination{
		Name: "drive", Type: domain.TypeGoogleDrive,
		GoogleClientID: "client", GoogleClientSecretEnc: "secret",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	redirect, err := e.svc.GoogleCallback(e.ctx, req, "bogus-state", "code", "", "")
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if !strings.Contains(redirect, "error%3Ainvalid_state") {
		t.Fatalf("state inválido deveria falhar: %s", redirect)
	}
	state, err := e.svc.GoogleConnect(e.ctx, req, e.orgID, dest.ID)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	parsed, _ := url.Parse(state)
	st := parsed.Query().Get("state")
	redirect, err = e.svc.GoogleCallback(e.ctx, req, st, "code", "access_denied", "user denied")
	if err != nil {
		t.Fatalf("callback deny: %v", err)
	}
	if !strings.Contains(redirect, "error%3Aaccess_denied") {
		t.Fatalf("deny deveria propagar: %s", redirect)
	}
	redirect, _ = e.svc.GoogleCallback(e.ctx, req, st, "code", "", "")
	if !strings.Contains(redirect, "error%3Atoken_exchange") {
		t.Fatalf("code sem exchange deveria falhar: %s", redirect)
	}
	redirect, _ = e.svc.GoogleCallback(e.ctx, req, st, "code", "", "")
	if !strings.Contains(redirect, "error%3Ainvalid_state") {
		t.Fatalf("state consumido deveria falhar: %s", redirect)
	}
}

func TestGoogleBaseURLFromRequest(t *testing.T) {
	e := newEnv(t)
	req := httptest.NewRequest("GET", "http://app.example.com/api/v1/s3-destinations/google/callback", nil)
	if got := e.svc.baseURL(req); got != "http://app.example.com" {
		t.Fatalf("derivado do host = %q", got)
	}
	req.Header.Set("X-Forwarded-Proto", "https")
	if got := e.svc.baseURL(req); got != "https://app.example.com" {
		t.Fatalf("derivado com X-Forwarded-Proto = %q", got)
	}
	e.svc.PublicURL = "https://aether.myserver.com"
	if got := e.svc.baseURL(req); got != "https://aether.myserver.com" {
		t.Fatalf("PublicURL deve ter precedência = %q", got)
	}
}

func TestOIDCStore(t *testing.T) {
	e := newEnv(t)
	provider, err := e.svc.CreateOIDC(e.ctx, e.orgID, "google", "https://accounts.google.com", "client123", "secret", "")
	if err != nil {
		t.Fatalf("create oidc: %v", err)
	}
	if provider.Scopes != "openid email profile" || provider.ClientSecretEnc == "secret" {
		t.Fatalf("oidc inesperado: %+v", provider)
	}
	list, err := e.svc.ListOIDC(e.ctx, e.orgID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list oidc: %v %d", err, len(list))
	}
	if err := e.svc.DeleteOIDC(e.ctx, provider.ID, e.orgID); err != nil {
		t.Fatalf("delete oidc: %v", err)
	}
}

type stubDiscoverer struct {
	url string
	err error
}

func (s *stubDiscoverer) AuthURL(ctx context.Context, issuer, clientID, scopes, providerID string) (string, error) {
	return s.url, s.err
}

func (s *stubDiscoverer) Exchange(ctx context.Context, issuer, clientID, clientSecret, providerID, code string) (*domain.OIDCUser, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &domain.OIDCUser{Email: "user@example.com", Name: "User"}, nil
}

func TestPublicOIDC(t *testing.T) {
	e := newEnv(t)
	first, err := e.svc.CreateOIDC(e.ctx, e.orgID, "google", "https://accounts.google.com", "client123", "secret", "")
	if err != nil {
		t.Fatalf("create oidc: %v", err)
	}
	disabled, err := e.svc.Store.CreateOIDC(e.ctx, &domain.OIDCProvider{
		OrgID: e.orgID, Name: "disabled", Issuer: "https://ex.com", ClientID: "c",
		Scopes: "openid", Enabled: false,
	})
	if err != nil {
		t.Fatalf("create disabled: %v", err)
	}
	providers, err := e.svc.PublicOIDC(e.ctx)
	if err != nil {
		t.Fatalf("public oidc: %v", err)
	}
	if len(providers) != 1 || providers[0].ID != first.ID {
		t.Fatalf("deveria listar só o habilitado: %+v", providers)
	}
	if providers[0].ID == disabled.ID {
		t.Fatalf("disabled não deveria aparecer")
	}
}

func TestOIDCAuthURL(t *testing.T) {
	e := newEnv(t)
	e.svc.OIDC = &stubDiscoverer{url: "https://ex.com/auth?x=1"}
	provider, err := e.svc.CreateOIDC(e.ctx, e.orgID, "google", "https://accounts.google.com", "client123", "secret", "")
	if err != nil {
		t.Fatalf("create oidc: %v", err)
	}
	url, err := e.svc.OIDCAuthURL(e.ctx, provider.ID)
	if err != nil {
		t.Fatalf("auth url: %v", err)
	}
	if url != "https://ex.com/auth?x=1" {
		t.Fatalf("url: %s", url)
	}
	disabled, err := e.svc.Store.CreateOIDC(e.ctx, &domain.OIDCProvider{
		OrgID: e.orgID, Name: "disabled", Issuer: "https://ex.com", ClientID: "c",
		Scopes: "openid", Enabled: false,
	})
	if err != nil {
		t.Fatalf("create disabled: %v", err)
	}
	if _, err := e.svc.OIDCAuthURL(e.ctx, disabled.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("disabled deveria bloquear: %v", err)
	}
}

func TestOIDCCallback(t *testing.T) {
	e := newEnv(t)
	e.svc.OIDC = &stubDiscoverer{}
	provider, err := e.svc.CreateOIDC(e.ctx, e.orgID, "google", "https://accounts.google.com", "client123", "secret", "")
	if err != nil {
		t.Fatalf("create oidc: %v", err)
	}
	user, err := e.svc.OIDCCallback(e.ctx, provider.ID, "code123")
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if user.Email != "user@example.com" {
		t.Fatalf("user: %+v", user)
	}
	disabled, err := e.svc.Store.CreateOIDC(e.ctx, &domain.OIDCProvider{
		OrgID: e.orgID, Name: "disabled", Issuer: "https://ex.com", ClientID: "c",
		Scopes: "openid", Enabled: false,
	})
	if err != nil {
		t.Fatalf("create disabled: %v", err)
	}
	if _, err := e.svc.OIDCCallback(e.ctx, disabled.ID, "code"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("disabled deveria bloquear: %v", err)
	}
}
