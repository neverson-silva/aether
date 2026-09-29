package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestGoogleDriveTokenTransportRefreshesStoredConnection(t *testing.T) {
	refreshGrants := make(chan url.Values, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "invalid refresh request", http.StatusBadRequest)
				return
			}
			refreshGrants <- r.Form
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "refreshed-access-token", "token_type": "Bearer", "expires_in": 3600})
			return
		}
		if r.Header.Get("Authorization") != "Bearer refreshed-access-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := &googleDriveTokenTransport{
		next:         http.DefaultTransport,
		oauthConfig:  &oauth2.Config{ClientID: "stored-client-id", ClientSecret: "stored-client-secret", Endpoint: oauth2.Endpoint{TokenURL: server.URL + "/token"}},
		accessToken:  "expired-access-token",
		refreshToken: "stored-refresh-token",
	}
	client := &http.Client{Transport: transport}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/drive", nil)
	if err != nil {
		t.Fatalf("create Drive request: %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Drive request: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("Drive status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	refreshGrant := <-refreshGrants
	if refreshGrant.Get("grant_type") != "refresh_token" || refreshGrant.Get("refresh_token") != "stored-refresh-token" {
		t.Fatalf("refresh grant = %s", strings.TrimSpace(refreshGrant.Encode()))
	}
	if refreshGrant.Get("code") != "" || refreshGrant.Get("redirect_uri") != "" {
		t.Fatalf("interactive OAuth fields were sent: %s", refreshGrant.Encode())
	}
}
