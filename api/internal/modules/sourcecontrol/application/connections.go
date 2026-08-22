package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/sourcecontrol/domain"
	"aether/internal/modules/sourcecontrol/ports"
	"aether/internal/platform/scm/github"
)

type Connections struct {
	Store     ports.ConnectionStore
	Provider  ports.InstallationProvider
	Cipher    ports.SecretCipher
	PublicURL string
	APIURL    string
}

func (s *Connections) ConnectGitHub(ctx context.Context, organizationID uuid.UUID, installationID string) (*domain.Connection, error) {
	connection := &domain.Connection{
		OrganizationID: organizationID, Provider: domain.ProviderGitHub,
		InstallationID: installationID, Status: "active", Metadata: []byte(`{}`),
	}
	connections, err := s.Store.ListConnections(ctx, organizationID, domain.ProviderGitHub)
	if err != nil {
		return nil, err
	}
	for _, existing := range connections {
		if existing.InstallationID == "" && existing.CredentialsEnc != "" {
			connection.CredentialsEnc = existing.CredentialsEnc
			provider, providerErr := s.providerFromConnection(&existing)
			if providerErr != nil {
				return nil, providerErr
			}
			installation, providerErr := provider.GetInstallation(ctx, installationID)
			if providerErr != nil {
				return nil, providerErr
			}
			connection.ExternalAccountID = installation.AccountID
			connection.ExternalAccountName = installation.AccountName
			break
		}
	}
	if connection.ExternalAccountID == "" {
		if s.Provider == nil {
			return nil, errors.New("github app is not configured")
		}
		installation, err := s.Provider.GetInstallation(ctx, installationID)
		if err != nil {
			return nil, err
		}
		connection.ExternalAccountID = installation.AccountID
		connection.ExternalAccountName = installation.AccountName
	}
	return s.Store.UpsertConnection(ctx, connection)
}

func (s *Connections) List(ctx context.Context, organizationID uuid.UUID) ([]domain.Connection, error) {
	return s.Store.ListConnections(ctx, organizationID, domain.ProviderGitHub)
}

func (s *Connections) ListRepositories(ctx context.Context, installationID string) ([]domain.Repository, error) {
	provider, err := s.ProviderForInstallation(ctx, installationID)
	if err != nil {
		if s.Provider == nil {
			return nil, err
		}
		return s.Provider.ListRepositories(ctx, installationID)
	}
	return provider.ListRepositories(ctx, installationID)
}

func (s *Connections) ListBranches(ctx context.Context, installationID, repositoryID string) ([]domain.Branch, error) {
	provider, err := s.ProviderForInstallation(ctx, installationID)
	if err != nil {
		return nil, err
	}
	return provider.GetBranches(ctx, repositoryID)
}

func (s *Connections) GetFile(ctx context.Context, installationID, repositoryID, path, ref string) (string, error) {
	provider, err := s.ProviderForInstallation(ctx, installationID)
	if err != nil {
		return "", err
	}
	return provider.GetFile(ctx, repositoryID, path, ref)
}

func (s *Connections) StartManifest(ctx context.Context, organizationID, userID uuid.UUID, publicURL, returnURL string) (string, string, error) {
	if strings.TrimSpace(publicURL) == "" {
		return "", "", errors.New("aether public url is not configured")
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	state := hex.EncodeToString(raw)
	if err := s.Store.CreateManifestState(ctx, state, organizationID, userID, returnURL, time.Now().UTC().Add(10*time.Minute)); err != nil {
		return "", "", err
	}
	manifest, err := json.Marshal(map[string]any{
		"name":                "Aether-" + strings.ToUpper(state[:8]),
		"url":                 strings.TrimRight(publicURL, "/"),
		"hook_attributes":     map[string]string{"url": strings.TrimRight(publicURL, "/") + "/api/v1/webhooks/github"},
		"redirect_url":        strings.TrimRight(publicURL, "/") + "/api/v1/source-control/github/manifest/callback",
		"setup_url":           strings.TrimRight(publicURL, "/") + "/api/v1/source-control/github/install-callback?return_url=" + url.QueryEscape(returnURL),
		"public":              true,
		"default_permissions": map[string]string{"contents": "read", "metadata": "read"},
		"default_events":      []string{"push"},
		"callback_urls":       []string{strings.TrimRight(publicURL, "/") + "/api/v1/source-control/github/callback"},
	})
	if err != nil {
		return "", "", err
	}
	return string(manifest), state, nil
}

func (s *Connections) CompleteManifest(ctx context.Context, state, code string) (*domain.Connection, string, error) {
	organizationID, _, _, err := s.Store.ConsumeManifestState(ctx, state)
	if err != nil {
		return nil, "", err
	}
	app, err := github.ConvertManifest(ctx, s.APIURL, code)
	if err != nil {
		return nil, "", err
	}
	credentials, err := json.Marshal(domain.GitHubAppCredentials{
		AppID: app.ID, Slug: app.Slug, ClientID: app.ClientID, ClientSecret: app.ClientSecret,
		PrivateKey: app.PrivateKey, WebhookSecret: app.WebhookSecret,
	})
	if err != nil {
		return nil, "", err
	}
	if s.Cipher == nil {
		return nil, "", errors.New("source control secret cipher is not configured")
	}
	credentialsEnc, err := s.Cipher.Encrypt(string(credentials))
	if err != nil {
		return nil, "", err
	}
	connection, err := s.Store.UpsertConnection(ctx, &domain.Connection{
		OrganizationID: organizationID, Provider: domain.ProviderGitHub, ExternalAccountName: app.Slug,
		Status: "created", Metadata: []byte(`{}`), CredentialsEnc: credentialsEnc,
	})
	if err != nil {
		return nil, "", err
	}
	return connection, "https://github.com/apps/" + app.Slug + "/installations/new?state=" + url.QueryEscape(connection.ID.String()), nil
}

func (s *Connections) CompleteInstallation(ctx context.Context, connectionID uuid.UUID, installationID string) (*domain.Connection, error) {
	connection, err := s.Store.GetConnection(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	provider, err := s.providerFromConnection(connection)
	if err != nil {
		return nil, err
	}
	installation, err := provider.GetInstallation(ctx, installationID)
	if err != nil {
		return nil, err
	}
	return s.Store.AttachInstallation(ctx, connectionID, installation.ID, installation.AccountID, installation.AccountName)
}

func (s *Connections) ProviderForInstallation(ctx context.Context, installationID string) (*github.Provider, error) {
	connection, err := s.Store.GetConnectionByInstallation(ctx, domain.ProviderGitHub, installationID)
	if err != nil {
		return nil, err
	}
	if s.Cipher == nil || connection.CredentialsEnc == "" {
		return nil, fmt.Errorf("github app credentials are not configured for installation %s", installationID)
	}
	provider, err := s.providerFromConnection(connection)
	if err != nil {
		return nil, err
	}
	return provider.ForInstallation(installationID), nil
}

func (s *Connections) providerFromConnection(connection *domain.Connection) (*github.Provider, error) {
	plain, err := s.Cipher.Decrypt(connection.CredentialsEnc)
	if err != nil {
		return nil, err
	}
	var credentials domain.GitHubAppCredentials
	if err := json.Unmarshal([]byte(plain), &credentials); err != nil {
		return nil, err
	}
	provider, err := github.New(credentials.AppID, credentials.PrivateKey, credentials.WebhookSecret, s.APIURL)
	if err != nil {
		return nil, err
	}
	return provider, nil
}
