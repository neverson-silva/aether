package compose

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Project struct {
	Directory string
	File      string
	EnvFile   string
	Name      string
}

type Executor interface {
	Execute(context.Context, Project, ...string) (string, error)
}

type StreamingExecutor interface {
	ExecuteWithLogs(context.Context, Project, func(string), ...string) (string, error)
}

type Docker struct {
	Binary           string
	Host             string
	PublishedNetwork string
}

type externalNetworkAttachment struct {
	Service string
	Network string
	Aliases []string
}

func NewDocker(host string) *Docker {
	return &Docker{Binary: "docker", Host: host}
}

func NewDockerWithPublishedNetwork(host, publishedNetwork string) *Docker {
	return &Docker{Binary: "docker", Host: host, PublishedNetwork: publishedNetwork}
}

func (d *Docker) Execute(ctx context.Context, project Project, args ...string) (string, error) {
	return d.execute(ctx, project, nil, args...)
}

func (d *Docker) ExecuteWithLogs(ctx context.Context, project Project, sink func(string), args ...string) (string, error) {
	return d.execute(ctx, project, sink, args...)
}

func (d *Docker) execute(ctx context.Context, project Project, sink func(string), args ...string) (string, error) {
	if strings.TrimSpace(project.Directory) == "" || strings.TrimSpace(project.File) == "" {
		return "", fmt.Errorf("compose project is incomplete")
	}
	content, err := os.ReadFile(project.File)
	if err != nil {
		return "", fmt.Errorf("read compose configuration: %w", err)
	}
	prepared, changed, err := inlineManagedTemplateConfigs(string(content), project.Directory)
	if err != nil {
		return "", fmt.Errorf("materialize managed template configs: %w", err)
	}
	normalized, normalizeErr := normalizeUnsupportedExposeRanges(prepared)
	if normalizeErr != nil {
		return "", fmt.Errorf("normalize exposed port ranges: %w", normalizeErr)
	}
	if normalized != prepared {
		prepared = normalized
		changed = true
	}
	if d.PublishedNetwork != "" {
		withPublishedNetwork, networkChanged, networkErr := injectPublishedNetwork(prepared, d.PublishedNetwork)
		if networkErr != nil {
			return "", fmt.Errorf("inject published network: %w", networkErr)
		}
		if networkChanged {
			prepared = withPublishedNetwork
			changed = true
		}
	}
	if changed {
		overlay, err := os.CreateTemp(project.Directory, ".compose-template-*.yml")
		if err != nil {
			return "", fmt.Errorf("create managed template overlay: %w", err)
		}
		overlayFile := overlay.Name()
		defer func() { _ = os.Remove(filepath.Clean(overlayFile)) }()
		if _, err := overlay.WriteString(prepared); err != nil {
			_ = overlay.Close()
			return "", fmt.Errorf("write managed template overlay: %w", err)
		}
		if err := overlay.Close(); err != nil {
			return "", fmt.Errorf("close managed template overlay: %w", err)
		}
		project.File = overlayFile
		content = []byte(prepared)
	}
	if err := ValidatePolicy(string(content)); err != nil {
		return "", err
	}
	binary := d.Binary
	if binary == "" {
		binary = "docker"
	}
	composeFile, cleanup, err := d.prepareNetworks(ctx, project)
	if err != nil {
		return "", err
	}
	defer cleanup()
	composeArgs := []string{"compose", "--project-directory", project.Directory, "--project-name", project.Name}
	if project.EnvFile != "" {
		composeArgs = append(composeArgs, "--env-file", project.EnvFile)
	}
	composeCommand := func(file string, command []string) []string {
		commandArgs := append([]string{}, composeArgs...)
		commandArgs = append(commandArgs, "-f", file)
		return append(commandArgs, command...)
	}
	runCommand := func(command []string, commandSink func(string)) (string, error) {
		cmd := exec.CommandContext(ctx, binary, command...)
		cmd.Dir = project.Directory
		env := os.Environ()
		if len(command) > 0 && command[0] == "compose" {
			filtered := make([]string, 0, len(env))
			for _, value := range env {
				if strings.HasPrefix(value, "DOCKER_API_VERSION=") {
					continue
				}
				filtered = append(filtered, value)
			}
			env = filtered
			env = append(env, "DOCKER_CONFIG=/tmp/aether-docker-config")
		}
		if d.Host != "" {
			env = append(env, "DOCKER_HOST="+d.Host)
		}
		cmd.Env = env
		var outputBuffer bytes.Buffer
		if commandSink == nil {
			cmd.Stdout = &outputBuffer
			cmd.Stderr = &outputBuffer
		} else {
			writer := &streamWriter{buffer: &outputBuffer, sink: commandSink}
			cmd.Stdout = writer
			cmd.Stderr = writer
		}
		err := cmd.Run()
		return outputBuffer.String(), err
	}
	run := func() (string, error) {
		return runCommand(composeCommand(composeFile, args), composeNetworkEndpointLogSink(sink))
	}
	output, err := run()
	if err != nil && isComposeNetworkEndpointError(args, output) {
		if waitErr := waitForComposeRetry(ctx); waitErr != nil {
			return "", waitErr
		}
		output, err = run()
		if err != nil && isComposeNetworkEndpointError(args, output) {
			output, err = d.executeNetworkFallback(project, composeFile, args, composeCommand, runCommand, composeNetworkEndpointLogSink(sink))
		}
	}
	if err != nil && isPlatformManifestError(args, output) {
		output, err = d.executePlatformFallback(project, composeFile, args, composeCommand, runCommand, composeNetworkEndpointLogSink(sink))
	}
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return "", fmt.Errorf("docker compose %s: %w", strings.Join(args, " "), err)
		}
		return string(output), fmt.Errorf("docker compose %s: %s: %w", strings.Join(args, " "), message, err)
	}
	return string(output), nil
}

func inlineManagedTemplateConfigs(content, baseDir string) (string, bool, error) {
	var document map[string]any
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", false, err
	}
	configs, ok := document["configs"].(map[string]any)
	if !ok {
		return content, false, nil
	}
	changed := false
	for name, raw := range configs {
		if !strings.HasPrefix(name, "aether-template-file-") {
			continue
		}
		config, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if content, ok := config["content"].(string); ok {
			config["content"] = escapeComposeConfigContent(content)
			changed = true
			continue
		}
		file, ok := config["file"].(string)
		if !ok || !strings.HasPrefix(file, "./template-mounts/") {
			continue
		}
		candidate, err := managedTemplatePath(baseDir, file)
		if err != nil {
			return "", false, fmt.Errorf("config %s: %w", name, err)
		}
		data, err := os.ReadFile(candidate)
		if err != nil {
			return "", false, fmt.Errorf("read config %s: %w", name, err)
		}
		config["content"] = escapeComposeConfigContent(string(data))
		delete(config, "file")
		changed = true
	}
	if !changed {
		return content, false, nil
	}
	encoded, err := yaml.Marshal(document)
	if err != nil {
		return "", false, err
	}
	return string(encoded), true, nil
}

func escapeComposeConfigContent(content string) string {
	content = strings.ReplaceAll(content, "$$", "$")
	return strings.ReplaceAll(content, "$", "$$")
}

func managedTemplatePath(baseDir, file string) (string, error) {
	root, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	candidate, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(file, "./"))))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes the compose workspace")
	}
	return candidate, nil
}

func isComposeNetworkEndpointError(args []string, output string) bool {
	return len(args) > 0 && args[0] == "up" && strings.Contains(strings.ToLower(output), "multiple network endpoints")
}

func isPlatformManifestError(args []string, output string) bool {
	lower := strings.ToLower(output)
	return len(args) > 0 && args[0] == "up" && strings.Contains(lower, "no matching manifest") && strings.Contains(lower, "linux/arm64")
}

func composeNetworkEndpointLogSink(sink func(string)) func(string) {
	if sink == nil {
		return nil
	}
	return func(line string) {
		if strings.Contains(strings.ToLower(line), "multiple network endpoints") {
			return
		}
		sink(line)
	}
}

func waitForComposeRetry(ctx context.Context) error {
	timer := time.NewTimer(250 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *Docker) executeNetworkFallback(project Project, composeFile string, args []string, composeCommand func(string, []string) []string, runCommand func([]string, func(string)) (string, error), sink func(string)) (string, error) {
	content, err := os.ReadFile(composeFile)
	if err != nil {
		return "", fmt.Errorf("read compose network fallback: %w", err)
	}
	fallbackContent, attachments, err := detachExternalNetworkAttachments(string(content))
	if err != nil {
		return "", err
	}
	if len(attachments) == 0 {
		return "", errors.New("compose network endpoint fallback has no external network attachments")
	}
	if err := ValidatePolicy(fallbackContent); err != nil {
		return "", fmt.Errorf("validate compose network fallback: %w", err)
	}
	fallback, err := os.CreateTemp(project.Directory, ".compose-network-fallback-*.yml")
	if err != nil {
		return "", fmt.Errorf("create compose network fallback: %w", err)
	}
	fallbackFile := fallback.Name()
	defer func() { _ = os.Remove(filepath.Clean(fallbackFile)) }()
	if _, err := fallback.WriteString(fallbackContent); err != nil {
		_ = fallback.Close()
		return "", fmt.Errorf("write compose network fallback: %w", err)
	}
	if err := fallback.Close(); err != nil {
		return "", fmt.Errorf("close compose network fallback: %w", err)
	}

	output, err := runCommand(composeCommand(fallbackFile, args), sink)
	if err != nil {
		return output, err
	}
	if err := connectExternalNetworkAttachments(fallbackFile, attachments, composeCommand, runCommand); err != nil {
		return output, err
	}
	return output, nil
}

func (d *Docker) executePlatformFallback(project Project, composeFile string, args []string, composeCommand func(string, []string) []string, runCommand func([]string, func(string)) (string, error), sink func(string)) (string, error) {
	content, err := os.ReadFile(composeFile)
	if err != nil {
		return "", fmt.Errorf("read compose platform fallback: %w", err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(content, &document); err != nil {
		return "", fmt.Errorf("parse compose platform fallback: %w", err)
	}
	services, ok := document["services"].(map[string]any)
	if !ok || len(services) == 0 {
		return "", errors.New("compose platform fallback has no services")
	}
	for _, rawService := range services {
		service, ok := rawService.(map[string]any)
		if ok {
			if _, exists := service["platform"]; !exists {
				service["platform"] = "linux/amd64"
			}
		}
	}
	fallbackContent, err := yaml.Marshal(document)
	if err != nil {
		return "", fmt.Errorf("encode compose platform fallback: %w", err)
	}
	if err := ValidatePolicy(string(fallbackContent)); err != nil {
		return "", fmt.Errorf("validate compose platform fallback: %w", err)
	}
	fallback, err := os.CreateTemp(project.Directory, ".compose-platform-fallback-*.yml")
	if err != nil {
		return "", fmt.Errorf("create compose platform fallback: %w", err)
	}
	fallbackFile := fallback.Name()
	defer func() { _ = os.Remove(filepath.Clean(fallbackFile)) }()
	if _, err := fallback.Write(fallbackContent); err != nil {
		_ = fallback.Close()
		return "", fmt.Errorf("write compose platform fallback: %w", err)
	}
	if err := fallback.Close(); err != nil {
		return "", fmt.Errorf("close compose platform fallback: %w", err)
	}
	return runCommand(composeCommand(fallbackFile, args), sink)
}

func detachExternalNetworkAttachments(content string) (string, []externalNetworkAttachment, error) {
	var document map[string]any
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", nil, fmt.Errorf("parse compose network fallback: %w", err)
	}
	rawNetworks, ok := document["networks"].(map[string]any)
	if !ok {
		return content, nil, nil
	}
	externalNetworks := make(map[string]string)
	for key, raw := range rawNetworks {
		config, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		external, _ := config["external"].(bool)
		if !external {
			continue
		}
		name := key
		if configuredName, ok := config["name"].(string); ok && strings.TrimSpace(configuredName) != "" {
			name = configuredName
		}
		externalNetworks[key] = name
	}
	if len(externalNetworks) == 0 {
		return content, nil, nil
	}

	services, ok := document["services"].(map[string]any)
	if !ok {
		return "", nil, errors.New("compose network fallback has no services")
	}
	attachments := make([]externalNetworkAttachment, 0)
	serviceNames := make([]string, 0, len(services))
	for serviceName := range services {
		serviceNames = append(serviceNames, serviceName)
	}
	sort.Strings(serviceNames)
	for _, serviceName := range serviceNames {
		rawService := services[serviceName]
		service, ok := rawService.(map[string]any)
		if !ok {
			continue
		}
		rawServiceNetworks, exists := service["networks"]
		if !exists {
			continue
		}
		switch networks := rawServiceNetworks.(type) {
		case map[string]any:
			kept := make(map[string]any, len(networks))
			for networkKey, networkConfig := range networks {
				networkName, external := externalNetworks[networkKey]
				if !external {
					kept[networkKey] = networkConfig
					continue
				}
				attachments = append(attachments, externalNetworkAttachment{Service: serviceName, Network: networkName, Aliases: networkAliases(networkConfig)})
			}
			if len(kept) == 0 {
				delete(service, "networks")
			} else {
				service["networks"] = kept
			}
		case []any:
			kept := make([]any, 0, len(networks))
			for _, network := range networks {
				networkKey := strings.TrimSpace(fmt.Sprint(network))
				networkName, external := externalNetworks[networkKey]
				if !external {
					kept = append(kept, network)
					continue
				}
				attachments = append(attachments, externalNetworkAttachment{Service: serviceName, Network: networkName})
			}
			if len(kept) == 0 {
				delete(service, "networks")
			} else {
				service["networks"] = kept
			}
		}
	}
	for networkKey := range externalNetworks {
		delete(rawNetworks, networkKey)
	}
	if len(rawNetworks) == 0 {
		delete(document, "networks")
	}
	updated, err := yaml.Marshal(document)
	if err != nil {
		return "", nil, fmt.Errorf("encode compose network fallback: %w", err)
	}
	return string(updated), attachments, nil
}

func networkAliases(raw any) []string {
	config, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	values, ok := config["aliases"].([]any)
	if !ok {
		return nil
	}
	aliases := make([]string, 0, len(values))
	for _, value := range values {
		alias := strings.TrimSpace(fmt.Sprint(value))
		if alias != "" {
			aliases = append(aliases, alias)
		}
	}
	return aliases
}

func connectExternalNetworkAttachments(fallbackFile string, attachments []externalNetworkAttachment, composeCommand func(string, []string) []string, runCommand func([]string, func(string)) (string, error)) error {
	for _, attachment := range attachments {
		output, err := runCommand(composeCommand(fallbackFile, []string{"ps", "-q", attachment.Service}), nil)
		if err != nil {
			return fmt.Errorf("list containers for service %s: %w", attachment.Service, err)
		}
		containers := strings.Fields(output)
		if len(containers) == 0 {
			return fmt.Errorf("no containers found for service %s after Compose fallback", attachment.Service)
		}
		for _, container := range containers {
			command := []string{"network", "connect"}
			for _, alias := range attachment.Aliases {
				command = append(command, "--alias", alias)
			}
			command = append(command, attachment.Network, container)
			connectOutput, connectErr := runCommand(command, nil)
			if connectErr != nil && !strings.Contains(strings.ToLower(connectOutput), "already exists") {
				networkOutput, inspectErr := runCommand([]string{"network", "inspect", attachment.Network}, nil)
				if inspectErr == nil && strings.Contains(networkOutput, container) {
					continue
				}
				containerNetworks, containerInspectErr := runCommand([]string{"inspect", "--format", "{{json .NetworkSettings.Networks}}", container}, nil)
				if containerInspectErr == nil && strings.Contains(containerNetworks, attachment.Network) {
					continue
				}
				return fmt.Errorf("connect service %s to network %s: %s: %w", attachment.Service, attachment.Network, strings.TrimSpace(connectOutput), connectErr)
			}
		}
	}
	return nil
}

func normalizeUnsupportedExposeRanges(content string) (string, error) {
	var document map[string]any
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", err
	}
	services, ok := document["services"].(map[string]any)
	if !ok {
		return content, nil
	}
	changed := false
	for _, rawService := range services {
		service, ok := rawService.(map[string]any)
		if !ok {
			continue
		}
		expose, ok := service["expose"].([]any)
		if !ok {
			continue
		}
		kept := make([]any, 0, len(expose))
		for _, rawPort := range expose {
			if isExposePortRange(fmt.Sprint(rawPort)) {
				changed = true
				continue
			}
			kept = append(kept, rawPort)
		}
		if len(kept) == 0 {
			delete(service, "expose")
		} else {
			service["expose"] = kept
		}
	}
	if !changed {
		return content, nil
	}
	encoded, err := yaml.Marshal(document)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func injectPublishedNetwork(content, networkName string) (string, bool, error) {
	var document map[string]any
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", false, err
	}
	services, ok := document["services"].(map[string]any)
	if !ok {
		return content, false, nil
	}
	networks, ok := document["networks"].(map[string]any)
	changed := false
	if !ok {
		networks = map[string]any{}
		document["networks"] = networks
		changed = true
	}
	if _, exists := networks[networkName]; !exists {
		networks[networkName] = map[string]any{"name": networkName, "external": true}
		changed = true
	}
	for _, rawService := range services {
		service, ok := rawService.(map[string]any)
		if !ok {
			continue
		}
		if _, hasNetworkMode := service["network_mode"]; hasNetworkMode {
			continue
		}
		rawNetworks, exists := service["networks"]
		if !exists {
			service["networks"] = map[string]any{
				"default":   map[string]any{},
				networkName: map[string]any{},
			}
			changed = true
			continue
		}
		switch serviceNetworks := rawNetworks.(type) {
		case map[string]any:
			if _, exists := serviceNetworks[networkName]; !exists {
				serviceNetworks[networkName] = map[string]any{}
				changed = true
			}
		case []any:
			found := false
			for _, value := range serviceNetworks {
				if fmt.Sprint(value) == networkName {
					found = true
					break
				}
			}
			if !found {
				service["networks"] = append(serviceNetworks, networkName)
				changed = true
			}
		}
	}
	if !changed {
		return content, false, nil
	}
	updated, err := yaml.Marshal(document)
	if err != nil {
		return "", false, err
	}
	return string(updated), true, nil
}

func isExposePortRange(value string) bool {
	value = strings.TrimSpace(value)
	if slash := strings.LastIndexByte(value, '/'); slash >= 0 {
		value = value[:slash]
	}
	parts := strings.SplitN(value, "-", 2)
	if len(parts) != 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, digit := range part {
			if digit < '0' || digit > '9' {
				return false
			}
		}
	}
	return true
}

func (d *Docker) prepareNetworks(ctx context.Context, project Project) (string, func(), error) {
	content, err := os.ReadFile(project.File)
	if err != nil {
		return "", func() {}, err
	}
	updated, changed, err := markExistingNetworks(string(content), func(name string) bool {
		binary := d.Binary
		if binary == "" {
			binary = "docker"
		}
		command := exec.CommandContext(ctx, binary, "network", "inspect", name)
		if d.Host != "" {
			command.Env = append(os.Environ(), "DOCKER_HOST="+d.Host)
		}
		return command.Run() == nil
	})
	if err != nil {
		return "", func() {}, err
	}
	if !changed {
		return project.File, func() {}, nil
	}
	overlay, err := os.CreateTemp(project.Directory, ".compose-network-*.yml")
	if err != nil {
		return "", func() {}, err
	}
	name := overlay.Name()
	if _, err := overlay.WriteString(updated); err != nil {
		_ = overlay.Close()
		_ = os.Remove(name)
		return "", func() {}, err
	}
	if err := overlay.Close(); err != nil {
		_ = os.Remove(name)
		return "", func() {}, err
	}
	return name, func() { _ = os.Remove(filepath.Clean(name)) }, nil
}

func markExistingNetworks(content string, exists func(string) bool) (string, bool, error) {
	var document map[string]any
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", false, err
	}
	rawNetworks, ok := document["networks"].(map[string]any)
	if !ok {
		return content, false, nil
	}
	changed := false
	for key, raw := range rawNetworks {
		config, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if external, ok := config["external"].(bool); ok && external {
			continue
		}
		if internal, ok := config["internal"].(bool); ok && internal {
			continue
		}
		name, ok := config["name"].(string)
		if !ok || strings.TrimSpace(name) == "" || !exists(name) {
			continue
		}
		config["external"] = true
		rawNetworks[key] = config
		changed = true
	}
	if !changed {
		return content, false, nil
	}
	updated, err := yaml.Marshal(document)
	if err != nil {
		return "", false, err
	}
	return string(updated), true, nil
}

type streamWriter struct {
	buffer *bytes.Buffer
	sink   func(string)
	mu     sync.Mutex
}

func (w *streamWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.sink != nil {
		line := strings.TrimSpace(string(data))
		if line != "" {
			w.sink(line)
		}
	}
	return w.buffer.Write(data)
}

var _ io.Writer = (*streamWriter)(nil)
