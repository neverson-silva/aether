package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"aether/internal/domain"
)

type ComposeApp struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	ComposeYAML string    `json:"compose"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type composeFile struct {
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	Image   string            `yaml:"image"`
	Build   string            `yaml:"build"`
	Command string            `yaml:"command"`
	Ports   []string          `yaml:"ports"`
	Env     map[string]string `yaml:"environment"`
	Volumes []string          `yaml:"volumes"`
	Restart string            `yaml:"restart"`
}

func (c *Core) SaveCompose(orgID, projectID, name, content string) (*ComposeApp, error) {
	var cf composeFile
	if err := yaml.Unmarshal([]byte(content), &cf); err != nil {
		return nil, fmt.Errorf("compose inválido: %w", err)
	}
	if len(cf.Services) == 0 {
		return nil, fmt.Errorf("compose sem serviços")
	}
	now := time.Now().UTC()
	enc, err := c.Secrets.EncryptString(content)
	if err != nil {
		return nil, err
	}
	ca := &ComposeApp{ID: domain.NewID(), OrgID: orgID, ProjectID: projectID, Name: name, ComposeYAML: content, Status: "stopped", CreatedAt: now}
	if _, err := c.DB.Exec(`INSERT INTO compose_apps(id,org_id,project_id,name,compose,status,created_at) VALUES(?,?,?,?,?,?,?)`,
		ca.ID, ca.OrgID, ca.ProjectID, ca.Name, enc, ca.Status, now.UTC().Format(time.RFC3339)); err != nil {
		return nil, err
	}
	return ca, nil
}

func (c *Core) GetCompose(id string) (*ComposeApp, error) {
	var ca ComposeApp
	var created, encrypted string
	err := c.DB.QueryRow(`SELECT id,org_id,project_id,name,compose,status,created_at FROM compose_apps WHERE id=?`, id).Scan(
		&ca.ID, &ca.OrgID, &ca.ProjectID, &ca.Name, &encrypted, &ca.Status, &created)
	if err != nil {
		return nil, err
	}
	plain, derr := c.Secrets.DecryptString(encrypted)
	if derr != nil {
		return nil, derr
	}
	ca.ComposeYAML = plain
	ca.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &ca, nil
}

func (c *Core) ListCompose(orgID string) ([]ComposeApp, error) {
	rows, err := c.DB.Query(`SELECT id,org_id,project_id,name,compose,status,created_at FROM compose_apps WHERE org_id=? ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ComposeApp
	for rows.Next() {
		var ca ComposeApp
		var created, encrypted string
		if err := rows.Scan(&ca.ID, &ca.OrgID, &ca.ProjectID, &ca.Name, &encrypted, &ca.Status, &created); err != nil {
			return nil, err
		}
		plain, derr := c.Secrets.DecryptString(encrypted)
		if derr != nil {
			return nil, derr
		}
		ca.ComposeYAML = plain
		ca.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, ca)
	}
	return out, rows.Err()
}

func (c *Core) ComposeUp(id string) error {
	ca, err := c.GetCompose(id)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	dir := filepath.Join(c.Cfg.BuildsDir, "composeapps", ca.Name)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	file := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(file, []byte(ca.ComposeYAML), 0o640); err != nil {
		return err
	}
	// Runtime OCI-genérico (podman ou docker): `podman compose up -d`.
	cmd := exec.CommandContext(ctx, c.Driver.Name(), "compose", "-f", file, "up", "-d")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s compose up: %w\n%s", c.Driver.Name(), err, strings.TrimSpace(string(out)))
	}
	_, err = c.DB.Exec(`UPDATE compose_apps SET status='running' WHERE id=?`, id)
	return err
}

func (c *Core) ComposeDown(id string) error {
	ca, err := c.GetCompose(id)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	dir := filepath.Join(c.Cfg.BuildsDir, "composeapps", ca.Name)
	file := filepath.Join(dir, "docker-compose.yml")
	if _, err := os.Stat(file); err == nil {
		_ = exec.CommandContext(ctx, c.Driver.Name(), "compose", "-f", file, "down", "-v").Run()
	}
	_, err = c.DB.Exec(`UPDATE compose_apps SET status='stopped' WHERE id=?`, id)
	return err
}

func (c *Core) DeleteCompose(id string) error {
	if err := c.ComposeDown(id); err != nil {
		return err
	}
	_, err := c.DB.Exec(`DELETE FROM compose_apps WHERE id=?`, id)
	return err
}

func mapToEnv(m map[string]string) []string {
	var out []string
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}

type ComposeValidation struct {
	Valid      bool                 `json:"valid"`
	Services   []ComposeServiceInfo `json:"services"`
	Volumes    []string             `json:"volumes"`
	Networks   []string             `json:"networks"`
	Errors     []string             `json:"errors"`
	Warnings   []string             `json:"warnings"`
	DependsOn  map[string][]string  `json:"depends_on"`
	TotalPorts int                  `json:"total_ports"`
}

type ComposeServiceInfo struct {
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Build   string   `json:"build"`
	Ports   []string `json:"ports"`
	Volumes []string `json:"volumes"`
	Restart string   `json:"restart"`
}

func ValidateComposeYAML(content string) ComposeValidation {
	out := ComposeValidation{Errors: []string{}, Warnings: []string{}, Services: []ComposeServiceInfo{}, Volumes: []string{}, Networks: []string{}}
	out.DependsOn = map[string][]string{}
	var cf struct {
		Services map[string]struct {
			Image     string            `yaml:"image"`
			Build     any               `yaml:"build"`
			Command   string            `yaml:"command"`
			Ports     []string          `yaml:"ports"`
			Env       map[string]string `yaml:"environment"`
			Volumes   []string          `yaml:"volumes"`
			Restart   string            `yaml:"restart"`
			DependsOn any               `yaml:"depends_on"`
		} `yaml:"services"`
		Volumes  map[string]any `yaml:"volumes"`
		Networks map[string]any `yaml:"networks"`
	}
	if err := yaml.Unmarshal([]byte(content), &cf); err != nil {
		out.Valid = false
		out.Errors = append(out.Errors, "YAML parse error: "+err.Error())
		return out
	}
	if len(cf.Services) == 0 {
		out.Errors = append(out.Errors, "no services defined")
		out.Valid = false
		return out
	}
	volumes := map[string]bool{}
	for name := range cf.Volumes {
		volumes[name] = true
		out.Volumes = append(out.Volumes, name)
	}
	for name := range cf.Networks {
		out.Networks = append(out.Networks, name)
	}
	for name, svc := range cf.Services {
		info := ComposeServiceInfo{Name: name, Image: svc.Image, Ports: svc.Ports, Volumes: svc.Volumes, Restart: svc.Restart}
		if svc.Build != nil {
			info.Build = fmt.Sprintf("%v", svc.Build)
		}
		out.Services = append(out.Services, info)
		for _, v := range svc.Volumes {
			if strings.HasPrefix(v, "./") || strings.HasPrefix(v, "/") || strings.HasPrefix(v, "~") {
				continue
			}
			if !strings.Contains(v, ":") && !volumes[v] {
				out.Warnings = append(out.Warnings, fmt.Sprintf("service %s: volume %q não declarado em volumes:", name, v))
			}
		}
		out.TotalPorts += len(svc.Ports)
	}
	if len(out.Errors) == 0 {
		out.Valid = true
	}
	return out
}
