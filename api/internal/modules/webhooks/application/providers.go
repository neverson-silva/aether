package application

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	appsdomain "aether/internal/modules/apps/domain"
	"aether/internal/modules/webhooks/domain"
	"aether/internal/platform/git"
)

type ProviderHooks struct {
	Apps      ProviderAppStore
	Deployer  Deployer
	Passwords domain.PasswordCipher
	Logger    *slog.Logger
}

type ProviderAppStore interface {
	GetAppByID(ctx context.Context, id uuid.UUID) (*appsdomain.App, error)
}

type Deployer interface {
	Deploy(ctx context.Context, appID, orgID uuid.UUID, trigger, commit string) (any, error)
}

func (p *ProviderHooks) gitApp(ctx context.Context, appID uuid.UUID) (*appsdomain.App, int, any) {
	app, err := p.Apps.GetAppByID(ctx, appID)
	if err != nil {
		return nil, http.StatusNotFound, map[string]any{"error": "application not found"}
	}
	if app.SourceType != "git" || app.GitURL == "" {
		return nil, http.StatusBadRequest, map[string]any{"error": "application is not a git source"}
	}
	return app, 0, nil
}

func (p *ProviderHooks) GitHub(ctx context.Context, appID uuid.UUID, body io.Reader, signature string) (int, any) {
	if p.Logger != nil {
		p.Logger.Info("github webhook received", "app_id", appID)
	}
	app, status, payload := p.gitApp(ctx, appID)
	if app == nil {
		return status, payload
	}
	raw, err := io.ReadAll(io.LimitReader(body, 1<<20))
	if err != nil {
		return http.StatusBadRequest, map[string]any{"error": "invalid body"}
	}
	secret, err := p.secret(app)
	if err != nil {
		return http.StatusForbidden, map[string]any{"error": "webhook not configured"}
	}
	if signature != "" {
		if err := git.VerifyGitHubSignature(raw, signature, []byte(secret)); err != nil {
			return http.StatusUnauthorized, map[string]any{"error": err.Error()}
		}
	}
	event, err := git.ParsePushEvent(raw)
	if err != nil {
		return http.StatusBadRequest, map[string]any{"error": err.Error()}
	}
	if p.Logger != nil {
		p.Logger.Info("github webhook parsed", "app_id", app.ID, "app_name", app.Name, "branch", event.Branch())
	}
	if event.Branch() != app.GitBranch {
		return http.StatusOK, map[string]any{"status": "ignored", "reason": "branch diferente"}
	}
	deployment, err := p.Deployer.Deploy(ctx, app.ID, app.OrgID, "webhook", "")
	if err != nil {
		return http.StatusInternalServerError, map[string]any{"error": err.Error()}
	}
	return http.StatusAccepted, deployment
}

func (p *ProviderHooks) GitLab(ctx context.Context, appID uuid.UUID, body io.Reader, token string) (int, any) {
	app, status, payload := p.gitApp(ctx, appID)
	if app == nil {
		return status, payload
	}
	if token != "" {
		secret, err := p.secret(app)
		if err != nil {
			return http.StatusForbidden, map[string]any{"error": "webhook not configured"}
		}
		if token != secret {
			return http.StatusUnauthorized, map[string]any{"error": "invalid token"}
		}
	}
	raw, err := io.ReadAll(io.LimitReader(body, 1<<20))
	if err != nil {
		return http.StatusBadRequest, map[string]any{"error": "invalid body"}
	}
	if push, err := git.ParseGitLabPushEvent(raw); err == nil {
		if push.Branch() != app.GitBranch {
			return http.StatusOK, map[string]any{"status": "ignored"}
		}
		deployment, err := p.Deployer.Deploy(ctx, app.ID, app.OrgID, "gitlab", "")
		if err != nil {
			return http.StatusInternalServerError, map[string]any{"error": err.Error()}
		}
		return http.StatusAccepted, deployment
	}
	if mr, err := git.ParseGitLabMergeEvent(raw); err == nil {
		if mr.ObjectAttributes.Action == "close" || mr.ObjectAttributes.State == "closed" {
			return http.StatusOK, map[string]any{"status": "closed"}
		}
		if mr.ObjectAttributes.SourceBranch != "" {
			deployment, err := p.Deployer.Deploy(ctx, app.ID, app.OrgID, "gitlab-preview", "")
			if err != nil {
				return http.StatusInternalServerError, map[string]any{"error": err.Error()}
			}
			return http.StatusAccepted, deployment
		}
	}
	return http.StatusOK, map[string]any{"status": "ignored"}
}

func (p *ProviderHooks) Bitbucket(ctx context.Context, appID uuid.UUID, body io.Reader, signature string) (int, any) {
	app, status, payload := p.gitApp(ctx, appID)
	if app == nil {
		return status, payload
	}
	raw, err := io.ReadAll(io.LimitReader(body, 1<<20))
	if err != nil {
		return http.StatusBadRequest, map[string]any{"error": "invalid body"}
	}
	if signature != "" {
		secret, err := p.secret(app)
		if err != nil {
			return http.StatusForbidden, map[string]any{"error": "webhook not configured"}
		}
		if err := git.VerifyGitHubSignature(raw, signature, []byte(secret)); err != nil {
			return http.StatusUnauthorized, map[string]any{"error": err.Error()}
		}
	}
	if push, err := git.ParseBitbucketPushEvent(raw); err == nil {
		if push.Branch() != app.GitBranch {
			return http.StatusOK, map[string]any{"status": "ignored"}
		}
		deployment, err := p.Deployer.Deploy(ctx, app.ID, app.OrgID, "bitbucket", "")
		if err != nil {
			return http.StatusInternalServerError, map[string]any{"error": err.Error()}
		}
		return http.StatusAccepted, deployment
	}
	return http.StatusOK, map[string]any{"status": "ignored"}
}

func (p *ProviderHooks) secret(app *appsdomain.App) (string, error) {
	if app.WebhookSecret == "" {
		return "", errors.New("webhook not configured")
	}
	return p.Passwords.Decrypt(app.WebhookSecret)
}
