package core

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"aether/internal/domain"
	"aether/internal/runtime"
)

const registryImage = "registry:2"

func (c *Core) RegistrySettings() (domain.RegistrySettings, error) {
	return c.Store.GetRegistrySettings()
}

func (c *Core) RegistryEnable(ctx context.Context, enabled bool) (domain.RegistrySettings, error) {
	cfg, err := c.Store.GetRegistrySettings()
	if err != nil {
		return cfg, err
	}
	cfg.Enabled = enabled
	if enabled {
		cfg.Host = "127.0.0.1"
		cfg.Port = 5000
		if err := c.Driver.NetworkCreate(ctx, "aether-internal"); err != nil {
			if !strings.Contains(err.Error(), "exists") {
				return cfg, err
			}
		}
		cfg.Status = "starting"
		_ = c.Store.SaveRegistrySettings(&cfg)
		if cfg.ContainerID != "" {
			_ = c.Driver.Stop(ctx, cfg.ContainerID)
			_ = c.Driver.Remove(ctx, cfg.ContainerID, true)
		}
		hostPort := strconv.Itoa(cfg.Port)
		id, err := c.Driver.Run(ctx, runtime.RunSpec{
			Name:     "aether-registry",
			Image:    registryImage,
			Env:      []string{"REGISTRY_STORAGE_DELETE_ENABLED=true"},
			Ports:    []runtime.PortBinding{{HostPort: hostPort, ContainerPort: "5000"}},
			Volumes:  []runtime.VolumeMount{{Source: "aether-registry-data", Target: "/var/lib/registry"}},
			Networks: []string{"aether-internal"},
			Restart:  "always",
			Labels:   map[string]string{"aether.role": "registry"},
		})
		if err != nil {
			cfg.Status = "failed"
			_ = c.Store.SaveRegistrySettings(&cfg)
			return cfg, err
		}
		cfg.ContainerID = id
		cfg.Status = "running"
	} else {
		if cfg.ContainerID != "" {
			_ = c.Driver.Stop(ctx, cfg.ContainerID)
			_ = c.Driver.Remove(ctx, cfg.ContainerID, true)
			cfg.ContainerID = ""
		}
		cfg.Status = "stopped"
	}
	_ = c.Store.SaveRegistrySettings(&cfg)
	return cfg, nil
}

func (c *Core) RegistryPush(ctx context.Context, imageRef string) error {
	cfg, err := c.Store.GetRegistrySettings()
	if err != nil || !cfg.Enabled || cfg.Status != "running" {
		return fmt.Errorf("registry interno desabilitado")
	}
	skopeo, err := exec.LookPath("skopeo")
	if err != nil {
		return fmt.Errorf("skopeo não encontrado (instale com 'brew install skopeo' ou apt-get install skopeo)")
	}
	name := "aether.local/" + imageRef
	if strings.Contains(imageRef, ":") {
		name = "aether.local/" + imageRef
	} else {
		name = "aether.local/" + imageRef + ":latest"
	}
	cmd := exec.CommandContext(ctx, skopeo, "copy", "--override-os", "linux", "--dest-tls-verify=false", "docker://"+imageRef, "docker://"+cfg.Host+":"+strconv.Itoa(cfg.Port)+"/"+name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("skopeo copy: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (c *Core) RegistryImages(ctx context.Context) ([]domain.RegistryImage, error) {
	cfg, err := c.Store.GetRegistrySettings()
	if err != nil || !cfg.Enabled || cfg.Status != "running" {
		return nil, nil
	}
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	resp, err := client.Get("http://" + cfg.Host + ":" + strconv.Itoa(cfg.Port) + "/v2/_catalog")
	if err != nil {
		return nil, fmt.Errorf("catalog: %w", err)
	}
	defer resp.Body.Close()
	var cat struct {
		Repositories []string `json:"repositories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cat); err != nil {
		return nil, err
	}
	out := []domain.RegistryImage{}
	for _, repo := range cat.Repositories {
		img := domain.RegistryImage{Repo: strings.TrimPrefix(repo, "aether.local/")}
		tagsResp, err := client.Get("http://" + cfg.Host + ":" + strconv.Itoa(cfg.Port) + "/v2/" + repo + "/tags/list")
		if err != nil {
			continue
		}
		var tags struct {
			Tags []string `json:"tags"`
		}
		_ = json.NewDecoder(tagsResp.Body).Decode(&tags)
		tagsResp.Body.Close()
		img.Tags = tags.Tags
		for _, t := range img.Tags {
			size := c.registryImageSize(client, cfg, repo, t)
			if size > img.Size {
				img.Size = size
			}
		}
		out = append(out, img)
	}
	return out, nil
}

func (c *Core) registryImageSize(client *http.Client, cfg domain.RegistrySettings, repo, tag string) int64 {
	url := "http://" + cfg.Host + ":" + strconv.Itoa(cfg.Port) + "/v2/" + repo + "/manifests/" + tag
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json")
	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	var m struct {
		Config struct {
			Size int64 `json:"size"`
		} `json:"config"`
		Layers []struct {
			Size int64 `json:"size"`
		} `json:"layers"`
		Manifests []struct {
			Size int64 `json:"size"`
		} `json:"manifests"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		if cl := resp.Header.Get("Content-Length"); cl != "" {
			if n, err := strconv.ParseInt(cl, 10, 64); err == nil {
				return n
			}
		}
		return 0
	}
	size := m.Config.Size
	for _, l := range m.Layers {
		size += l.Size
	}
	for _, mm := range m.Manifests {
		size += mm.Size
	}
	return size
}

func (c *Core) RegistryDelete(ctx context.Context, repo, tag string) error {
	cfg, err := c.Store.GetRegistrySettings()
	if err != nil || !cfg.Enabled || cfg.Status != "running" {
		return fmt.Errorf("registry interno desabilitado")
	}
	client := &http.Client{Timeout: 10 * time.Second}
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
		return errors.New("manifest digest vazio")
	}
	req, _ = http.NewRequest("DELETE", url, nil)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json")
	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}
