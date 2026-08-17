package application

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"aether/internal/gitops/domain"
)

type GitOps struct {
	Store domain.Store
}

func (g *GitOps) Create(ctx context.Context, orgID uuid.UUID, name, repoURL, branch, path, applyMode string) (*domain.GitOps, error) {
	name = strings.TrimSpace(name)
	repoURL = strings.TrimSpace(repoURL)
	if name == "" || repoURL == "" {
		return nil, domain.ErrValidation
	}
	if branch == "" {
		branch = "main"
	}
	if path == "" {
		path = "aether.yml"
	}
	if applyMode == "" {
		applyMode = "manual"
	}
	if applyMode != "manual" && applyMode != "auto" {
		return nil, domain.ErrValidation
	}
	return g.Store.CreateGitOps(ctx, &domain.GitOps{
		OrgID: orgID, Name: name, RepoURL: repoURL, Branch: branch, Path: path,
		ApplyMode: applyMode, LastStatus: "pending",
	})
}

func (g *GitOps) List(ctx context.Context, orgID uuid.UUID) ([]domain.GitOps, error) {
	return g.Store.ListByOrg(ctx, orgID)
}

func (g *GitOps) Sync(ctx context.Context, id, orgID uuid.UUID, sha string, added, changed, removed int) (*domain.GitOps, error) {
	source, err := g.Store.GetGitOps(ctx, id)
	if err != nil {
		return nil, err
	}
	if source.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	if added < 0 || changed < 0 || removed < 0 {
		return nil, domain.ErrValidation
	}
	if err := g.Store.UpdateSync(ctx, id, domain.SyncResult{
		SHA: sha, Added: added, Changed: changed, Removed: removed, SyncedAt: now(),
	}); err != nil {
		return nil, err
	}
	return g.Store.GetGitOps(ctx, id)
}

func (g *GitOps) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	return g.Store.DeleteGitOps(ctx, id, orgID)
}

func now() time.Time {
	return time.Now().UTC()
}
