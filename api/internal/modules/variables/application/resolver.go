package application

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"

	appsdomain "aether/internal/modules/apps/domain"
	"aether/internal/modules/variables/domain"
)

type Resolver struct {
	Vars   domain.Store
	Apps   ServiceAppStore
	Cipher domain.SecretCipher
}

type ServiceAppStore interface {
	GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error)
	ListEnvVars(ctx context.Context, appID uuid.UUID) ([]appsdomain.EnvVar, error)
}

type ServiceScopeStore interface {
	GetServiceScope(ctx context.Context, serviceID, orgID uuid.UUID) (uuid.UUID, *uuid.UUID, error)
	ListEnvVarsByService(ctx context.Context, serviceID uuid.UUID) ([]appsdomain.EnvVar, error)
}

func (r *Resolver) Effective(ctx context.Context, appID, orgID uuid.UUID) (map[string]string, error) {
	resolved, err := r.Resolved(ctx, appID, orgID)
	if err != nil {
		return nil, err
	}
	kv := make(map[string]string, len(resolved))
	for _, v := range resolved {
		kv[v.Key] = v.Value
	}
	return kv, nil
}

type scopeEntry struct {
	Key    string
	Value  string
	Source string
	Secret bool
}

func (r *Resolver) Resolved(ctx context.Context, appID, orgID uuid.UUID) ([]domain.ResolvedVariable, error) {
	app, appErr := r.Apps.GetApp(ctx, appID, orgID)
	projectID := uuid.Nil
	var environmentID *uuid.UUID
	serviceVars := []appsdomain.EnvVar(nil)
	if appErr == nil {
		projectID = app.ProjectID
		environmentID = app.EnvironmentID
		serviceVars, appErr = r.Apps.ListEnvVars(ctx, appID)
	} else if serviceStore, ok := r.Apps.(ServiceScopeStore); ok {
		projectID, environmentID, appErr = serviceStore.GetServiceScope(ctx, appID, orgID)
		if appErr == nil {
			serviceVars, appErr = serviceStore.ListEnvVarsByService(ctx, appID)
		}
	}
	if appErr != nil {
		return nil, appErr
	}

	scopes := newVarScopes(map[string]string{})
	merged := map[string]scopeEntry{}

	project, err := r.Vars.ListVariables(ctx, projectID, uuid.Nil)
	if err != nil {
		return nil, err
	}
	for _, v := range project {
		val := r.decrypt(v)
		scopes.project[v.Key] = val
		merged[v.Key] = scopeEntry{Key: v.Key, Value: val, Source: "project", Secret: v.IsSecret}
	}

	if environmentID != nil {
		env, err := r.Vars.ListVariables(ctx, projectID, *environmentID)
		if err != nil {
			return nil, err
		}
		for _, v := range env {
			val := r.decrypt(v)
			scopes.environment[v.Key] = val
			merged[v.Key] = scopeEntry{Key: v.Key, Value: val, Source: "environment", Secret: v.IsSecret}
		}
	}

	for _, v := range serviceVars {
		val := r.decryptEnvVar(v)
		scopes.service[v.Name] = val
		merged[v.Name] = scopeEntry{Key: v.Name, Value: val, Source: "service", Secret: v.Secret}
	}

	values := make(map[string]string, len(merged))
	for _, e := range merged {
		values[e.Key] = e.Value
	}
	scopes.merged = values

	resolved := make(map[string]string, len(merged))
	for key := range merged {
		val, err := scopes.resolve(key)
		if err != nil {
			return nil, err
		}
		resolved[key] = val
	}

	out := make([]domain.ResolvedVariable, 0, len(merged))
	for _, e := range merged {
		out = append(out, domain.ResolvedVariable{Key: e.Key, Value: resolved[e.Key], Source: e.Source, Secret: e.Secret})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

func (r *Resolver) decrypt(v domain.Variable) string {
	if !v.IsSecret || v.Value == "" || r.Cipher == nil {
		return v.Value
	}
	plain, err := r.Cipher.Decrypt(v.Value)
	if err != nil {
		return v.Value
	}
	return plain
}

func (r *Resolver) decryptEnvVar(v appsdomain.EnvVar) string {
	if !v.Secret || v.Value == "" || r.Cipher == nil {
		return v.Value
	}
	plain, err := r.Cipher.Decrypt(v.Value)
	if err != nil {
		return v.Value
	}
	return plain
}

var placeholderRe = regexp.MustCompile(`\$\{\{?([A-Za-z_][A-Za-z0-9_.]*)\}?\}`)

type varScopes struct {
	merged      map[string]string
	project     map[string]string
	environment map[string]string
	service     map[string]string
}

func newVarScopes(merged map[string]string) *varScopes {
	return &varScopes{
		merged: merged, project: map[string]string{},
		environment: map[string]string{}, service: map[string]string{},
	}
}

func (s *varScopes) lookup(scope, name string) (string, bool) {
	switch scope {
	case "project":
		v, ok := s.project[name]
		return v, ok
	case "environment":
		v, ok := s.environment[name]
		return v, ok
	case "service":
		v, ok := s.service[name]
		return v, ok
	default:
		v, ok := s.merged[name]
		return v, ok
	}
}

func splitScope(name string) (string, string) {
	if i := strings.Index(name, "."); i > 0 {
		prefix := name[:i]
		switch prefix {
		case "project", "environment", "service":
			return prefix, name[i+1:]
		}
	}
	return "", name
}

// resolve expande placeholders no valor efetivo de uma chave, com resolução
func (s *varScopes) resolve(key string) (string, error) {
	return s.resolveKey("merged", key, map[string]bool{})
}

func (s *varScopes) resolveKey(scope, name string, resolving map[string]bool) (string, error) {
	ckey := scope + "\x00" + name
	if resolving[ckey] {
		return "", fmt.Errorf("circular variable reference: %s", name)
	}
	val, ok := s.lookup(scope, name)
	if !ok {
		return "", nil
	}
	matches := placeholderRe.FindAllStringSubmatchIndex(val, -1)
	if len(matches) == 0 {
		return val, nil
	}
	resolving[ckey] = true
	defer delete(resolving, ckey)

	var sb strings.Builder
	last := 0
	for _, m := range matches {
		sb.WriteString(val[last:m[0]])
		full := val[m[0]:m[1]]
		refName := val[m[2]:m[3]]
		refScope, ref := splitScope(refName)
		if _, ok := s.lookup(refScope, ref); !ok {
			sb.WriteString(full)
		} else {
			rv, err := s.resolveKey(refScope, ref, resolving)
			if err != nil {
				return "", err
			}
			sb.WriteString(rv)
		}
		last = m[1]
	}
	sb.WriteString(val[last:])
	return sb.String(), nil
}
