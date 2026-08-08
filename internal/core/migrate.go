package core

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type MigrateResult struct {
	Platform string       `json:"platform"`
	Services []MigrateSvc `json:"services"`
}

type MigrateSvc struct {
	Dir        string            `json:"dir"`
	Name       string            `json:"name"`
	Compose    string            `json:"compose"`
	Env        map[string]string `json:"env"`
	Secrets    []string          `json:"secrets"`
	SourceType string            `json:"source_type"`
}

func DiscoverCoolify(dir string) (*MigrateResult, error) {
	return discover(dir, "coolify")
}

func DiscoverDokploy(dir string) (*MigrateResult, error) {
	return discover(dir, "dokploy")
}

func discover(root, platform string) (*MigrateResult, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	res := &MigrateResult{Platform: platform}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub := filepath.Join(root, e.Name())
		composePath := ""
		if _, err := os.Stat(filepath.Join(sub, "docker-compose.yml")); err == nil {
			composePath = filepath.Join(sub, "docker-compose.yml")
		} else if _, err := os.Stat(filepath.Join(sub, "docker-compose.yaml")); err == nil {
			composePath = filepath.Join(sub, "docker-compose.yaml")
		} else if _, err := os.Stat(filepath.Join(sub, "compose.yml")); err == nil {
			composePath = filepath.Join(sub, "compose.yml")
		}
		if composePath == "" {
			continue
		}
		raw, err := os.ReadFile(composePath)
		if err != nil {
			continue
		}
		svc := MigrateSvc{
			Dir:        sub,
			Name:       e.Name(),
			Compose:    string(raw),
			Env:        map[string]string{},
			SourceType: "compose",
		}
		if _, err := os.Stat(filepath.Join(sub, ".env")); err == nil {
			f, err := os.Open(filepath.Join(sub, ".env"))
			if err == nil {
				sc := bufio.NewScanner(f)
				for sc.Scan() {
					line := strings.TrimSpace(sc.Text())
					if line == "" || strings.HasPrefix(line, "#") {
						continue
					}
					parts := strings.SplitN(line, "=", 2)
					if len(parts) != 2 {
						continue
					}
					k := strings.TrimSpace(parts[0])
					v := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
					svc.Env[k] = v
					lower := strings.ToLower(k)
					if strings.Contains(lower, "key") || strings.Contains(lower, "secret") || strings.Contains(lower, "token") || strings.Contains(lower, "password") {
						svc.Secrets = append(svc.Secrets, k)
					}
				}
				f.Close()
			}
		}
		res.Services = append(res.Services, svc)
	}
	sort.Slice(res.Services, func(i, j int) bool { return res.Services[i].Name < res.Services[j].Name })
	return res, nil
}

func (c *Core) ImportDiscovered(orgID string, res *MigrateResult) (int, error) {
	count := 0
	for _, svc := range res.Services {
		name := sanitizeComposeName(svc.Name)
		if _, err := c.SaveCompose(orgID, "", name, svc.Compose); err != nil {
			return count, err
		}
		for k, v := range svc.Env {
			secret := containsAny(k, svc.Secrets)
			_ = c.Store.SetEnvVar("cmp:"+name, k, v, secret)
		}
		count++
	}
	return count, nil
}

func sanitizeComposeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return strings.ToLower(b.String())
}

func containsAny(s string, list []string) bool {
	for _, l := range list {
		if s == l {
			return true
		}
	}
	return false
}
