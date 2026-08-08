package core

import (
	"aether/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (c *Core) ImageRetentionDefault() int {
	n := c.Cfg.ImageRetention
	if n <= 0 {
		return 5
	}
	return n
}

func (c *Core) imageRetentionFor(app *domain.App) int {
	if app.ImageRetention > 0 {
		return app.ImageRetention
	}
	return c.ImageRetentionDefault()
}

func (c *Core) registryClient() (*domain.RegistrySettings, *http.Client, error) {
	cfg, err := c.Store.GetRegistrySettings()
	if err != nil || !cfg.Enabled || cfg.Status != "running" {
		return nil, nil, fmt.Errorf("registry interno indisponível")
	}
	return &cfg, &http.Client{Timeout: 10 * time.Second}, nil
}

// appImageTags lista os números de deploy de imagens próprias do app no registry interno.
// appImageRepo normaliza o nome da imagem Docker para o app
// (docker exige repository name em minúsculas).
func appImageRepo(app *domain.App) string {
	return "aether.local/" + strings.ToLower(app.Name)
}

func (c *Core) appImageTags(app *domain.App) ([]int, error) {
	cfg, client, err := c.registryClient()
	if err != nil {
		return nil, err
	}
	repo := appImageRepo(app)
	resp, err := client.Get("http://" + cfg.Host + ":" + strconv.Itoa(cfg.Port) + "/v2/" + repo + "/tags/list")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var tags struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}
	var out []int
	for _, t := range tags.Tags {
		if n, err := strconv.Atoi(t); err == nil {
			out = append(out, n)
		}
	}
	sort.Ints(out)
	return out, nil
}

func (c *Core) removeLocalImage(imageRef string) error {
	bin, err := exec.LookPath("podman")
	if err != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "rmi", "-f", imageRef)
	out, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(out), "No such image") {
		return err
	}
	return nil
}

// cleanupAppImages remove imagens antigas do app mantendo as N mais recentes + a em uso.
func (c *Core) cleanupAppImages(ctx context.Context, app *domain.App) error {
	retention := c.imageRetentionFor(app)
	if retention <= 0 {
		return nil
	}
	tags, err := c.appImageTags(app)
	if err != nil {
		if strings.Contains(err.Error(), "registry interno indisponível") {
			return nil
		}
		return err
	}
	if len(tags) <= retention {
		return nil
	}
	// manter: as N mais recentes + a do container ativo
	keep := map[int]bool{}
	for i := len(tags) - retention; i < len(tags); i++ {
		keep[tags[i]] = true
	}
	if deploys, err := c.Store.ListDeployments(app.ID, 1); err == nil && len(deploys) > 0 && deploys[0].ContainerID != "" {
		if deploys[0].Status == domain.DeploymentReady {
			keep[int(deploys[0].Number)] = true
		}
	}
	cfg, client, err := c.registryClient()
	if err != nil {
		return nil
	}
	removed := 0
	for _, n := range tags {
		if keep[n] {
			continue
		}
		repo := appImageRepo(app)
		if err := c.registryDeleteTag(client, cfg, repo, strconv.Itoa(n)); err == nil {
			_ = c.removeLocalImage(repo + ":" + strconv.Itoa(n))
			removed++
		}
	}
	if removed > 0 {
		c.notify.emit(app.OrgID, "system.images_cleanup", "Image retention cleaned "+strconv.Itoa(removed)+" image(s) for "+app.Name, map[string]any{
			"app_id": app.ID, "app_name": app.Name, "removed": removed,
		})
	}
	return nil
}

func (c *Core) registryDeleteTag(client *http.Client, cfg *domain.RegistrySettings, repo, tag string) error {
	url := "http://" + cfg.Host + ":" + strconv.Itoa(cfg.Port) + "/v2/" + repo + "/manifests/" + tag
	req, _ := http.NewRequest("HEAD", url, nil)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	digest := resp.Header.Get("Docker-Content-Digest")
	resp.Body.Close()
	if digest == "" {
		return fmt.Errorf("digest vazio para %s:%s", repo, tag)
	}
	req, _ = http.NewRequest("DELETE", url, nil)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json")
	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Core) RunImageRetention(ctx context.Context) error {
	orgs, err := c.Store.ListOrgs()
	if err != nil {
		return err
	}
	for _, org := range orgs {
		apps, err := c.Store.ListApps(org.ID)
		if err != nil {
			continue
		}
		for _, app := range apps {
			if app.SourceType != domain.SourceGit {
				continue
			}
			_ = c.cleanupAppImages(ctx, &app)
		}
	}
	return nil
}

func (c *Core) StartImageRetentionLoop(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.withLockSkip("lock:image-cleanup", lockCleanupTTL, func() {
					_ = c.RunImageRetention(context.Background())
				})
			}
		}
	}()
}

// RemoveAppImages apaga TODAS as imagens próprias do app (registry + local) — usado no delete.
func (c *Core) RemoveAppImages(ctx context.Context, app *domain.App) error {
	tags, err := c.appImageTags(app)
	if err != nil {
		if strings.Contains(err.Error(), "registry interno indisponível") {
			return nil
		}
		return err
	}
	cfg, client, err := c.registryClient()
	if err != nil {
		return nil
	}
	for _, n := range tags {
		repo := appImageRepo(app)
		if err := c.registryDeleteTag(client, cfg, repo, strconv.Itoa(n)); err == nil {
			_ = c.removeLocalImage(repo + ":" + strconv.Itoa(n))
		}
	}
	return nil
}
