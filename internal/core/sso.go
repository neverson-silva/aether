package core

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"aether/internal/domain"
	"aether/internal/security"
)

type oidcDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	Issuer                string `json:"issuer"`
}

func (c *Core) CreateOIDCProvider(orgID, name, issuer, clientID, clientSecret, scopes string) (*domain.OIDCProvider, error) {
	p := &domain.OIDCProvider{
		ID:        "oidc-" + domain.NewID(),
		OrgID:     orgID,
		Name:      name,
		Issuer:    strings.TrimSuffix(issuer, "/"),
		ClientID:  clientID,
		Scopes:    scopes,
		Enabled:   true,
		CreatedAt: time.Now().UTC(),
	}
	if p.Scopes == "" {
		p.Scopes = "openid email profile"
	}
	if clientSecret != "" {
		enc, err := c.Secrets.EncryptString(clientSecret)
		if err != nil {
			return nil, err
		}
		p.ClientSecretEnc = enc
	}
	if err := c.Store.CreateOIDCProvider(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (c *Core) DiscoverOIDC(ctx context.Context, issuer string) (*oidcDiscovery, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	wellKnown := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"
	resp, err := client.Get(wellKnown)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var d oidcDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (c *Core) OIDCAuthURL(p *domain.OIDCProvider) (string, error) {
	disc, err := c.DiscoverOIDC(context.Background(), p.Issuer)
	if err != nil {
		return "", err
	}
	state := make([]byte, 24)
	rand.Read(state)
	callback := c.oidcCallbackURL(p)
	u, _ := url.Parse(disc.AuthorizationEndpoint)
	q := u.Query()
	q.Set("client_id", p.ClientID)
	q.Set("redirect_uri", callback)
	q.Set("response_type", "code")
	q.Set("scope", p.Scopes)
	q.Set("state", base64.RawURLEncoding.EncodeToString(state))
	q.Set("nonce", base64.RawURLEncoding.EncodeToString(state))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c *Core) oidcCallbackURL(p *domain.OIDCProvider) string {
	base := c.Cfg.PublicURL
	if base == "" {
		base = "http://127.0.0.1:8080"
	}
	return strings.TrimSuffix(base, "/") + "/api/v1/sso/" + p.ID + "/callback"
}

type OIDCToken struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
}

type OIDCUser struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Sub           string `json:"sub"`
}

func (c *Core) OIDCExchange(ctx context.Context, p *domain.OIDCProvider, code string) (*OIDCUser, error) {
	disc, err := c.DiscoverOIDC(ctx, p.Issuer)
	if err != nil {
		return nil, err
	}
	secret := ""
	if p.ClientSecretEnc != "" {
		if s, err := c.Secrets.DecryptString(p.ClientSecretEnc); err == nil {
			secret = s
		}
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.oidcCallbackURL(p))
	form.Set("client_id", p.ClientID)
	form.Set("client_secret", secret)
	req, _ := http.NewRequestWithContext(ctx, "POST", disc.TokenEndpoint, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var tok OIDCToken
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, err
	}
	if tok.AccessToken == "" && tok.IDToken == "" {
		return nil, fmt.Errorf("token vazio do provider")
	}
	userReq, _ := http.NewRequestWithContext(ctx, "GET", disc.UserinfoEndpoint, nil)
	userReq.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	uresp, err := client.Do(userReq)
	if err != nil {
		return nil, err
	}
	defer uresp.Body.Close()
	var user OIDCUser
	if err := json.NewDecoder(uresp.Body).Decode(&user); err != nil {
		return nil, err
	}
	if user.Email == "" {
		user.Email = user.Sub
	}
	return &user, nil
}

func (c *Core) OIDCLogin(user *OIDCUser) (*domain.User, string, error) {
	existing, err := c.Store.GetUserByEmail(user.Email)
	if err != nil {
		name := user.Name
		if name == "" {
			name = user.Email
		}
		u, org, err := c.CreateUserAndOrg(user.Email, name, "")
		if err != nil {
			return nil, "", err
		}
		token, err := c.Tokens.Sign(security.Claims{Subject: u.ID, OrgID: org.ID, Role: "owner"}, 24*time.Hour)
		return u, token, err
	}
	orgID, _, err := c.defaultOrg(existing.ID, existing.Name)
	if err != nil {
		return nil, "", err
	}
	token, err := c.Tokens.Sign(security.Claims{Subject: existing.ID, OrgID: orgID, Role: "owner", GlobalRole: existing.GlobalRole}, 24*time.Hour)
	return existing, token, err
}
