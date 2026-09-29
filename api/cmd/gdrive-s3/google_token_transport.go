package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/oauth2"
)

type googleDriveTokenTransport struct {
	next         http.RoundTripper
	oauthConfig  *oauth2.Config
	accessToken  string
	refreshToken string
	mu           sync.Mutex
}

func (t *googleDriveTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token := t.currentAccessToken()
	response, err := t.roundTripWithToken(req, token)
	if err != nil || response.StatusCode != http.StatusUnauthorized {
		return response, err
	}

	responseToken, err := t.refreshAfterUnauthorized(req.Context(), token)
	if err != nil {
		response.Body.Close()
		return nil, err
	}
	if req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
		return response, nil
	}
	response.Body.Close()

	retry := req.Clone(req.Context())
	if req.Body != nil && req.Body != http.NoBody {
		retry.Body, err = req.GetBody()
		if err != nil {
			return nil, err
		}
	}
	return t.roundTripWithToken(retry, responseToken)
}

func (t *googleDriveTokenTransport) currentAccessToken() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.accessToken
}

func (t *googleDriveTokenTransport) refreshAfterUnauthorized(ctx context.Context, failedToken string) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.accessToken != failedToken {
		return t.accessToken, nil
	}
	if strings.TrimSpace(t.refreshToken) == "" {
		return "", errors.New("Google Drive refresh token is missing")
	}
	token, err := t.oauthConfig.TokenSource(ctx, &oauth2.Token{RefreshToken: t.refreshToken}).Token()
	if err != nil {
		return "", errors.New("Google Drive access token refresh failed")
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return "", errors.New("Google Drive refresh returned an empty access token")
	}
	t.accessToken = token.AccessToken
	if token.RefreshToken != "" {
		t.refreshToken = token.RefreshToken
	}
	return t.accessToken, nil
}

func (t *googleDriveTokenTransport) roundTripWithToken(req *http.Request, token string) (*http.Response, error) {
	request := req.Clone(req.Context())
	if request.Header == nil {
		request.Header = make(http.Header)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	return t.next.RoundTrip(request)
}
