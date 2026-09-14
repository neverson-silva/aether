package infra

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"aether/internal/modules/settings/domain"
	"aether/internal/platform/security"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OIDCDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
	Issuer                string `json:"issuer"`
}

type OIDCDiscoverer struct {
	PublicURL     string
	Client        *http.Client
	AllowLoopback bool
	states        map[string]oidcState
	stateMu       sync.Mutex
	stateDB       *pgxpool.Pool
}

type oidcState struct {
	ProviderID string
	Nonce      string
	Expires    time.Time
}

func NewOIDCDiscoverer(publicURL string, pools ...*pgxpool.Pool) *OIDCDiscoverer {
	var stateDB *pgxpool.Pool
	if len(pools) > 0 {
		stateDB = pools[0]
	}
	return &OIDCDiscoverer{
		PublicURL: publicURL,
		Client:    security.NewHTTPClient(15 * time.Second),
		states:    make(map[string]oidcState),
		stateDB:   stateDB,
	}
}

type OIDCToken struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
}

func (d *OIDCDiscoverer) Exchange(ctx context.Context, issuer, clientID, clientSecret, providerID, code, state string) (*domain.OIDCUser, error) {
	nonce, err := d.consumeState(state, providerID)
	if err != nil {
		return nil, err
	}
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
	if err := d.validateURL(disc.TokenEndpoint); err != nil {
		return nil, err
	}
	resp, err := d.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, errors.New("provider token request failed")
	}
	var token OIDCToken
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&token); err != nil {
		return nil, err
	}
	if token.AccessToken == "" || token.IDToken == "" || disc.JWKSURI == "" {
		return nil, errors.New("empty provider token")
	}
	claims, err := d.verifyIDToken(ctx, token.IDToken, disc.JWKSURI, issuer, clientID, nonce)
	if err != nil {
		return nil, err
	}
	userReq, err := http.NewRequestWithContext(ctx, http.MethodGet, disc.UserinfoEndpoint, nil)
	if err != nil {
		return nil, err
	}
	userReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
	if err := d.validateURL(disc.UserinfoEndpoint); err != nil {
		return nil, err
	}
	uresp, err := d.Client.Do(userReq)
	if err != nil {
		return nil, err
	}
	defer uresp.Body.Close()
	if uresp.StatusCode < http.StatusOK || uresp.StatusCode >= http.StatusMultipleChoices {
		return nil, errors.New("provider userinfo request failed")
	}
	var user domain.OIDCUser
	if err := json.NewDecoder(io.LimitReader(uresp.Body, 1<<20)).Decode(&user); err != nil {
		return nil, err
	}
	if user.Email == "" {
		user.Email = user.Sub
	}
	if !user.EmailVerified || user.Sub == "" || claims.Subject != user.Sub {
		return nil, errors.New("provider identity is not verified")
	}
	return &user, nil
}

func (d *OIDCDiscoverer) Discover(ctx context.Context, issuer string) (*OIDCDiscovery, error) {
	if err := d.validateURL(issuer); err != nil {
		return nil, err
	}
	wellKnown := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"
	resp, err := d.Client.Get(wellKnown)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, errors.New("provider discovery request failed")
	}
	var disc OIDCDiscovery
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&disc); err != nil {
		return nil, err
	}
	if disc.AuthorizationEndpoint == "" || disc.TokenEndpoint == "" || disc.UserinfoEndpoint == "" || disc.Issuer == "" || disc.JWKSURI == "" {
		return nil, errors.New("provider discovery metadata incomplete")
	}
	if disc.Issuer != "" {
		issuerURL, err := url.Parse(issuer)
		if err != nil {
			return nil, err
		}
		discoveredURL, err := url.Parse(disc.Issuer)
		if err != nil || !strings.EqualFold(strings.TrimSuffix(issuerURL.String(), "/"), strings.TrimSuffix(discoveredURL.String(), "/")) {
			return nil, errors.New("provider issuer mismatch")
		}
	}
	return &disc, nil
}

func (d *OIDCDiscoverer) AuthURL(ctx context.Context, issuer, clientID, scopes, providerID string) (string, error) {
	disc, err := d.Discover(ctx, issuer)
	if err != nil {
		return "", err
	}
	rawState := make([]byte, 24)
	rawNonce := make([]byte, 24)
	if _, err := rand.Read(rawState); err != nil {
		return "", err
	}
	if _, err := rand.Read(rawNonce); err != nil {
		return "", err
	}
	state := base64.RawURLEncoding.EncodeToString(rawState)
	nonce := base64.RawURLEncoding.EncodeToString(rawNonce)
	if d.stateDB == nil {
		d.stateMu.Lock()
		if len(d.states) >= 4096 {
			for key, entry := range d.states {
				if !entry.Expires.After(time.Now()) {
					delete(d.states, key)
				}
			}
			if len(d.states) >= 4096 {
				d.stateMu.Unlock()
				return "", errors.New("provider authentication state capacity exceeded")
			}
		}
		d.states[state] = oidcState{ProviderID: providerID, Nonce: nonce, Expires: time.Now().Add(10 * time.Minute)}
		d.stateMu.Unlock()
	}
	if d.stateDB != nil {
		if _, err := d.stateDB.Exec(ctx, `INSERT INTO oidc_auth_states (state_key, provider_id, nonce, expires_at) VALUES ($1, $2, $3, now() + interval '10 minutes')`, state, providerID, nonce); err != nil {
			return "", err
		}
	}
	callback := d.CallbackURL(providerID)
	u, err := url.Parse(disc.AuthorizationEndpoint)
	if err != nil {
		return "", err
	}
	if err := d.validateURL(u.String()); err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("client_id", clientID)
	q.Set("redirect_uri", callback)
	q.Set("response_type", "code")
	q.Set("scope", scopes)
	q.Set("state", state)
	q.Set("nonce", nonce)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (d *OIDCDiscoverer) consumeState(state, providerID string) (string, error) {
	if state == "" {
		return "", errors.New("missing provider state")
	}
	if d.stateDB != nil {
		var nonce string
		if err := d.stateDB.QueryRow(context.Background(), `DELETE FROM oidc_auth_states WHERE state_key = $1 AND provider_id = $2 AND expires_at > now() RETURNING nonce`, state, providerID).Scan(&nonce); err != nil {
			return "", errors.New("invalid provider state")
		}
		return nonce, nil
	}
	d.stateMu.Lock()
	defer d.stateMu.Unlock()
	entry, ok := d.states[state]
	delete(d.states, state)
	if !ok || entry.ProviderID != providerID || !entry.Expires.After(time.Now()) {
		return "", errors.New("invalid provider state")
	}
	return entry.Nonce, nil
}

type oidcJWKS struct {
	Keys []oidcJWK `json:"keys"`
}
type oidcJWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
	Alg string `json:"alg"`
}

type oidcClaims struct {
	jwt.RegisteredClaims
	Nonce string `json:"nonce"`
}

func (d *OIDCDiscoverer) verifyIDToken(ctx context.Context, raw, jwksURL, issuer, clientID, nonce string) (*oidcClaims, error) {
	if err := d.validateURL(jwksURL); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := d.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("provider jwks request failed")
	}
	var set oidcJWKS
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&set); err != nil {
		return nil, err
	}
	keyFunc := func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodRS256 {
			return nil, errors.New("unsupported provider signing method")
		}
		kid, _ := token.Header["kid"].(string)
		for _, key := range set.Keys {
			if key.Kid == kid && key.Kty == "RSA" {
				return rsaJWK(key)
			}
		}
		return nil, errors.New("provider signing key not found")
	}
	claims := &oidcClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, keyFunc, jwt.WithIssuer(issuer), jwt.WithAudience(clientID), jwt.WithValidMethods([]string{"RS256"}))
	if err != nil || !token.Valid || claims.Subject == "" || claims.Nonce != nonce {
		return nil, errors.New("invalid provider id token")
	}
	return claims, nil
}

func rsaJWK(key oidcJWK) (*rsa.PublicKey, error) {
	n, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, err
	}
	e, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil || len(e) == 0 || len(e) > 4 {
		return nil, errors.New("invalid provider key")
	}
	var exponent uint32
	for _, b := range e {
		exponent = exponent<<8 | uint32(b)
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(exponent)}, nil
}

func (d *OIDCDiscoverer) validateURL(raw string) error {
	if d.AllowLoopback {
		return security.ValidateOutboundURLForTest(raw)
	}
	return security.ValidateOutboundURL(raw)
}

func (d *OIDCDiscoverer) CallbackURL(providerID string) string {
	base := d.PublicURL
	if base == "" {
		base = "http://127.0.0.1:8080"
	}
	return strings.TrimSuffix(base, "/") + "/api/v1/sso/" + providerID + "/callback"
}
