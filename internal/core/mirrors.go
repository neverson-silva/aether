package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"aether/internal/domain"
)

func (c *Core) CreateMirror(name, source, dest string, destTLSVerify bool, tagsFilter, schedule string) (*domain.RegistryMirror, error) {
	m := &domain.RegistryMirror{
		ID:            "mir-" + domain.NewID(),
		Name:          name,
		Source:        source,
		Dest:          dest,
		DestTLSVerify: destTLSVerify,
		TagsFilter:    tagsFilter,
		Schedule:      schedule,
		Status:        "idle",
		CreatedAt:     time.Now().UTC(),
	}
	if err := c.Store.CreateMirror(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *Core) RunMirror(ctx context.Context, m *domain.RegistryMirror) error {
	skopeo, err := exec.LookPath("skopeo")
	if err != nil {
		return fmt.Errorf("skopeo não encontrado")
	}
	if err := c.Store.UpdateMirrorStatus(m.ID, "running", time.Now().UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	defer c.Store.UpdateMirrorStatus(m.ID, "idle", time.Now().UTC().Format(time.RFC3339))
	repos, err := c.registryCatalog(ctx, m.Source)
	if err != nil {
		return err
	}
	copied := 0
	for _, repo := range repos {
		tags := c.registryTags(ctx, m.Source, repo)
		for _, tag := range tags {
			if m.TagsFilter != "" && !strings.Contains(tag, m.TagsFilter) {
				continue
			}
			src := m.Source + "/" + repo + ":" + tag
			dst := m.Dest + "/" + repo + ":" + tag
			cmd := exec.CommandContext(ctx, skopeo, "copy",
				"--src-tls-verify=false",
				fmt.Sprintf("--dest-tls-verify=%t", m.DestTLSVerify),
				"docker://"+src, "docker://"+dst)
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("[mirror] %s -> %s: %v (%s)", src, dst, err, strings.TrimSpace(string(out)))
				continue
			}
			copied++
		}
	}
	return nil
}

func (c *Core) registryCatalog(ctx context.Context, base string) ([]string, error) {
	var cat struct {
		Repositories []string `json:"repositories"`
	}
	if err := c.getJSON(ctx, base+"/v2/_catalog", &cat); err != nil {
		return nil, err
	}
	return cat.Repositories, nil
}

func (c *Core) registryTags(ctx context.Context, base, repo string) []string {
	var tags struct {
		Tags []string `json:"tags"`
	}
	if err := c.getJSON(ctx, base+"/v2/"+repo+"/tags/list", &tags); err != nil {
		return nil
	}
	return tags.Tags
}

func (c *Core) getJSON(ctx context.Context, url string, out any) error {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("GET %s: %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
