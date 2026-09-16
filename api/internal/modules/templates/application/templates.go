package application

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	appsdomain "aether/internal/modules/apps/domain"
	"aether/internal/modules/templates/domain"
	variablesDomain "aether/internal/modules/variables/domain"
	composeengine "aether/internal/platform/compose"
)

type Templates struct {
	Store                    domain.Store
	Apps                     AppStore
	Catalog                  RemoteCatalog
	Variables                variablesDomain.Store
	ProvisionTemplateDomains func(context.Context, uuid.UUID, *domain.ComposeApp, []domain.TemplateDomain) error
	TemplateDomainGenerator  func(string) string
	IngressNetwork           string
	DataDir                  string
}

var errCatalogUnavailable = errors.New("template catalog unavailable")

type RemoteCatalog interface {
	List(ctx context.Context) ([]domain.Template, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Template, error)
}

type RemoteCatalogLogo interface {
	Logo(ctx context.Context, id uuid.UUID) ([]byte, string, error)
}

func (t *Templates) Logo(ctx context.Context, id uuid.UUID) ([]byte, string, error) {
	provider, ok := t.Catalog.(RemoteCatalogLogo)
	if !ok {
		return nil, "", domain.ErrNotFound
	}
	return provider.Logo(ctx, id)
}

func (t *Templates) List(ctx context.Context, filter domain.Filter) ([]domain.Template, error) {
	if t.Catalog == nil {
		return nil, errCatalogUnavailable
	}
	remote, err := t.Catalog.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Template, 0, len(remote))
	for _, template := range remote {
		if filter.Category != "" && !strings.EqualFold(filter.Category, template.Category) {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(template.Name+" "+template.Description), strings.ToLower(filter.Search)) {
			continue
		}
		out = append(out, template)
	}
	return out, nil
}

func (t *Templates) Install(ctx context.Context, templateID, orgID, projectID uuid.UUID, name string, overrides map[string]string) (*domain.Template, error) {
	if t.Catalog == nil {
		return nil, errCatalogUnavailable
	}
	tpl, err := t.Catalog.Get(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if _, err := t.Apps.GetProject(ctx, projectID, orgID); err != nil {
		return nil, err
	}
	var environmentID *uuid.UUID
	if defaults, ok := t.Apps.(interface {
		DefaultEnvironment(context.Context, uuid.UUID) (uuid.UUID, error)
	}); ok {
		defaultID, err := defaults.DefaultEnvironment(ctx, projectID)
		if err != nil && !errors.Is(err, appsdomain.ErrNotFound) {
			return nil, fmt.Errorf("resolve default environment: %w", err)
		}
		if err == nil {
			environmentID = &defaultID
		}
	}
	appName := strings.TrimSpace(name)
	if appName == "" {
		appName = tpl.Name
	}
	if err := validateName(appName); err != nil {
		return nil, domain.ErrValidation
	}
	compose := strings.TrimSpace(tpl.ComposeYAML)
	if compose == "" {
		compose, err = composeYAML(tpl.Definition, overrides)
		if err != nil {
			return nil, domain.ErrValidation
		}
	}
	compose, err = composeengine.NormalizeNamedResourceDefinitions(compose)
	if err != nil {
		return nil, domain.ErrValidation
	}
	compose, err = composeengine.NormalizeTemplateCompose(compose)
	if err != nil {
		return nil, domain.ErrValidation
	}
	port, hasPort, err := composePortSelection(compose)
	if err != nil {
		return nil, domain.ErrValidation
	}
	if !hasPort {
		port, err = composeContainerPort(compose)
		if err != nil {
			return nil, domain.ErrValidation
		}
	}
	configuredVariables, resolvedDomains, resolvedMounts := t.resolveTemplateConfig(tpl, appName, overrides)
	configuredEnvironment := mergeTemplateEnvironmentVariables(templateEnvironmentVariables(compose), configuredVariables)
	compose, err = injectComposeSecurityDefaults(compose)
	if err != nil {
		return nil, fmt.Errorf("%w: apply template security defaults: %v", domain.ErrValidation, err)
	}
	if err := composeengine.ValidatePolicy(compose); err != nil {
		return nil, fmt.Errorf("%w: template Compose is not supported: %v", domain.ErrValidation, err)
	}
	if _, serviceVariables := t.Apps.(ServiceVariableStore); !serviceVariables {
		if err := t.ensureTemplateVariables(ctx, projectID, compose, overrides, configuredEnvironment); err != nil {
			return nil, err
		}
	}
	created, err := t.Store.CreateComposeApp(ctx, &domain.ComposeApp{
		OrgID: orgID, ProjectID: projectID, EnvironmentID: environmentID, Name: appName, Compose: compose, Port: port, Status: "stopped",
	})
	if err != nil {
		return nil, err
	}
	cleanupCreated := func(cause error) error {
		if cleanupErr := t.Store.DeleteComposeApp(ctx, created.ID, orgID); cleanupErr != nil {
			return fmt.Errorf("%w: cleanup failed: %v", cause, cleanupErr)
		}
		return cause
	}
	updatedCompose := created.Compose
	if len(resolvedMounts) > 0 {
		mountApp := *created
		mountApp.Compose = updatedCompose
		updatedCompose, err = t.materializeTemplateMounts(&mountApp, resolvedMounts)
		if err != nil {
			return nil, cleanupCreated(err)
		}
	}
	if len(resolvedDomains) > 0 {
		ingressApp := *created
		ingressApp.Compose = updatedCompose
		updatedCompose, err = t.materializeTemplateIngress(&ingressApp, resolvedDomains)
		if err != nil {
			return nil, cleanupCreated(err)
		}
	}
	if updatedCompose != created.Compose {
		if err := composeengine.ValidatePolicy(updatedCompose); err != nil {
			return nil, cleanupCreated(fmt.Errorf("%w: generated template Compose is not supported: %v", domain.ErrValidation, err))
		}
		if err := t.Store.UpdateComposeApp(ctx, created.ID, updatedCompose, created.Port); err != nil {
			return nil, cleanupCreated(fmt.Errorf("save template runtime configuration: %w", err))
		}
		created.Compose = updatedCompose
		compose = updatedCompose
	}
	if err := t.ensureServiceVariables(ctx, created, compose, overrides, configuredEnvironment); err != nil {
		return nil, cleanupCreated(err)
	}
	if t.ProvisionTemplateDomains != nil && len(resolvedDomains) > 0 {
		if err := t.ProvisionTemplateDomains(ctx, orgID, created, resolvedDomains); err != nil {
			return nil, cleanupCreated(fmt.Errorf("provision template domains: %w", err))
		}
	}
	tpl.Installs++
	tpl.ComposeYAML = compose
	return tpl, nil
}

func (t *Templates) ensureServiceVariables(ctx context.Context, app *domain.ComposeApp, compose string, overrides map[string]string, configured []templateEnvironmentVariable) error {
	writer, ok := t.Apps.(ServiceVariableStore)
	if !ok || app == nil || app.ServiceID == uuid.Nil {
		return nil
	}
	existing, err := writer.ListServiceEnvVars(ctx, app.ServiceID)
	if err != nil {
		return fmt.Errorf("list service variables: %w", err)
	}
	known := make(map[string]struct{}, len(existing))
	for _, variable := range existing {
		known[variable.Name] = struct{}{}
	}
	projectValues, err := t.resolvedProjectValues(ctx, app.ProjectID)
	if err != nil {
		return err
	}
	for _, variable := range mergeTemplateEnvironmentVariables(templateEnvironmentVariables(compose), configured) {
		if _, exists := known[variable.Name]; exists {
			continue
		}
		value := strings.TrimSpace(overrides[variable.Name])
		if value == "" {
			value = strings.TrimSpace(projectValues[variable.Name])
		}
		if value == "" {
			value = strings.TrimSpace(variable.Value)
		}
		if value == "" && isGeneratedTemplateSecret(variable.Name) {
			value, err = generatedTemplateSecret()
			if err != nil {
				return fmt.Errorf("generate service secret: %w", err)
			}
		}
		if value == "" {
			continue
		}
		if err := writer.UpsertServiceEnvVar(ctx, app.ServiceID, variable.Name, value, isGeneratedTemplateSecret(variable.Name)); err != nil {
			return fmt.Errorf("save service variable %s: %w", variable.Name, err)
		}
	}
	return nil
}

func (t *Templates) resolveTemplateConfig(tpl *domain.Template, appName string, overrides map[string]string) ([]templateEnvironmentVariable, []domain.TemplateDomain, []domain.TemplateMount) {
	if tpl == nil {
		return nil, nil, nil
	}
	definitions := make(map[string]string, len(tpl.Variables))
	for _, variable := range tpl.Variables {
		definitions[variable.Name] = variable.Value
	}
	for _, variable := range tpl.Environment {
		if _, exists := definitions[variable.Name]; !exists {
			definitions[variable.Name] = variable.Value
		}
	}
	resolved := make(map[string]string, len(definitions))
	resolving := make(map[string]bool, len(definitions))
	helperCache := make(map[string]string)
	var resolveVariable func(string) string
	resolveVariable = func(name string) string {
		if value, ok := resolved[name]; ok {
			return value
		}
		if resolving[name] {
			return ""
		}
		raw, ok := definitions[name]
		if !ok {
			return ""
		}
		resolving[name] = true
		value := resolveTemplateString(raw, name, definitions, resolveVariable, t.TemplateDomainGenerator, helperCache)
		delete(resolving, name)
		resolved[name] = value
		return value
	}
	configured := make([]templateEnvironmentVariable, 0, len(tpl.Environment))
	for _, variable := range tpl.Environment {
		value := resolveTemplateString(variable.Value, variable.Name, definitions, resolveVariable, t.TemplateDomainGenerator, helperCache)
		if override, ok := overrides[variable.Name]; ok && strings.TrimSpace(override) != "" {
			value = resolveTemplateString(override, variable.Name, definitions, resolveVariable, t.TemplateDomainGenerator, helperCache)
		}
		configured = append(configured, templateEnvironmentVariable{Name: variable.Name, Value: value})
	}
	domains := make([]domain.TemplateDomain, 0, len(tpl.Domains))
	for _, templateDomain := range tpl.Domains {
		resolvedDomain := templateDomain
		seed := appName + "-" + templateDomain.ServiceName + "-" + strconv.Itoa(templateDomain.Port)
		resolvedDomain.Host = resolveTemplateString(templateDomain.Host, seed, definitions, resolveVariable, t.TemplateDomainGenerator, helperCache)
		resolvedDomain.Path = resolveTemplateString(templateDomain.Path, seed, definitions, resolveVariable, t.TemplateDomainGenerator, helperCache)
		if resolvedDomain.Path == "" {
			resolvedDomain.Path = "/"
		}
		domains = append(domains, resolvedDomain)
	}
	mounts := make([]domain.TemplateMount, 0, len(tpl.Mounts))
	for _, mount := range tpl.Mounts {
		resolvedMount := mount
		resolvedMount.FilePath = resolveTemplateString(mount.FilePath, appName, definitions, resolveVariable, t.TemplateDomainGenerator, helperCache)
		resolvedMount.Content = resolveTemplateString(mount.Content, mount.FilePath, definitions, resolveVariable, t.TemplateDomainGenerator, helperCache)
		mounts = append(mounts, resolvedMount)
	}
	return configured, domains, mounts
}

func resolveTemplateString(value, seed string, definitions map[string]string, resolveVariable func(string) string, domainGenerator func(string) string, helperCache map[string]string) string {
	for iteration := 0; iteration < 64; iteration++ {
		start := strings.Index(value, "${")
		if start < 0 {
			return value
		}
		end := strings.IndexByte(value[start+2:], '}')
		if end < 0 {
			return value
		}
		end += start + 2
		token := value[start+2 : end]
		replacement := ""
		if _, ok := definitions[token]; ok {
			replacement = resolveVariable(token)
		} else {
			replacement = resolveTemplateHelper(token, seed, definitions, resolveVariable, domainGenerator, helperCache)
		}
		value = value[:start] + replacement + value[end+1:]
	}
	return value
}

func resolveTemplateHelper(token, seed string, definitions map[string]string, resolveVariable func(string) string, domainGenerator func(string) string, cache map[string]string) string {
	parts := strings.Split(strings.TrimSpace(token), ":")
	name := strings.ToLower(parts[0])
	cacheKey := seed + ":" + token
	if value, ok := cache[cacheKey]; ok {
		return value
	}
	length := 32
	if len(parts) == 2 {
		if parsed, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && parsed > 0 && parsed <= 4096 {
			length = parsed
		}
	}
	var value string
	switch name {
	case "domain":
		if domainGenerator != nil {
			value = domainGenerator(seed)
		}
	case "password":
		value = generatedTemplateString(length)
	case "hash":
		digest := sha256.Sum256([]byte(generatedTemplateString(length)))
		value = hex.EncodeToString(digest[:])
		if len(value) > length {
			value = value[:length]
		}
	case "base64":
		randomBytes := make([]byte, length)
		if _, err := rand.Read(randomBytes); err != nil {
			value = base64.RawStdEncoding.EncodeToString([]byte(generatedTemplateString(length)))
		} else {
			value = base64.RawStdEncoding.EncodeToString(randomBytes)
		}
	case "uuid":
		value = uuid.New().String()
	case "randomport":
		value = strconv.Itoa(1024 + int(generatedTemplateByte())%64512)
	case "timestamp":
		value = templateTimestamp(parts, false)
	case "timestamps":
		value = templateTimestamp(parts, true)
	case "timestampms":
		value = templateTimestamp(parts, false)
	case "email":
		value = "user-" + generatedTemplateString(8) + "@example.com"
	case "username":
		value = "user" + strings.ToLower(generatedTemplateString(8))
	case "jwt":
		value = resolveTemplateJWT(parts, seed, definitions, resolveVariable, cache)
	default:
		return "${" + token + "}"
	}
	cache[cacheKey] = value
	return value
}

func templateTimestamp(parts []string, seconds bool) string {
	now := time.Now()
	if len(parts) > 1 {
		if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(strings.Join(parts[1:], ":"))); err == nil {
			now = parsed
		}
	}
	if seconds {
		return strconv.FormatInt(now.Unix(), 10)
	}
	return strconv.FormatInt(now.UnixMilli(), 10)
}

func resolveTemplateJWT(parts []string, seed string, definitions map[string]string, resolveVariable func(string) string, cache map[string]string) string {
	if len(parts) < 2 {
		return generatedTemplateString(32)
	}
	if length, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && length > 0 && length <= 4096 {
		return generatedTemplateString(length)
	}
	secret := resolveVariable(parts[1])
	if secret == "" {
		secret = definitions[parts[1]]
	}
	payload := `{"iat":` + strconv.FormatInt(time.Now().Unix(), 10) + `}`
	if len(parts) >= 3 {
		payload = resolveTemplateString(definitions[parts[2]], parts[2], definitions, resolveVariable, nil, cache)
		if strings.TrimSpace(payload) == "" {
			payload = `{"iat":` + strconv.FormatInt(time.Now().Unix(), 10) + `}`
		}
	}
	var payloadValue any
	if err := json.Unmarshal([]byte(payload), &payloadValue); err != nil {
		payloadValue = map[string]any{"value": payload}
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	encodedPayload, err := json.Marshal(payloadValue)
	if err != nil {
		return generatedTemplateString(32)
	}
	encodedPayloadString := base64.RawURLEncoding.EncodeToString(encodedPayload)
	unsigned := header + "." + encodedPayloadString
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(unsigned))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return unsigned + "." + signature
}

func generatedTemplateString(length int) string {
	value := make([]byte, length)
	encoded := ""
	if _, err := rand.Read(value); err == nil {
		encoded = hex.EncodeToString(value)
	} else {
		encoded = hex.EncodeToString([]byte(strconv.FormatInt(time.Now().UnixNano(), 10)))
	}
	for len(encoded) < length {
		encoded += encoded
	}
	return encoded[:minTemplateLength(length)]
}

func generatedTemplateByte() byte {
	value := []byte{0}
	_, _ = rand.Read(value)
	return value[0]
}

func minTemplateLength(length int) int {
	if length < 1 {
		return 1
	}
	return length
}

func mergeTemplateEnvironmentVariables(base, configured []templateEnvironmentVariable) []templateEnvironmentVariable {
	merged := make([]templateEnvironmentVariable, 0, len(base)+len(configured))
	positions := make(map[string]int, len(base)+len(configured))
	for _, variable := range base {
		if variable.Name == "" {
			continue
		}
		positions[variable.Name] = len(merged)
		merged = append(merged, variable)
	}
	for _, variable := range configured {
		if variable.Name == "" {
			continue
		}
		if index, ok := positions[variable.Name]; ok {
			merged[index] = variable
			continue
		}
		positions[variable.Name] = len(merged)
		merged = append(merged, variable)
	}
	return merged
}

func (t *Templates) materializeTemplateMounts(app *domain.ComposeApp, mounts []domain.TemplateMount) (string, error) {
	if app == nil {
		return "", errors.New("template mounts require a compose app")
	}
	if len(mounts) == 0 {
		return app.Compose, nil
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(app.Compose), &document); err != nil {
		return "", fmt.Errorf("parse template mounts compose: %w", err)
	}
	services, ok := document["services"].(map[string]any)
	if !ok || len(services) == 0 {
		return "", errors.New("template mounts compose has no services")
	}
	serviceNames := make([]string, 0, len(services))
	for name := range services {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)
	configs, ok := document["configs"].(map[string]any)
	if !ok {
		configs = make(map[string]any)
		document["configs"] = configs
	}
	for index, mount := range mounts {
		target, err := templateMountTarget(mount.FilePath)
		if err != nil {
			return "", err
		}
		serviceName := strings.TrimSpace(mount.ServiceName)
		if serviceName == "" {
			serviceName = serviceNames[0]
		}
		rawService, ok := services[serviceName]
		if !ok {
			return "", fmt.Errorf("template mount references unknown service %s", serviceName)
		}
		service, ok := rawService.(map[string]any)
		if !ok {
			return "", fmt.Errorf("template mount service %s is invalid", serviceName)
		}
		removeTemplateMountVolume(service, target)
		fileName := strconv.Itoa(index)
		configName := "aether-template-file-" + fileName
		configs[configName] = map[string]any{"content": mount.Content}
		entries := make([]any, 0)
		if existing, ok := service["configs"].([]any); ok {
			entries = append(entries, existing...)
		}
		entries = append(entries, map[string]any{"source": configName, "target": target})
		service["configs"] = entries
	}
	encoded, err := yaml.Marshal(document)
	if err != nil {
		return "", fmt.Errorf("encode template mounts compose: %w", err)
	}
	return string(encoded), nil
}

func removeTemplateMountVolume(service map[string]any, target string) {
	volumes, ok := service["volumes"].([]any)
	if !ok {
		return
	}
	filtered := make([]any, 0, len(volumes))
	for _, raw := range volumes {
		volumeTarget := ""
		switch value := raw.(type) {
		case string:
			parts := strings.Split(value, ":")
			if len(parts) > 1 {
				volumeTarget = parts[1]
			}
		case map[string]any:
			volumeTarget = strings.TrimSpace(fmt.Sprint(value["target"]))
		}
		if volumeTarget != target {
			filtered = append(filtered, raw)
		}
	}
	if len(filtered) == 0 {
		delete(service, "volumes")
		return
	}
	service["volumes"] = filtered
}

func (t *Templates) materializeTemplateIngress(app *domain.ComposeApp, domains []domain.TemplateDomain) (string, error) {
	if app == nil {
		return "", errors.New("template ingress requires a compose app")
	}
	if len(domains) == 0 || strings.TrimSpace(t.IngressNetwork) == "" {
		return app.Compose, nil
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(app.Compose), &document); err != nil {
		return "", fmt.Errorf("parse template ingress compose: %w", err)
	}
	services, ok := document["services"].(map[string]any)
	if !ok || len(services) == 0 {
		return "", errors.New("template ingress compose has no services")
	}
	networks, ok := document["networks"].(map[string]any)
	if !ok {
		networks = make(map[string]any)
		document["networks"] = networks
	}
	if _, exists := networks[t.IngressNetwork]; !exists {
		networks[t.IngressNetwork] = map[string]any{"name": t.IngressNetwork, "external": true}
	}
	for _, mapping := range domains {
		serviceName := strings.TrimSpace(mapping.ServiceName)
		service, exists := services[serviceName]
		if !exists {
			return "", fmt.Errorf("template domain references unknown service %s", serviceName)
		}
		serviceMap, ok := service.(map[string]any)
		if !ok {
			return "", fmt.Errorf("template domain service %s is invalid", serviceName)
		}
		ensureTemplateIngressNetwork(serviceMap, t.IngressNetwork, templateIngressAlias(app.ID, serviceName))
		if mapping.Publish {
			ensureTemplatePublishedPort(serviceMap, mapping.Port)
		}
	}
	encoded, err := yaml.Marshal(document)
	if err != nil {
		return "", fmt.Errorf("encode template ingress compose: %w", err)
	}
	return string(encoded), nil
}

func ensureTemplatePublishedPort(service map[string]any, port int) {
	if port < 1 || port > 65535 {
		return
	}
	value, ok := service["ports"]
	if !ok {
		service["ports"] = []any{fmt.Sprintf("%d:%d", port, port)}
		return
	}
	ports, ok := value.([]any)
	if !ok {
		return
	}
	for _, item := range ports {
		if strings.Contains(fmt.Sprint(item), ":"+strconv.Itoa(port)) {
			return
		}
	}
	service["ports"] = append(ports, fmt.Sprintf("%d:%d", port, port))
}

func ensureTemplateIngressNetwork(service map[string]any, networkName, alias string) {
	entry := map[string]any{"aliases": []any{alias}}
	current, ok := service["networks"]
	if !ok {
		service["networks"] = map[string]any{"default": map[string]any{}, networkName: entry}
		return
	}
	if values, ok := current.([]any); ok {
		mapped := make(map[string]any, len(values)+1)
		for _, value := range values {
			name := strings.TrimSpace(fmt.Sprint(value))
			if name != "" {
				mapped[name] = map[string]any{}
			}
		}
		mapped[networkName] = entry
		service["networks"] = mapped
		return
	}
	if mapped, ok := current.(map[string]any); ok {
		mapped[networkName] = entry
		return
	}
	service["networks"] = map[string]any{networkName: entry}
}

func templateIngressAlias(serviceID uuid.UUID, serviceName string) string {
	clean := strings.ToLower(strings.TrimSpace(serviceName))
	clean = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, clean)
	clean = strings.Trim(clean, "-")
	if clean == "" {
		clean = "service"
	}
	return "app-" + serviceID.String()[:8] + "-" + clean
}

func templateMountTarget(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\\\r\n\"`") {
		return "", errors.New("template mount path is invalid")
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", errors.New("template mount path escapes the container")
	}
	if !strings.HasPrefix(clean, "/") {
		clean = "/" + clean
	}
	return clean, nil
}

func (t *Templates) resolvedProjectValues(ctx context.Context, projectID uuid.UUID) (map[string]string, error) {
	if t.Variables == nil {
		return map[string]string{}, nil
	}
	var variables []variablesDomain.Variable
	if reader, ok := t.Variables.(ResolvedProjectVariableStore); ok {
		resolved, err := reader.ListResolvedVariables(ctx, projectID, uuid.Nil)
		if err != nil {
			return nil, fmt.Errorf("list resolved project variables: %w", err)
		}
		variables = resolved
	} else {
		listed, err := t.Variables.ListVariables(ctx, projectID, uuid.Nil)
		if err != nil {
			return nil, fmt.Errorf("list project variables: %w", err)
		}
		variables = listed
	}
	values := make(map[string]string, len(variables))
	for _, variable := range variables {
		if variable.Value != "" {
			values[variable.Key] = variable.Value
		}
	}
	return values, nil
}

func (t *Templates) ensureTemplateVariables(ctx context.Context, projectID uuid.UUID, compose string, overrides map[string]string, configured []templateEnvironmentVariable) error {
	if t.Variables == nil {
		return nil
	}
	existing, err := t.Variables.ListVariables(ctx, projectID, uuid.Nil)
	if err != nil {
		return fmt.Errorf("list template variables: %w", err)
	}
	current := make(map[string]variablesDomain.Variable, len(existing))
	for _, variable := range existing {
		current[variable.Key] = variable
	}
	for _, variable := range mergeTemplateEnvironmentVariables(templateEnvironmentVariables(compose), configured) {
		value := strings.TrimSpace(overrides[variable.Name])
		if value == "" {
			value = variable.Value
		}
		if value == "" && isGeneratedTemplateSecret(variable.Name) {
			value, err = generatedTemplateSecret()
			if err != nil {
				return fmt.Errorf("generate template secret: %w", err)
			}
		}
		if value == "" {
			continue
		}
		if previous, ok := current[variable.Name]; ok && strings.TrimSpace(previous.Value) != "" {
			continue
		}
		if _, err := t.Variables.UpsertVariable(ctx, &variablesDomain.Variable{ProjectID: projectID, Key: variable.Name, Value: value, IsSecret: isGeneratedTemplateSecret(variable.Name)}); err != nil {
			return fmt.Errorf("save template variable %s: %w", variable.Name, err)
		}
	}
	return nil
}

type templateEnvironmentVariable struct {
	Name  string
	Value string
}

func templateEnvironmentVariables(content string) []templateEnvironmentVariable {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return nil
	}
	root := &document
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	if root.Kind != yaml.MappingNode {
		return nil
	}
	services := yamlMappingValue(root, "services")
	if services == nil || services.Kind != yaml.MappingNode {
		return nil
	}
	seen := map[string]struct{}{}
	variables := make([]templateEnvironmentVariable, 0)
	for i := 0; i+1 < len(services.Content); i += 2 {
		environment := yamlMappingValue(services.Content[i+1], "environment")
		if environment == nil {
			continue
		}
		switch environment.Kind {
		case yaml.MappingNode:
			for j := 0; j+1 < len(environment.Content); j += 2 {
				name := strings.TrimSpace(environment.Content[j].Value)
				value := environment.Content[j+1].Value
				if strings.Contains(value, "${") {
					continue
				}
				appendTemplateEnvironmentVariable(&variables, seen, name, value)
			}
		case yaml.SequenceNode:
			for _, item := range environment.Content {
				name, value, found := strings.Cut(item.Value, "=")
				if !found {
					name, value = item.Value, ""
				}
				if strings.Contains(value, "${") {
					continue
				}
				appendTemplateEnvironmentVariable(&variables, seen, strings.TrimSpace(name), value)
			}
		}
	}
	return variables
}

func appendTemplateEnvironmentVariable(variables *[]templateEnvironmentVariable, seen map[string]struct{}, name, value string) {
	if name == "" {
		return
	}
	if _, exists := seen[name]; exists {
		return
	}
	seen[name] = struct{}{}
	*variables = append(*variables, templateEnvironmentVariable{Name: name, Value: value})
}

func yamlMappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

func isGeneratedTemplateSecret(name string) bool {
	lower := strings.ToLower(name)
	for _, part := range []string{"password", "passwd", "secret", "token", "api_key", "access_key", "private_key", "credential"} {
		if strings.Contains(lower, part) {
			return true
		}
	}
	return false
}

func generatedTemplateSecret() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func (t *Templates) ListCompose(ctx context.Context, orgID uuid.UUID) ([]domain.ComposeApp, error) {
	return t.Store.ListComposeAppsByOrg(ctx, orgID)
}

func (t *Templates) DeleteCompose(ctx context.Context, id, orgID uuid.UUID) error {
	return t.Store.DeleteComposeApp(ctx, id, orgID)
}

type serviceDef struct {
	Name     string            `json:"name"`
	Image    string            `json:"image"`
	Port     int               `json:"port"`
	Env      map[string]string `json:"env"`
	Volumes  []string          `json:"volumes"`
	Versions []string          `json:"versions"`
}

type templateDef struct {
	Services []serviceDef `json:"services"`
}

func composeYAML(definition string, overrides map[string]string) (string, error) {
	var def templateDef
	if err := json.Unmarshal([]byte(definition), &def); err != nil {
		return "", err
	}
	if len(def.Services) == 0 {
		return "", fmt.Errorf("template with no services")
	}
	compose := map[string]any{
		"version":  "3.8",
		"services": composeServices(def.Services, overrides),
	}
	raw, err := yaml.Marshal(compose)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func composeServices(services []serviceDef, overrides map[string]string) map[string]any {
	out := make(map[string]any, len(services))
	for _, svc := range services {
		name := svc.Name
		if name == "" {
			name = "app"
		}
		image := svc.Image
		if image == "" && len(svc.Versions) > 0 {
			image = svc.Versions[0]
		}
		if override, ok := overrides["image"]; ok && override != "" {
			image = override
		}
		entry := map[string]any{
			"image":      image,
			"restart":    "no",
			"mem_limit":  "512m",
			"cpus":       "1.0",
			"pids_limit": 256,
		}
		if svc.Port > 0 {
			entry["expose"] = []string{strconv.Itoa(svc.Port)}
		}
		if len(svc.Env) > 0 {
			environment := make(map[string]string, len(svc.Env))
			for key, value := range svc.Env {
				if override, ok := overrides[key]; ok {
					value = override
				}
				environment[key] = value
			}
			entry["environment"] = environment
		}
		if len(svc.Volumes) > 0 {
			entry["volumes"] = svc.Volumes
		}
		out[name] = entry
	}
	return out
}

func validateName(name string) error {
	if len(name) < 1 || len(name) > 64 {
		return domain.ErrValidation
	}
	return nil
}
