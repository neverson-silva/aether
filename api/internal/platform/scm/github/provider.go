package github

import (
	"bytes"
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"aether/internal/modules/sourcecontrol/domain"
)

type Provider struct {
	AppID          int64
	PrivateKey     *rsa.PrivateKey
	WebhookKey     []byte
	APIURL         string
	Client         *http.Client
	InstallationID string
}

type APIError struct {
	StatusCode int
	Status     string
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("github api returned %s: %s", e.Status, e.Message)
}

type ManifestApp struct {
	ID            int64  `json:"id"`
	Slug          string `json:"slug"`
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	PrivateKey    string `json:"pem"`
	WebhookSecret string `json:"webhook_secret"`
}

func New(appID int64, privateKeyPEM, webhookSecret, apiURL string) (*Provider, error) {
	key, err := parsePrivateKey([]byte(privateKeyPEM))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(apiURL) == "" {
		apiURL = "https://api.github.com"
	}
	return &Provider{
		AppID: appID, PrivateKey: key, WebhookKey: []byte(webhookSecret),
		APIURL: strings.TrimRight(apiURL, "/"), Client: http.DefaultClient,
	}, nil
}

func ConvertManifest(ctx context.Context, apiURL, code string) (ManifestApp, error) {
	provider := &Provider{APIURL: strings.TrimRight(apiURL, "/"), Client: http.DefaultClient}
	if provider.APIURL == "" {
		provider.APIURL = "https://api.github.com"
	}
	var response ManifestApp
	if err := provider.request(ctx, http.MethodPost, "/app-manifests/"+url.PathEscape(code)+"/conversions", "", nil, &response); err != nil {
		return ManifestApp{}, err
	}
	return response, nil
}

func (p *Provider) ForInstallation(installationID string) *Provider {
	copy := *p
	copy.InstallationID = installationID
	return &copy
}

func (p *Provider) VerifyWebhook(headers http.Header, body []byte) error {
	if len(p.WebhookKey) == 0 {
		return errors.New("github webhook secret is not configured")
	}
	signature := headers.Get("X-Hub-Signature-256")
	if signature == "" {
		return errors.New("github webhook signature is missing")
	}
	if !verifySignature(body, signature, p.WebhookKey) {
		return errors.New("invalid github webhook signature")
	}
	return nil
}

func (p *Provider) ParsePushWebhook(headers http.Header, body []byte) (domain.PushEvent, error) {
	return ParsePushWebhook(headers, body)
}

func ParsePushWebhook(headers http.Header, body []byte) (domain.PushEvent, error) {
	if headers.Get("X-GitHub-Event") != "push" {
		return domain.PushEvent{}, errors.New("unsupported github webhook event")
	}
	var payload struct {
		Ref        string `json:"ref"`
		Before     string `json:"before"`
		After      string `json:"after"`
		Repository struct {
			ID            int64  `json:"id"`
			Name          string `json:"name"`
			FullName      string `json:"full_name"`
			DefaultBranch string `json:"default_branch"`
			Owner         struct {
				Login string `json:"login"`
			} `json:"owner"`
		} `json:"repository"`
		Installation struct {
			ID int64 `json:"id"`
		} `json:"installation"`
		HeadCommit struct {
			ID      string `json:"id"`
			Message string `json:"message"`
			Author  struct {
				Name string `json:"name"`
			} `json:"author"`
		} `json:"head_commit"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return domain.PushEvent{}, fmt.Errorf("decode github push: %w", err)
	}
	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")
	return domain.PushEvent{
		SourceEvent: domain.SourceEvent{
			Provider: domain.ProviderGitHub, DeliveryID: headers.Get("X-GitHub-Delivery"),
			EventType: headers.Get("X-GitHub-Event"), InstallationID: strconv.FormatInt(payload.Installation.ID, 10),
			Repository: domain.Repository{
				ID: strconv.FormatInt(payload.Repository.ID, 10), Owner: payload.Repository.Owner.Login,
				Name: payload.Repository.Name, FullName: payload.Repository.FullName,
				DefaultBranch: payload.Repository.DefaultBranch,
			}, OccurredAt: time.Now().UTC(),
		},
		Ref: payload.Ref, Branch: branch, BeforeSHA: payload.Before, AfterSHA: payload.After,
		Commit: domain.Commit{SHA: payload.HeadCommit.ID, Message: payload.HeadCommit.Message, Author: payload.HeadCommit.Author.Name},
	}, nil
}

func (p *Provider) ListRepositories(ctx context.Context, installationID string) ([]domain.Repository, error) {
	token, err := p.installationToken(ctx, installationID)
	if err != nil {
		return nil, err
	}
	var response struct {
		Repositories []repositoryResponse `json:"repositories"`
	}
	if err := p.request(ctx, http.MethodGet, "/installation/repositories?per_page=100", token, nil, &response); err != nil {
		return nil, err
	}
	result := make([]domain.Repository, 0, len(response.Repositories))
	for _, repository := range response.Repositories {
		result = append(result, repository.domain())
	}
	return result, nil
}

func (p *Provider) GetInstallation(ctx context.Context, installationID string) (domain.Installation, error) {
	token, err := p.appJWT()
	if err != nil {
		return domain.Installation{}, err
	}
	var response struct {
		ID      int64 `json:"id"`
		Account struct {
			ID    int64  `json:"id"`
			Login string `json:"login"`
			Name  string `json:"name"`
		} `json:"account"`
	}
	if err := p.request(ctx, http.MethodGet, "/app/installations/"+url.PathEscape(installationID), token, nil, &response); err != nil {
		return domain.Installation{}, err
	}
	accountName := response.Account.Login
	if accountName == "" {
		accountName = response.Account.Name
	}
	return domain.Installation{
		ID: strconv.FormatInt(response.ID, 10), AccountID: strconv.FormatInt(response.Account.ID, 10), AccountName: accountName,
	}, nil
}

func (p *Provider) Uninstall(ctx context.Context, installationID string) error {
	token, err := p.appJWT()
	if err != nil {
		return err
	}
	return p.request(ctx, http.MethodDelete, "/app/installations/"+url.PathEscape(installationID), token, nil, nil)
}

func (p *Provider) GetRepository(ctx context.Context, repoID string) (domain.Repository, error) {
	var response repositoryResponse
	token, err := p.installationToken(ctx, "")
	if err != nil {
		return domain.Repository{}, err
	}
	if err := p.request(ctx, http.MethodGet, "/repositories/"+url.PathEscape(repoID), token, nil, &response); err != nil {
		return domain.Repository{}, err
	}
	return response.domain(), nil
}

func (p *Provider) GetBranches(ctx context.Context, repoID string) ([]domain.Branch, error) {
	repository, err := p.GetRepository(ctx, repoID)
	if err != nil {
		return nil, err
	}
	token, err := p.installationTokenForRepository(ctx, repository)
	if err != nil {
		return nil, err
	}
	var response []struct {
		Name string `json:"name"`
	}
	if err := p.request(ctx, http.MethodGet, "/repos/"+repository.FullName+"/branches?per_page=100", token, nil, &response); err != nil {
		return nil, err
	}
	branches := make([]domain.Branch, 0, len(response))
	for _, branch := range response {
		branches = append(branches, domain.Branch{Name: branch.Name})
	}
	return branches, nil
}

func (p *Provider) GetFile(ctx context.Context, repoID, path, ref string) (string, error) {
	repository, err := p.GetRepository(ctx, repoID)
	if err != nil {
		return "", err
	}
	token, err := p.installationTokenForRepository(ctx, repository)
	if err != nil {
		return "", err
	}
	query := "?ref=" + url.QueryEscape(ref)
	var response struct {
		Type     string `json:"type"`
		Encoding string `json:"encoding"`
		Content  string `json:"content"`
	}
	if err := p.request(ctx, http.MethodGet, "/repos/"+repository.FullName+"/contents/"+strings.TrimLeft(path, "/")+query, token, nil, &response); err != nil {
		return "", err
	}
	if response.Type != "file" || response.Encoding != "base64" {
		return "", errors.New("github path is not a file")
	}
	content, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(response.Content, "\n", ""))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (p *Provider) GetCommit(ctx context.Context, repoID, sha string) (domain.Commit, error) {
	repository, err := p.GetRepository(ctx, repoID)
	if err != nil {
		return domain.Commit{}, err
	}
	token, err := p.installationTokenForRepository(ctx, repository)
	if err != nil {
		return domain.Commit{}, err
	}
	var response struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name string `json:"name"`
			} `json:"author"`
		} `json:"commit"`
	}
	if err := p.request(ctx, http.MethodGet, "/repos/"+repository.FullName+"/commits/"+url.PathEscape(sha), token, nil, &response); err != nil {
		return domain.Commit{}, err
	}
	return domain.Commit{SHA: response.SHA, Message: response.Commit.Message, Author: response.Commit.Author.Name}, nil
}

func (p *Provider) GetChangedFiles(ctx context.Context, repoID, before, after string) ([]string, error) {
	repository, err := p.GetRepository(ctx, repoID)
	if err != nil {
		return nil, err
	}
	token, err := p.installationTokenForRepository(ctx, repository)
	if err != nil {
		return nil, err
	}
	var response struct {
		Files []struct {
			Filename string `json:"filename"`
		} `json:"files"`
	}
	compare := url.PathEscape(before) + "..." + url.PathEscape(after)
	if err := p.request(ctx, http.MethodGet, "/repos/"+repository.FullName+"/compare/"+compare, token, nil, &response); err != nil {
		return nil, err
	}
	files := make([]string, 0, len(response.Files))
	for _, file := range response.Files {
		files = append(files, file.Filename)
	}
	return files, nil
}

func (p *Provider) CreateCloneCredential(ctx context.Context, repoID string) (domain.CloneCredential, error) {
	repository, err := p.GetRepository(ctx, repoID)
	if err != nil {
		return domain.CloneCredential{}, err
	}
	token, expiresAt, err := p.installationTokenForRepositoryWithExpiry(ctx, repository)
	if err != nil {
		return domain.CloneCredential{}, err
	}
	return domain.CloneCredential{Username: "x-access-token", Secret: token, ExpiresAt: expiresAt}, nil
}

func (p *Provider) installationTokenForRepository(ctx context.Context, repository domain.Repository) (string, error) {
	token, _, err := p.installationTokenForRepositoryWithExpiry(ctx, repository)
	return token, err
}

func (p *Provider) installationTokenForRepositoryWithExpiry(ctx context.Context, repository domain.Repository) (string, time.Time, error) {
	jwt, err := p.appJWT()
	if err != nil {
		return "", time.Time{}, err
	}
	installationID := p.InstallationID
	if installationID == "" {
		return "", time.Time{}, errors.New("github installation id is required")
	}
	var response struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := p.request(ctx, http.MethodPost, "/app/installations/"+url.PathEscape(installationID)+"/access_tokens", jwt, nil, &response); err != nil {
		return "", time.Time{}, err
	}
	return response.Token, response.ExpiresAt, nil
}

func (p *Provider) installationToken(ctx context.Context, installationID string) (string, error) {
	jwt, err := p.appJWT()
	if err != nil {
		return "", err
	}
	if installationID == "" {
		installationID = p.InstallationID
	}
	if installationID == "" {
		return "", errors.New("github installation id is required")
	}
	var response struct {
		Token string `json:"token"`
	}
	if err := p.request(ctx, http.MethodPost, "/app/installations/"+url.PathEscape(installationID)+"/access_tokens", jwt, nil, &response); err != nil {
		return "", err
	}
	return response.Token, nil
}

func (p *Provider) appJWT() (string, error) {
	if p.AppID == 0 || p.PrivateKey == nil {
		return "", errors.New("github app credentials are not configured")
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims, _ := json.Marshal(map[string]any{"iat": time.Now().Add(-time.Minute).Unix(), "exp": time.Now().Add(9 * time.Minute).Unix(), "iss": p.AppID})
	body := base64.RawURLEncoding.EncodeToString(claims)
	unsigned := header + "." + body
	hash := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, p.PrivateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func parsePrivateKey(raw []byte) (*rsa.PrivateKey, error) {
	raw = []byte(strings.ReplaceAll(string(raw), `\n`, "\n"))
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("github private key is not valid PEM")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("github private key is not an RSA key")
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("github private key is not an RSA key")
	}
	return rsaKey, nil
}

func (p *Provider) request(ctx context.Context, method, path, token string, body []byte, result any) error {
	requestBody := io.Reader(nil)
	if body != nil {
		requestBody = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.APIURL+path, requestBody)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return &APIError{StatusCode: response.StatusCode, Status: response.Status, Message: strings.TrimSpace(string(message))}
	}
	if result == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(result)
}

type repositoryResponse struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
}

func (r repositoryResponse) domain() domain.Repository {
	return domain.Repository{ID: strconv.FormatInt(r.ID, 10), Owner: r.Owner.Login, Name: r.Name, FullName: r.FullName, DefaultBranch: r.DefaultBranch}
}

func verifySignature(body []byte, signature string, secret []byte) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	expected := hmac.New(sha256.New, secret)
	_, _ = expected.Write(body)
	provided, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	return err == nil && hmac.Equal(expected.Sum(nil), provided)
}
