package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"aether/internal/settings/domain"
	"aether/internal/storage/gdrive"
)

const (
	googleAuthEndpoint = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenEndpoint = "https://oauth2.googleapis.com/token"
	googleUserInfoEndpoint = "https://www.googleapis.com/oauth2/v2/userinfo"
	googleDriveScope = "https://www.googleapis.com/auth/drive.file"
)

// GoogleConnect starts the OAuth flow for a Google Drive destination.
func (s *Settings) GoogleConnect(ctx context.Context, r *http.Request, orgID, destID uuid.UUID) (string, error) {
	dest, err := s.Store.GetS3(ctx, destID, orgID)
	if err != nil {
		return "", err
	}
	if !dest.IsOAuth() {
		return "", domain.ErrValidation
	}
	if _, err := s.googleClientSecret(dest); err != nil {
		return "", err
	}
	if s.oauthStates == nil {
		s.oauthStates = newOAuthStateStore()
	}
	state, err := s.oauthStates.create(destID, orgID)
	if err != nil {
		return "", err
	}
	if err := s.Store.UpdateS3OAuth(ctx, destID, orgID, domain.OAuthConnecting, "", "", ""); err != nil {
		return "", err
	}
	params := url.Values{}
	params.Set("client_id", dest.GoogleClientID)
	params.Set("redirect_uri", s.googleRedirectURI(r))
	params.Set("response_type", "code")
	params.Set("scope", googleDriveScope)
	params.Set("state", state)
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")
	return googleAuthEndpoint + "?" + params.Encode(), nil
}

func (s *Settings) baseURL(r *http.Request) string {
	if s.PublicURL != "" {
		return strings.TrimSuffix(s.PublicURL, "/")
	}
	if r == nil {
		return ""
	}
	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// googleClientSecret returns the destination's OAuth client secret.
func (s *Settings) googleClientSecret(dest *domain.S3Destination) (string, error) {
	if dest.GoogleClientID == "" || dest.GoogleClientSecretEnc == "" {
		return "", ErrGoogleNotConfigured
	}
	secret, err := s.Passwords.Decrypt(dest.GoogleClientSecretEnc)
	if err != nil {
		return "", ErrGoogleNotConfigured
	}
	return secret, nil
}

type googleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

type googleUserInfo struct {
	Email string `json:"email"`
}

func (s *Settings) googleRedirectURI(r *http.Request) string {
	if s.GoogleRedirectURI != "" {
		return s.GoogleRedirectURI
	}
	return s.baseURL(r) + "/api/v1/s3-destinations/google/callback"
}

func (s *Settings) httpClient() *http.Client {
	if s.HTTPClient != nil {
		return s.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

// GoogleCallback validates the OAuth state and exchanges the code for tokens.
func (s *Settings) GoogleCallback(ctx context.Context, r *http.Request, state, code, oauthError, oauthErrorDescription string) (string, error) {
	base := s.baseURL(r)
	redirect := func(status string) string {
		return base + "/storage?oauth=google-drive&status=" + url.QueryEscape(status)
	}
	if oauthError != "" {
		return redirect("error:" + oauthError), nil
	}
	if s.oauthStates == nil {
		return redirect("error:invalid_state"), nil
	}
	entry, ok := s.oauthStates.consume(state)
	if !ok {
		return redirect("error:invalid_state"), nil
	}
	dest, err := s.Store.GetS3(ctx, entry.DestID, entry.OrgID)
	if err != nil {
		return redirect("error:not_found"), nil
	}
	if !dest.IsOAuth() {
		return redirect("error:invalid_destination"), nil
	}
	secret, err := s.googleClientSecret(dest)
	if err != nil {
		return redirect("error:not_configured"), nil
	}

	tokens, err := s.exchangeCode(ctx, code, dest.GoogleClientID, secret, s.googleRedirectURI(r))
	if err != nil {
		return redirect("error:token_exchange"), nil
	}
	accessEnc, err := s.Passwords.Encrypt(tokens.AccessToken)
	if err != nil {
		return redirect("error:storage"), nil
	}
	refreshEnc := ""
	if tokens.RefreshToken != "" {
		refreshEnc, err = s.Passwords.Encrypt(tokens.RefreshToken)
		if err != nil {
			return redirect("error:storage"), nil
		}
	}
	email := ""
	if info, err := s.fetchUserInfo(ctx, tokens.AccessToken); err == nil {
		email = info.Email
	}
	if err := s.Store.UpdateS3OAuth(ctx, entry.DestID, entry.OrgID, domain.OAuthConnected, email, accessEnc, refreshEnc); err != nil {
		return redirect("error:storage"), nil
	}
	if name := strings.TrimSpace(dest.Bucket); name != "" {
		if client, err := gdrive.NewTokenClient(tokens.AccessToken); err == nil {
			_, _ = gdrive.EnsureRootFolder(ctx, client, name)
		}
	}
	return redirect("connected"), nil
}

func (s *Settings) exchangeCode(ctx context.Context, code, clientID, clientSecret, redirectURI string) (googleTokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("redirect_uri", redirectURI)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	return s.tokenRequest(ctx, form)
}

func (s *Settings) refreshTokenRequest(ctx context.Context, refreshToken, clientID, clientSecret string) (googleTokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	return s.tokenRequest(ctx, form)
}

func (s *Settings) tokenRequest(ctx context.Context, form url.Values) (googleTokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return googleTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.httpClient().Do(req)
	if err != nil {
		return googleTokenResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return googleTokenResponse{}, err
	}
	var out googleTokenResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return googleTokenResponse{}, err
	}
	if out.Error != "" {
		return googleTokenResponse{}, fmt.Errorf("google token: %s", out.Error)
	}
	if out.AccessToken == "" {
		return googleTokenResponse{}, errors.New("google token: empty access token")
	}
	return out, nil
}

func (s *Settings) fetchUserInfo(ctx context.Context, accessToken string) (googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoEndpoint, nil)
	if err != nil {
		return googleUserInfo{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := s.httpClient().Do(req)
	if err != nil {
		return googleUserInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return googleUserInfo{}, errors.New("google userinfo failed")
	}
	var info googleUserInfo
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&info); err != nil {
		return googleUserInfo{}, err
	}
	return info, nil
}

// googleAccessToken returns a usable access token, refreshing and persisting
// the new token when the stored one is invalid.
func (s *Settings) googleAccessToken(ctx context.Context, dest *domain.S3Destination) (string, error) {
	refreshToken, err := s.Passwords.Decrypt(dest.RefreshTokenEnc)
	if err != nil || refreshToken == "" {
		_ = s.Store.UpdateS3OAuth(ctx, dest.ID, dest.OrgID, domain.OAuthReauthRequired, dest.OAuthEmail, "", "")
		return "", domain.ErrReauthRequired
	}
	accessToken, err := s.Passwords.Decrypt(dest.AccessTokenEnc)
	if err == nil && accessToken != "" {
		if ok, _ := s.validAccessToken(ctx, accessToken); ok {
			return accessToken, nil
		}
	}
	clientSecret, err := s.googleClientSecret(dest)
	if err != nil {
		return "", domain.ErrReauthRequired
	}
	tokens, err := s.refreshTokenRequest(ctx, refreshToken, dest.GoogleClientID, clientSecret)
	if err != nil {
		_ = s.Store.UpdateS3OAuth(ctx, dest.ID, dest.OrgID, domain.OAuthReauthRequired, dest.OAuthEmail, "", "")
		return "", domain.ErrReauthRequired
	}
	accessEnc, err := s.Passwords.Encrypt(tokens.AccessToken)
	if err != nil {
		return "", err
	}
	if err := s.Store.UpdateS3OAuth(ctx, dest.ID, dest.OrgID, domain.OAuthConnected, dest.OAuthEmail, accessEnc, dest.RefreshTokenEnc); err != nil {
		return "", err
	}
	return tokens.AccessToken, nil
}

func (s *Settings) validAccessToken(ctx context.Context, accessToken string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoEndpoint, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := s.httpClient().Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	return resp.StatusCode == http.StatusOK, nil
}

// GoogleDisconnect clears the stored Google credentials.
func (s *Settings) GoogleDisconnect(ctx context.Context, orgID, destID uuid.UUID) error {
	return s.Store.UpdateS3OAuth(ctx, destID, orgID, domain.OAuthNone, "", "", "")
}

// googleOAuthClient builds an HTTP client that injects the Google access
// token and refreshes it once on 401.
func (s *Settings) googleOAuthClient(dest *domain.S3Destination) (*http.Client, error) {
	mu := &sync.Mutex{}
	current := ""
	getToken := func() (string, error) {
		mu.Lock()
		defer mu.Unlock()
		if current != "" {
			return current, nil
		}
		tok, err := s.googleAccessToken(context.Background(), dest)
		if err != nil {
			return "", err
		}
		current = tok
		return tok, nil
	}
	rt := &oauthRoundTripper{next: s.httpClient().Transport, getToken: getToken, refresh: func() {
		mu.Lock()
		current = ""
		mu.Unlock()
	}}
	if rt.next == nil {
		rt.next = http.DefaultTransport
	}
	return &http.Client{Timeout: 60 * time.Second, Transport: rt}, nil
}

type oauthRoundTripper struct {
	next     http.RoundTripper
	getToken func() (string, error)
	refresh  func()
}

func (rt *oauthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := rt.getToken()
	if err != nil {
		return nil, err
	}
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+token)
	resp, err := rt.next.RoundTrip(req2)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		rt.refresh()
		token, err = rt.getToken()
		if err != nil {
			return nil, err
		}
		req3 := req.Clone(req.Context())
		req3.Header.Set("Authorization", "Bearer "+token)
		return rt.next.RoundTrip(req3)
	}
	return resp, nil
}
