package infra

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"aether/internal/modules/settings/domain"
)

type OIDCDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	Issuer                string `json:"issuer"`
}

type OIDCDiscoverer struct {
	PublicURL string
	Client    *http.Client
}

func NewOIDCDiscoverer(publicURL string) *OIDCDiscoverer {
	return &OIDCDiscoverer{
		PublicURL: publicURL,
		Client:    &http.Client{Timeout: 15 * time.Second},
	}
}

type OIDCToken struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
}

func (d *OIDCDiscoverer) Exchange(ctx context.Context, issuer, clientID, clientSecret, providerID, code string) (*domain.OIDCUser, error) {
	disc, err := d.Discover(ctx, issuer)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", d.CallbackURL(providerID))
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, disc.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := d.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var token OIDCToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	if token.AccessToken == "" && token.IDToken == "" {
		return nil, errors.New("empty provider token")
	}
	userReq, err := http.NewRequestWithContext(ctx, http.MethodGet, disc.UserinfoEndpoint, nil)
	if err != nil {
		return nil, err
	}
	userReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
	uresp, err := d.Client.Do(userReq)
	if err != nil {
		return nil, err
	}
	defer uresp.Body.Close()
	var user domain.OIDCUser
	if err := json.NewDecoder(uresp.Body).Decode(&user); err != nil {
		return nil, err
	}
	if user.Email == "" {
		user.Email = user.Sub
	}
	return &user, nil
}

func (d *OIDCDiscoverer) Discover(ctx context.Context, issuer string) (*OIDCDiscovery, error) {
	wellKnown := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"
	resp, err := d.Client.Get(wellKnown)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var disc OIDCDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&disc); err != nil {
		return nil, err
	}
	return &disc, nil
}

func (d *OIDCDiscoverer) AuthURL(ctx context.Context, issuer, clientID, scopes, providerID string) (string, error) {
	disc, err := d.Discover(ctx, issuer)
	if err != nil {
		return "", err
	}
	state := make([]byte, 24)
	_, _ = rand.Read(state)
	callback := d.CallbackURL(providerID)
	u, err := url.Parse(disc.AuthorizationEndpoint)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("client_id", clientID)
	q.Set("redirect_uri", callback)
	q.Set("response_type", "code")
	q.Set("scope", scopes)
	q.Set("state", base64.RawURLEncoding.EncodeToString(state))
	q.Set("nonce", base64.RawURLEncoding.EncodeToString(state))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (d *OIDCDiscoverer) CallbackURL(providerID string) string {
	base := d.PublicURL
	if base == "" {
		base = "http://127.0.0.1:8080"
	}
	return strings.TrimSuffix(base, "/") + "/api/v1/sso/" + providerID + "/callback"
}
