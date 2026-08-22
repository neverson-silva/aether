package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	appsdomain "aether/internal/modules/apps/domain"
)

type fakeProviderAppStore struct {
	app *appsdomain.App
	err error
}

func (f fakeProviderAppStore) GetAppByID(ctx context.Context, id uuid.UUID) (*appsdomain.App, error) {
	return f.app, f.err
}

type fakeDeployer struct {
	called  int
	trigger string
	err     error
}

func (f *fakeDeployer) Deploy(ctx context.Context, appID, orgID uuid.UUID, trigger, commit string) (any, error) {
	f.called++
	f.trigger = trigger
	if f.err != nil {
		return nil, f.err
	}
	return map[string]any{"status": "queued"}, nil
}

type fakeCipher struct{}

func (fakeCipher) Encrypt(plain string) (string, error) { return plain, nil }
func (fakeCipher) Decrypt(ciphertext string) (string, error) {
	return ciphertext, nil
}

func githubPayload(branch string) []byte {
	return []byte(`{"ref":"refs/heads/` + branch + `","repository":{"full_name":"org/repo"}}`)
}

func signGitHub(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func newProviderHooks(branch, secret string, deployErr error) (*ProviderHooks, *fakeDeployer) {
	appID := uuid.New()
	app := &appsdomain.App{
		ID: appID, OrgID: uuid.New(), SourceType: "git",
		GitURL: "https://github.com/org/repo", GitBranch: branch, WebhookSecret: secret,
	}
	deployer := &fakeDeployer{err: deployErr}
	return &ProviderHooks{
		Apps:      fakeProviderAppStore{app: app},
		Deployer:  deployer,
		Passwords: fakeCipher{},
	}, deployer
}

func TestGitHubDeploy(t *testing.T) {
	secret := "s3cr3t"
	hooks, deployer := newProviderHooks("main", secret, nil)
	payload := githubPayload("main")
	status, result := hooks.GitHub(context.Background(), uuid.New(), strings.NewReader(string(payload)), signGitHub(payload, secret))
	if status != http.StatusAccepted {
		t.Fatalf("status: %d (%+v)", status, result)
	}
	if deployer.called != 1 || deployer.trigger != "webhook" {
		t.Fatalf("deploy: %+v", deployer)
	}
}

func TestGitHubWrongBranch(t *testing.T) {
	hooks, deployer := newProviderHooks("main", "s3cr3t", nil)
	status, _ := hooks.GitHub(context.Background(), uuid.New(), strings.NewReader(string(githubPayload("dev"))), "")
	if status != http.StatusOK {
		t.Fatalf("status: %d", status)
	}
	if deployer.called != 0 {
		t.Fatalf("não deveria deployar")
	}
}

func TestGitHubBadSignature(t *testing.T) {
	hooks, deployer := newProviderHooks("main", "s3cr3t", nil)
	payload := githubPayload("main")
	status, _ := hooks.GitHub(context.Background(), uuid.New(), strings.NewReader(string(payload)), "sha256=bad")
	if status != http.StatusUnauthorized {
		t.Fatalf("status: %d", status)
	}
	if deployer.called != 0 {
		t.Fatalf("não deveria deployar")
	}
}

func TestGitHubNotFound(t *testing.T) {
	hooks, _ := newProviderHooks("main", "s3cr3t", nil)
	hooks.Apps = fakeProviderAppStore{err: appNotFound{}}
	status, _ := hooks.GitHub(context.Background(), uuid.New(), strings.NewReader(`{}`), "")
	if status != http.StatusNotFound {
		t.Fatalf("status: %d", status)
	}
}

func TestGitLabPush(t *testing.T) {
	hooks, deployer := newProviderHooks("main", "tok3n", nil)
	payload := `{"object_kind":"push","ref":"refs/heads/main"}`
	status, _ := hooks.GitLab(context.Background(), uuid.New(), strings.NewReader(payload), "tok3n")
	if status != http.StatusAccepted {
		t.Fatalf("status: %d", status)
	}
	if deployer.trigger != "gitlab" {
		t.Fatalf("trigger: %s", deployer.trigger)
	}
}

func TestGitLabBadToken(t *testing.T) {
	hooks, deployer := newProviderHooks("main", "tok3n", nil)
	status, _ := hooks.GitLab(context.Background(), uuid.New(), strings.NewReader(`{}`), "wrong")
	if status != http.StatusUnauthorized {
		t.Fatalf("status: %d", status)
	}
	if deployer.called != 0 {
		t.Fatalf("não deveria deployar")
	}
}

func TestBitbucketPush(t *testing.T) {
	hooks, deployer := newProviderHooks("main", "s3cr3t", nil)
	payload := `{"push":{"changes":[{"new":{"name":"main","type":"branch"}}]}}`
	status, _ := hooks.Bitbucket(context.Background(), uuid.New(), strings.NewReader(payload), "")
	if status != http.StatusAccepted {
		t.Fatalf("status: %d", status)
	}
	if deployer.trigger != "bitbucket" {
		t.Fatalf("trigger: %s", deployer.trigger)
	}
}

type appNotFound struct{}

func (appNotFound) Error() string { return "not found" }
