package application

import (
	"context"
	"errors"
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
	dest, err := e.svc.CreateS3(e.ctx, e.orgID, "backups", "s3.amazonaws.com", "bucket", "", "AKIA123", "secret456")
	if err != nil {
		t.Fatalf("create s3: %v", err)
	}
	if dest.Region != "us-east-1" {
		t.Fatalf("region default inesperada")
	}
	if dest.AccessKeyEnc == "AKIA123" || dest.AccessKeyEnc == "" {
		t.Fatalf("access key deveria estar encriptada")
	}
	if _, err := e.svc.CreateS3(e.ctx, e.orgID, "", "x", "y", "", "a", "b"); !errors.Is(err, domain.ErrValidation) {
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
