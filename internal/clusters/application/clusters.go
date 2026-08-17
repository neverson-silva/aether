package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/uuid"

	"aether/internal/clusters/domain"
)

type Clusters struct {
	Store domain.Store
}

func (c *Clusters) CreateCluster(ctx context.Context, orgID uuid.UUID, name string, labels []string) (*domain.Cluster, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrValidation
	}
	if labels == nil {
		labels = []string{}
	}
	return c.Store.CreateCluster(ctx, &domain.Cluster{OrgID: orgID, Name: name, Labels: labels})
}

func (c *Clusters) ListClusters(ctx context.Context, orgID uuid.UUID) ([]domain.Cluster, error) {
	return c.Store.ListClustersByOrg(ctx, orgID)
}

func (c *Clusters) DeleteCluster(ctx context.Context, clusterID, orgID uuid.UUID) error {
	cluster, err := c.Store.GetCluster(ctx, clusterID)
	if err != nil {
		return err
	}
	if cluster.OrgID != orgID {
		return domain.ErrNotFound
	}
	return c.Store.DeleteCluster(ctx, clusterID, orgID)
}

func (c *Clusters) AddServer(ctx context.Context, clusterID, orgID, serverID uuid.UUID) error {
	cluster, err := c.Store.GetCluster(ctx, clusterID)
	if err != nil {
		return err
	}
	if cluster.OrgID != orgID {
		return domain.ErrNotFound
	}
	if _, err := c.Store.GetServer(ctx, serverID); err != nil {
		return err
	}
	return c.Store.SetServerCluster(ctx, serverID, &clusterID)
}

func (c *Clusters) RemoveServer(ctx context.Context, clusterID, orgID, serverID uuid.UUID) error {
	cluster, err := c.Store.GetCluster(ctx, clusterID)
	if err != nil {
		return err
	}
	if cluster.OrgID != orgID {
		return domain.ErrNotFound
	}
	return c.Store.SetServerCluster(ctx, serverID, nil)
}

func (c *Clusters) ListServers(ctx context.Context) ([]domain.Server, error) {
	return c.Store.ListServers(ctx)
}

func (c *Clusters) DeleteServer(ctx context.Context, serverID uuid.UUID) error {
	return c.Store.DeleteServer(ctx, serverID)
}

func (c *Clusters) AgentToken(ctx context.Context) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := "aether-agent_" + base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	if err := c.Store.CreateServerToken(ctx, hex.EncodeToString(sum[:])); err != nil {
		return "", err
	}
	return token, nil
}

func (c *Clusters) GetRegistry(ctx context.Context) (*domain.Registry, error) {
	registry, err := c.Store.GetRegistry(ctx)
	if err == nil {
		return registry, nil
	}
	if err == domain.ErrNotFound {
		return &domain.Registry{Host: "127.0.0.1", Port: 5000, Status: "stopped"}, nil
	}
	return nil, err
}

func (c *Clusters) SetRegistryEnabled(ctx context.Context, enabled bool) (*domain.Registry, error) {
	return c.Store.SetRegistryEnabled(ctx, enabled)
}

func (c *Clusters) RegistryImages(ctx context.Context) ([]domain.RegistryImage, error) {
	out, err := exec.CommandContext(ctx, "podman", "images", "--format", "json").CombinedOutput()
	if err != nil {
		return nil, err
	}
	var images []struct {
		Names   []string `json:"Names"`
		ID      string   `json:"Id"`
		Size    int64    `json:"Size"`
		Created int64    `json:"Created"`
	}
	if err := json.Unmarshal(out, &images); err != nil {
		return nil, err
	}
	result := make([]domain.RegistryImage, 0, len(images))
	for _, img := range images {
		repo, tag := "none", "latest"
		if len(img.Names) > 0 {
			repo, tag = splitImage(img.Names[0])
		}
		result = append(result, domain.RegistryImage{
			Repo: repo, Tag: tag, ID: img.ID[:12],
			Size: img.Size, Created: img.Created,
		})
	}
	return result, nil
}

func (c *Clusters) RegistryDelete(ctx context.Context, repo, tag string) error {
	out, err := exec.CommandContext(ctx, "podman", "rmi", repo+":"+tag).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func splitImage(name string) (repo, tag string) {
	if idx := strings.LastIndex(name, ":"); idx >= 0 && !strings.Contains(name[idx:], "/") {
		return name[:idx], name[idx+1:]
	}
	return name, "latest"
}
