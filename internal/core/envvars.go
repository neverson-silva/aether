package core

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"aether/internal/domain"
)

var (
	envKeyRe  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	interpRe  = regexp.MustCompile(`\$\{\{\s*(?:environment|project)\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)
	envLineRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)
)

type envCache struct {
	mu   sync.Mutex
	vars map[string][]domain.EnvironmentVariable
	proj map[string][]domain.ProjectVariable
}

func (c *Core) cachedEnvVars(environmentID string) []domain.EnvironmentVariable {
	c.envCache.mu.Lock()
	defer c.envCache.mu.Unlock()
	if c.envCache.vars == nil {
		c.envCache.vars = map[string][]domain.EnvironmentVariable{}
	}
	if vars, ok := c.envCache.vars[environmentID]; ok {
		return vars
	}
	vars, err := c.Store.ListEnvVariables(environmentID)
	if err != nil {
		return nil
	}
	c.envCache.vars[environmentID] = vars
	return vars
}

func (c *Core) cachedProjectVars(projectID string) []domain.ProjectVariable {
	c.envCache.mu.Lock()
	defer c.envCache.mu.Unlock()
	if c.envCache.proj == nil {
		c.envCache.proj = map[string][]domain.ProjectVariable{}
	}
	if vars, ok := c.envCache.proj[projectID]; ok {
		return vars
	}
	vars, err := c.Store.ListProjectVariables(projectID)
	if err != nil {
		return nil
	}
	c.envCache.proj[projectID] = vars
	return vars
}

func (c *Core) invalidateProjectCache(projectID string) {
	c.envCache.mu.Lock()
	if c.envCache.proj != nil {
		delete(c.envCache.proj, projectID)
	}
	c.envCache.mu.Unlock()
}

func (c *Core) invalidateEnvCache(environmentID string) {
	c.envCache.mu.Lock()
	if c.envCache.vars != nil {
		delete(c.envCache.vars, environmentID)
	}
	c.envCache.mu.Unlock()
}

func (c *Core) decryptVar(v *domain.EnvironmentVariable) string {
	if !v.IsSecret {
		return v.Value
	}
	dec, err := c.Secrets.DecryptString(v.Value)
	if err != nil {
		return ""
	}
	return dec
}

func (c *Core) ListEnvironmentVars(projectID, environmentID string, includeSecrets bool) ([]domain.EnvironmentVariable, error) {
	env, err := c.Store.GetEnvironment(environmentID)
	if err != nil || env.ProjectID != projectID {
		return nil, ErrEnvNotFound
	}
	vars := c.cachedEnvVars(environmentID)
	out := make([]domain.EnvironmentVariable, 0, len(vars))
	for _, v := range vars {
		vv := v
		if vv.IsSecret && !includeSecrets {
			vv.Value = "••••••••••"
		} else if vv.IsSecret {
			vv.Value = c.decryptVar(&vv)
		}
		out = append(out, vv)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

func (c *Core) auditVar(projectID, environmentID, action, key, previous string, wasSecret bool) {
	// Auditoria nunca persiste valores secretos (nem criptografados): registra
	// apenas quem/quando/ação/escopo.
	if wasSecret {
		previous = ""
	}
	_ = c.Store.AddVariableAudit(&domain.VariableAudit{
		ID:            "va-" + domain.NewID(),
		ProjectID:     projectID,
		EnvironmentID: environmentID,
		Action:        action,
		Key:           key,
		PreviousValue: previous,
		CreatedAt:     time.Now().UTC(),
	})
}

func (c *Core) SetEnvironmentVar(projectID, environmentID, key, value string, secret bool) (*domain.EnvironmentVariable, error) {
	env, err := c.Store.GetEnvironment(environmentID)
	if err != nil || env.ProjectID != projectID {
		return nil, ErrEnvNotFound
	}
	if !envKeyRe.MatchString(key) {
		return nil, fmt.Errorf("invalid variable key: %s", key)
	}
	now := time.Now().UTC()
	stored := value
	if secret {
		enc, err := c.Secrets.EncryptString(value)
		if err != nil {
			return nil, err
		}
		stored = enc
	}
	var previous string
	wasSecret := false
	if existing, err := c.Store.GetEnvVariable(environmentID, key); err == nil {
		previous = existing.Value
		wasSecret = existing.IsSecret
	}
	v := &domain.EnvironmentVariable{
		ID:            "ev-" + domain.NewID(),
		ProjectID:     projectID,
		EnvironmentID: environmentID,
		Key:           key,
		Value:         stored,
		IsSecret:      secret,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := c.Store.UpsertEnvVariable(v); err != nil {
		return nil, err
	}
	c.auditVar(projectID, environmentID, "set", key, previous, wasSecret)
	c.invalidateEnvCache(environmentID)
	return v, nil
}

func (c *Core) DeleteEnvironmentVar(projectID, environmentID, key string) error {
	env, err := c.Store.GetEnvironment(environmentID)
	if err != nil || env.ProjectID != projectID {
		return ErrEnvNotFound
	}
	previous := ""
	wasSecret := false
	if existing, err := c.Store.GetEnvVariable(environmentID, key); err == nil {
		previous = existing.Value
		wasSecret = existing.IsSecret
	}
	if err := c.Store.DeleteEnvVariable(environmentID, key); err != nil {
		return err
	}
	c.auditVar(projectID, environmentID, "delete", key, previous, wasSecret)
	c.invalidateEnvCache(environmentID)
	return nil
}

func (c *Core) ReplaceEnvironmentVars(projectID, environmentID string, entries map[string]struct {
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}) (int, error) {
	env, err := c.Store.GetEnvironment(environmentID)
	if err != nil || env.ProjectID != projectID {
		return 0, ErrEnvNotFound
	}
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	now := time.Now().UTC()
	vars := make([]domain.EnvironmentVariable, 0, len(keys))
	for _, k := range keys {
		if !envKeyRe.MatchString(k) {
			return 0, fmt.Errorf("invalid variable key: %s", k)
		}
		entry := entries[k]
		stored := entry.Value
		if entry.Secret {
			enc, err := c.Secrets.EncryptString(entry.Value)
			if err != nil {
				return 0, err
			}
			stored = enc
		}
		vars = append(vars, domain.EnvironmentVariable{
			ID:            "ev-" + domain.NewID(),
			ProjectID:     projectID,
			EnvironmentID: environmentID,
			Key:           k,
			Value:         stored,
			IsSecret:      entry.Secret,
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}
	if err := c.Store.ReplaceEnvVariables(environmentID, projectID, vars); err != nil {
		return 0, err
	}
	c.auditVar(projectID, environmentID, "bulk_replace", "", "", false)
	c.invalidateEnvCache(environmentID)
	return len(vars), nil
}

func ParseEnvText(text string) (map[string]struct {
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}, error) {
	out := map[string]struct {
		Value  string `json:"value"`
		Secret bool   `json:"secret"`
	}{}
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := envLineRe.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("linha %d inválida: %s", i+1, line)
		}
		key, value := m[1], m[2]
		if _, dup := out[key]; dup {
			return nil, fmt.Errorf("chave duplicada na linha %d: %s", i+1, key)
		}
		value = strings.Trim(value, `"'`)
		out[key] = struct {
			Value  string `json:"value"`
			Secret bool   `json:"secret"`
		}{Value: value, Secret: false}
	}
	return out, nil
}

func EnvTextOf(vars []domain.EnvironmentVariable, resolve func(*domain.EnvironmentVariable) string) string {
	var b strings.Builder
	for _, v := range vars {
		val := v.Value
		if resolve != nil {
			val = resolve(&v)
		}
		b.WriteString(v.Key + "=" + val + "\n")
	}
	return b.String()
}

func ResolveInterpolations(env []string, lookup map[string]string) []string {
	out := make([]string, 0, len(env))
	for _, line := range env {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			out = append(out, line)
			continue
		}
		key, value := parts[0], parts[1]
		value = interpRe.ReplaceAllStringFunc(value, func(m string) string {
			sm := interpRe.FindStringSubmatch(m)
			if len(sm) == 2 {
				if v, ok := lookup[sm[1]]; ok {
					return v
				}
			}
			return ""
		})
		out = append(out, key+"="+value)
	}
	return out
}

func (c *Core) decryptProjectVar(v *domain.ProjectVariable) string {
	if !v.IsSecret {
		return v.Value
	}
	dec, err := c.Secrets.DecryptString(v.Value)
	if err != nil {
		return ""
	}
	return dec
}

func (c *Core) ListProjectVars(projectID string, includeSecrets bool) ([]domain.ProjectVariable, error) {
	vars := c.cachedProjectVars(projectID)
	out := make([]domain.ProjectVariable, 0, len(vars))
	for _, v := range vars {
		vv := v
		if vv.IsSecret && !includeSecrets {
			vv.Value = "••••••••••"
		} else if vv.IsSecret {
			vv.Value = c.decryptProjectVar(&vv)
		}
		out = append(out, vv)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

func (c *Core) auditProjectVar(projectID, action, key, previous string) {
	_ = c.Store.AddVariableAudit(&domain.VariableAudit{
		ID:            "va-" + domain.NewID(),
		ProjectID:     projectID,
		Action:        action,
		Key:           key,
		PreviousValue: previous,
		CreatedAt:     time.Now().UTC(),
	})
}

func (c *Core) SetProjectVar(projectID, key, value string, secret bool) (*domain.ProjectVariable, error) {
	if !envKeyRe.MatchString(key) {
		return nil, fmt.Errorf("invalid variable key: %s", key)
	}
	now := time.Now().UTC()
	stored := value
	if secret {
		enc, err := c.Secrets.EncryptString(value)
		if err != nil {
			return nil, err
		}
		stored = enc
	}
	var previous string
	if existing, err := c.Store.GetProjectVariable(projectID, key); err == nil {
		previous = existing.Value
	}
	v := &domain.ProjectVariable{
		ID:        "pv-" + domain.NewID(),
		ProjectID: projectID,
		Key:       key,
		Value:     stored,
		IsSecret:  secret,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := c.Store.UpsertProjectVariable(v); err != nil {
		return nil, err
	}
	c.auditProjectVar(projectID, "set", key, previous)
	c.invalidateProjectCache(projectID)
	return v, nil
}

func (c *Core) DeleteProjectVar(projectID, key string) error {
	previous := ""
	if existing, err := c.Store.GetProjectVariable(projectID, key); err == nil {
		previous = existing.Value
	}
	if err := c.Store.DeleteProjectVariable(projectID, key); err != nil {
		return err
	}
	c.auditProjectVar(projectID, "delete", key, previous)
	c.invalidateProjectCache(projectID)
	return nil
}

func (c *Core) ReplaceProjectVars(projectID string, entries map[string]struct {
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}) (int, error) {
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	now := time.Now().UTC()
	vars := make([]domain.ProjectVariable, 0, len(keys))
	for _, k := range keys {
		if !envKeyRe.MatchString(k) {
			return 0, fmt.Errorf("invalid variable key: %s", k)
		}
		entry := entries[k]
		stored := entry.Value
		if entry.Secret {
			enc, err := c.Secrets.EncryptString(entry.Value)
			if err != nil {
				return 0, err
			}
			stored = enc
		}
		vars = append(vars, domain.ProjectVariable{
			ID:        "pv-" + domain.NewID(),
			ProjectID: projectID,
			Key:       k,
			Value:     stored,
			IsSecret:  entry.Secret,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	if err := c.Store.ReplaceProjectVariables(projectID, vars); err != nil {
		return 0, err
	}
	c.auditProjectVar(projectID, "bulk_replace", "", "")
	c.invalidateProjectCache(projectID)
	return len(vars), nil
}
