package compose

import (
	"bytes"
	"fmt"
	"path"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type PolicyError struct {
	Violations []string
}

func (e *PolicyError) Error() string {
	if len(e.Violations) == 0 {
		return "compose security policy rejected the configuration"
	}
	return "compose security policy rejected the configuration: " + strings.Join(e.Violations, "; ")
}

func ValidatePolicy(content string) error {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return fmt.Errorf("parse compose configuration: %w", err)
	}
	root := documentRoot(&document)
	if root == nil || root.Kind != yaml.MappingNode {
		return fmt.Errorf("compose root must be a mapping")
	}

	violations := make([]string, 0)
	if containsAlias(root) {
		violations = append(violations, "YAML aliases are not supported")
	}
	for _, key := range []string{"secrets", "include"} {
		if nodeMapValue(root, key) != nil {
			violations = append(violations, "top-level "+key+" are not supported")
		}
	}
	validateNamedResources(root, "volumes", &violations)
	validateNamedResources(root, "networks", &violations)
	validateManagedConfigs(root, &violations)
	allowHostPorts := hasDokployHostPortsMarker(&document, root)

	services := nodeMapValue(root, "services")
	if services == nil || services.Kind != yaml.MappingNode || len(services.Content) == 0 {
		violations = append(violations, "services must contain at least one service")
	} else {
		if len(services.Content)/2 > 32 {
			violations = append(violations, "services cannot contain more than 32 entries")
		}
		for i := 0; i+1 < len(services.Content); i += 2 {
			name := services.Content[i].Value
			service := services.Content[i+1]
			if service.Kind != yaml.MappingNode {
				violations = append(violations, "service "+name+" must be a mapping")
				continue
			}
			validateServicePolicy(name, service, allowHostPorts, &violations)
		}
	}
	if len(violations) > 0 {
		return &PolicyError{Violations: violations}
	}
	return nil
}

func NormalizeNamedResourceDefinitions(content string) (string, error) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", fmt.Errorf("parse compose configuration: %w", err)
	}
	root := documentRoot(&document)
	if root == nil || root.Kind != yaml.MappingNode {
		return "", fmt.Errorf("compose root must be a mapping")
	}
	if hasDokployHostPortsMarker(&document, root) && nodeMapValue(root, "x-aether-allow-host-ports") == nil {
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "x-aether-allow-host-ports"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"},
		)
	}
	removeMapValue(root, "version")
	for _, kind := range []string{"volumes", "networks"} {
		resources := nodeMapValue(root, kind)
		if resources == nil || resources.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(resources.Content); i += 2 {
			value := resources.Content[i+1]
			if value.Kind == yaml.ScalarNode && value.Tag == "!!null" {
				resources.Content[i+1] = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			}
		}
	}
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return "", fmt.Errorf("encode compose configuration: %w", err)
	}
	_ = encoder.Close()
	return buffer.String(), nil
}

func NormalizeTemplateCompose(content string) (string, error) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", fmt.Errorf("parse template compose configuration: %w", err)
	}
	root := documentRoot(&document)
	if root == nil || root.Kind != yaml.MappingNode {
		return "", fmt.Errorf("template compose root must be a mapping")
	}

	root = resolveTemplateAliases(root)
	document.Content[0] = root
	normalizeTemplateResourceDefinitions(root)
	services := nodeMapValue(root, "services")
	if services != nil && services.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(services.Content); i += 2 {
			serviceName := services.Content[i].Value
			service := services.Content[i+1]
			if service.Kind == yaml.MappingNode {
				normalizeTemplateService(serviceName, service, root)
			}
		}
	}

	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return "", fmt.Errorf("encode template compose configuration: %w", err)
	}
	_ = encoder.Close()
	return buffer.String(), nil
}

func resolveTemplateAliases(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.AliasNode && node.Alias != nil {
		return resolveTemplateAliases(cloneTemplateNode(node.Alias))
	}
	for index, child := range node.Content {
		node.Content[index] = resolveTemplateAliases(child)
	}
	node.Anchor = ""
	node.Alias = nil
	return node
}

func cloneTemplateNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	clone := *node
	clone.Content = make([]*yaml.Node, len(node.Content))
	for index, child := range node.Content {
		clone.Content[index] = cloneTemplateNode(child)
	}
	clone.Alias = nil
	return &clone
}

func normalizeTemplateResourceDefinitions(root *yaml.Node) {
	volumes := nodeMapValue(root, "volumes")
	if volumes != nil && volumes.Kind == yaml.MappingNode {
		for i := 1; i < len(volumes.Content); i += 2 {
			volumes.Content[i] = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		}
	}
	networks := nodeMapValue(root, "networks")
	if networks != nil && networks.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(networks.Content); i += 2 {
			name := networks.Content[i].Value
			if strings.HasPrefix(name, "aether-") {
				continue
			}
			networks.Content[i+1] = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		}
	}
}

func normalizeTemplateService(serviceName string, service, root *yaml.Node) {
	if volumes := nodeMapValue(service, "volumes"); volumes != nil && volumes.Kind == yaml.SequenceNode {
		resources := nodeMapValue(root, "volumes")
		if resources == nil {
			resources = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			setTemplateMapValue(root, "volumes", resources)
		}
		for index, volume := range volumes.Content {
			name := "aether-template-" + sanitizeResourceName(serviceName) + "-" + strconv.Itoa(index)
			switch volume.Kind {
			case yaml.ScalarNode:
				parts := strings.Split(volume.Value, ":")
				if len(parts) > 1 && unsafeHostPath(parts[0]) {
					parts[0] = name
					volume.Value = strings.Join(parts, ":")
					ensureNamedResource(resources, name)
				}
			case yaml.MappingNode:
				source := nodeMapValue(volume, "source")
				typeValue := nodeMapValue(volume, "type")
				if (source != nil && unsafeHostPath(source.Value)) || (typeValue != nil && strings.EqualFold(strings.TrimSpace(typeValue.Value), "bind")) {
					setTemplateScalarValue(volume, "type", "volume")
					setTemplateScalarValue(volume, "source", name)
					removeMapValue(volume, "bind")
					ensureNamedResource(resources, name)
				}
			}
		}
	}
	for _, key := range []string{"blkio_config", "cpu_rt_period", "cpu_rt_runtime", "memswap_limit", "oom_kill_disable", "shm_size", "storage_opt", "sysctls", "tmpfs", "ulimits", "network_mode", "pid", "ipc", "uts", "userns_mode", "cgroup", "isolation", "runtime", "credential_spec", "privileged", "devices", "cap_add", "volumes_from", "external_links"} {
		removeMapValue(service, key)
	}
	if user := nodeMapValue(service, "user"); user != nil && isRootUser(user.Value) {
		removeMapValue(service, "user")
	}
	if security := nodeMapValue(service, "security_opt"); security != nil {
		if security.Kind != yaml.SequenceNode || len(security.Content) == 0 {
			removeMapValue(service, "security_opt")
		}
	}
	if deploy := nodeMapValue(service, "deploy"); deploy != nil && deploy.Kind == yaml.MappingNode {
		removeMapValue(deploy, "devices")
		removeMapValue(deploy, "resources")
		if len(deploy.Content) == 0 {
			removeMapValue(service, "deploy")
		}
	}
}

func sanitizeResourceName(value string) string {
	var builder strings.Builder
	for _, character := range strings.ToLower(value) {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') {
			builder.WriteRune(character)
			continue
		}
		builder.WriteByte('-')
	}
	name := strings.Trim(builder.String(), "-")
	if name == "" {
		return "service"
	}
	return name
}

func ensureNamedResource(resources *yaml.Node, name string) {
	if nodeMapValue(resources, name) == nil {
		resources.Content = append(resources.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name},
			&yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"},
		)
	}
}

func setTemplateMapValue(mapping *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = value
			return
		}
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value,
	)
}

func setTemplateScalarValue(mapping *yaml.Node, key, value string) {
	setTemplateMapValue(mapping, key, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value})
}

func hasDokployHostPortsMarker(document, root *yaml.Node) bool {
	comments := document.HeadComment + "\n" + document.LineComment + "\n" + document.FootComment
	comments += "\n" + root.HeadComment + "\n" + root.LineComment + "\n" + root.FootComment
	for _, node := range root.Content {
		comments += "\n" + node.HeadComment + "\n" + node.LineComment + "\n" + node.FootComment
	}
	if strings.Contains(strings.ToLower(comments), "dokploy: allow-host-ports") {
		return true
	}
	marker := nodeMapValue(root, "x-aether-allow-host-ports")
	return marker != nil && strings.EqualFold(strings.TrimSpace(marker.Value), "true")
}

func documentRoot(document *yaml.Node) *yaml.Node {
	if document.Kind == yaml.DocumentNode && len(document.Content) > 0 {
		return document.Content[0]
	}
	return document
}

func nodeMapValue(mapping *yaml.Node, key string) *yaml.Node {
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

func removeMapValue(mapping *yaml.Node, key string) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
			return
		}
	}
}

func validateNamedResources(root *yaml.Node, kind string, violations *[]string) {
	resources := nodeMapValue(root, kind)
	if resources == nil {
		return
	}
	if resources.Kind != yaml.MappingNode {
		*violations = append(*violations, "top-level "+kind+" must be a mapping")
		return
	}
	for i := 0; i+1 < len(resources.Content); i += 2 {
		name := resources.Content[i].Value
		value := resources.Content[i+1]
		if value.Kind == yaml.ScalarNode && value.Tag == "!!null" {
			continue
		}
		if value.Kind != yaml.MappingNode {
			*violations = append(*violations, "top-level "+kind+" "+name+" has an invalid definition")
			continue
		}
		for j := 0; j+1 < len(value.Content); j += 2 {
			key := value.Content[j].Value
			switch key {
			case "external":
				if explicitTrue(value.Content[j+1]) && kind == "networks" && strings.HasPrefix(name, "aether-") {
					continue
				}
				if !explicitFalse(value.Content[j+1]) {
					*violations = append(*violations, "top-level "+kind+" "+name+" cannot be external")
				}
			case "name":
				if kind != "networks" || !strings.HasPrefix(name, "aether-") || strings.TrimSpace(value.Content[j+1].Value) != name {
					*violations = append(*violations, "top-level "+kind+" "+name+" has an unsupported option")
				}
			case "internal":
				if kind != "networks" || !explicitBoolean(value.Content[j+1]) {
					*violations = append(*violations, "top-level "+kind+" "+name+" has an unsupported option")
				}
			default:
				*violations = append(*violations, "top-level "+kind+" "+name+" has an unsupported option")
			}
		}
	}
}

func validateServicePolicy(name string, service *yaml.Node, allowHostPorts bool, violations *[]string) {
	if image := nodeMapValue(service, "image"); image != nil && strings.HasSuffix(strings.ToLower(strings.TrimSpace(image.Value)), ":latest") {
		*violations = append(*violations, "service "+name+" cannot use mutable image tag latest")
	}
	if restart := nodeMapValue(service, "restart"); restart != nil && strings.ToLower(strings.TrimSpace(restart.Value)) != "no" {
		*violations = append(*violations, "service "+name+" can only use restart no")
	}
	validateResourceLimit(name, service, violations)
	for _, key := range []string{"network_mode", "pid", "ipc", "uts", "userns_mode", "cgroup", "isolation", "runtime", "credential_spec"} {
		if value := nodeMapValue(service, key); value != nil && strings.TrimSpace(value.Value) != "" {
			*violations = append(*violations, "service "+name+" cannot set "+key)
		}
	}
	if privileged := nodeMapValue(service, "privileged"); privileged != nil && !explicitFalse(privileged) {
		*violations = append(*violations, "service "+name+" cannot be privileged")
	}
	for _, key := range []string{"devices", "cap_add", "volumes_from", "external_links"} {
		if hasItems(nodeMapValue(service, key)) {
			*violations = append(*violations, "service "+name+" cannot set "+key)
		}
	}
	validateSecurityOptions(name, nodeMapValue(service, "security_opt"), violations)
	if user := nodeMapValue(service, "user"); user != nil && isRootUser(user.Value) {
		*violations = append(*violations, "service "+name+" cannot run as root")
	}
	validateServiceVolumes(name, nodeMapValue(service, "volumes"), violations)
	validateServiceConfigs(name, nodeMapValue(service, "configs"), violations)
	validateServiceNetworks(name, nodeMapValue(service, "networks"), violations)
	validateServicePorts(name, nodeMapValue(service, "ports"), allowHostPorts, violations)
	validateServicePathList(name, nodeMapValue(service, "env_file"), "env_file", violations)
	validateBuild(name, nodeMapValue(service, "build"), violations)
	if hasItems(nodeMapValue(service, "extends")) {
		*violations = append(*violations, "service "+name+" cannot extend another service")
	}
	if deploy := nodeMapValue(service, "deploy"); deploy != nil {
		if hasNestedItems(deploy, "devices") {
			*violations = append(*violations, "service "+name+" cannot configure deploy devices")
		}
		if nodeMapValue(deploy, "resources") != nil {
			*violations = append(*violations, "service "+name+" cannot configure deploy resources")
		}
	}
}

func validateManagedConfigs(root *yaml.Node, violations *[]string) {
	configs := nodeMapValue(root, "configs")
	if configs == nil {
		return
	}
	if configs.Kind != yaml.MappingNode {
		*violations = append(*violations, "top-level configs must be a mapping")
		return
	}
	for i := 0; i+1 < len(configs.Content); i += 2 {
		name := configs.Content[i].Value
		value := configs.Content[i+1]
		if !strings.HasPrefix(name, "aether-template-file-") || value.Kind != yaml.MappingNode {
			*violations = append(*violations, "top-level configs only support managed template files")
			continue
		}
		file := nodeMapValue(value, "file")
		content := nodeMapValue(value, "content")
		if (file == nil) == (content == nil) {
			*violations = append(*violations, "managed template config must define exactly one content source")
		}
		if file != nil && (file.Kind != yaml.ScalarNode || !strings.HasPrefix(file.Value, "./template-mounts/") || strings.Contains(file.Value, "..")) {
			*violations = append(*violations, "managed template config file must stay inside template-mounts")
		}
		for j := 0; j+1 < len(value.Content); j += 2 {
			if value.Content[j].Value != "file" && value.Content[j].Value != "content" {
				*violations = append(*violations, "managed template configs only support file or content")
			}
		}
	}
}

func validateServiceConfigs(name string, configs *yaml.Node, violations *[]string) {
	if configs == nil {
		return
	}
	if configs.Kind != yaml.SequenceNode {
		*violations = append(*violations, "service "+name+" configs must be a list")
		return
	}
	for _, config := range configs.Content {
		if config.Kind != yaml.MappingNode {
			*violations = append(*violations, "service "+name+" has an invalid config definition")
			continue
		}
		source := nodeMapValue(config, "source")
		target := nodeMapValue(config, "target")
		if source == nil || !strings.HasPrefix(source.Value, "aether-template-file-") || target == nil || !strings.HasPrefix(target.Value, "/") {
			*violations = append(*violations, "service "+name+" has an invalid managed config")
		}
	}
}

func validateServicePorts(name string, ports *yaml.Node, allowHostPorts bool, violations *[]string) {
	if ports == nil {
		return
	}
	if ports.Kind != yaml.SequenceNode {
		*violations = append(*violations, "service "+name+" ports must be a list")
		return
	}
	if len(ports.Content) > 64 {
		*violations = append(*violations, "service "+name+" cannot publish more than 64 ports")
	}
	for _, port := range ports.Content {
		if port.Kind == yaml.ScalarNode {
			parts := strings.Split(port.Value, ":")
			if len(parts) > 3 || !validPublishedPort(parts, len(parts)-2, allowHostPorts) {
				*violations = append(*violations, "service "+name+" has an invalid published port")
			}
			continue
		}
		if port.Kind != yaml.MappingNode {
			*violations = append(*violations, "service "+name+" has an invalid port definition")
			continue
		}
		published := nodeMapValue(port, "published")
		if published != nil && !validPortValue(published.Value, true, allowHostPorts) {
			*violations = append(*violations, "service "+name+" has an invalid published port")
		}
	}
}

func validPublishedPort(parts []string, index int, allowHostPorts bool) bool {
	if len(parts) == 1 {
		return validPortValue(parts[0], false, allowHostPorts)
	}
	return validPortValue(parts[index], true, allowHostPorts)
}

func validPortValue(value string, published, allowHostPorts bool) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return !published
	}
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return false
	}
	return !published || allowHostPorts || port >= 1024
}

func validateResourceLimit(name string, service *yaml.Node, violations *[]string) {
	for _, key := range []string{"blkio_config", "cpu_rt_period", "cpu_rt_runtime", "memswap_limit", "oom_kill_disable", "shm_size", "storage_opt", "sysctls", "tmpfs", "ulimits"} {
		if nodeMapValue(service, key) != nil {
			*violations = append(*violations, "service "+name+" cannot set "+key)
		}
	}
	if value := nodeMapValue(service, "mem_limit"); value != nil {
		bytes, ok := parseMemoryLimit(value.Value)
		if !ok || bytes <= 0 || bytes > 2*1024*1024*1024 {
			*violations = append(*violations, "service "+name+" memory limit must be between 1 byte and 2GiB")
		}
	}
	if value := nodeMapValue(service, "cpus"); value != nil {
		cpus, err := strconv.ParseFloat(strings.TrimSpace(value.Value), 64)
		if err != nil || cpus <= 0 || cpus > 2 {
			*violations = append(*violations, "service "+name+" CPU limit must be between 0 and 2")
		}
	}
	if value := nodeMapValue(service, "pids_limit"); value != nil {
		pids, err := strconv.Atoi(strings.TrimSpace(value.Value))
		if err != nil || pids <= 0 || pids > 1024 {
			*violations = append(*violations, "service "+name+" PID limit must be between 1 and 1024")
		}
	}
}

func parseMemoryLimit(raw string) (int64, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	units := []struct {
		suffix string
		factor float64
	}{
		{"gib", 1024 * 1024 * 1024}, {"gb", 1000 * 1000 * 1000}, {"gi", 1024 * 1024 * 1024}, {"g", 1000 * 1000 * 1000},
		{"mib", 1024 * 1024}, {"mb", 1000 * 1000}, {"mi", 1024 * 1024}, {"m", 1000 * 1000},
		{"kib", 1024}, {"kb", 1000}, {"ki", 1024}, {"k", 1000}, {"b", 1},
	}
	for _, unit := range units {
		if strings.HasSuffix(value, unit.suffix) {
			n, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, unit.suffix)), 64)
			if err != nil || n <= 0 {
				return 0, false
			}
			return int64(n * unit.factor), true
		}
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil || n <= 0 {
		return 0, false
	}
	return int64(n), true
}

func validateServiceVolumes(name string, volumes *yaml.Node, violations *[]string) {
	if volumes == nil || volumes.Kind != yaml.SequenceNode {
		return
	}
	for _, volume := range volumes.Content {
		if volume.Kind == yaml.ScalarNode {
			parts := strings.Split(volume.Value, ":")
			if len(parts) > 1 && unsafeHostPath(parts[0]) {
				*violations = append(*violations, "service "+name+" cannot bind host path volumes")
			}
			continue
		}
		if volume.Kind != yaml.MappingNode {
			*violations = append(*violations, "service "+name+" has an invalid volume definition")
			continue
		}
		typeValue := nodeMapValue(volume, "type")
		if typeValue != nil && strings.ToLower(strings.TrimSpace(typeValue.Value)) != "volume" {
			*violations = append(*violations, "service "+name+" can only use named volumes")
			continue
		}
		if source := nodeMapValue(volume, "source"); source != nil && unsafeHostPath(source.Value) {
			*violations = append(*violations, "service "+name+" cannot bind host path volumes")
		}
	}
}

func validateServiceNetworks(name string, networks *yaml.Node, violations *[]string) {
	if networks == nil || networks.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(networks.Content); i += 2 {
		networkName := networks.Content[i].Value
		config := networks.Content[i+1]
		if config.Kind != yaml.MappingNode {
			continue
		}
		external := nodeMapValue(config, "external")
		if nodeMapValue(config, "name") != nil || (external != nil && !explicitFalse(external)) {
			*violations = append(*violations, "service "+name+" cannot attach external network "+networkName)
		}
	}
}

func validateBuild(name string, build *yaml.Node, violations *[]string) {
	if build == nil {
		return
	}
	if build.Kind == yaml.ScalarNode {
		if unsafeWorkspacePath(build.Value) {
			*violations = append(*violations, "service "+name+" build context must stay inside the workspace")
		}
		return
	}
	if build.Kind != yaml.MappingNode {
		*violations = append(*violations, "service "+name+" has an invalid build definition")
		return
	}
	for _, key := range []string{"context", "dockerfile"} {
		if value := nodeMapValue(build, key); value != nil && (value.Kind != yaml.ScalarNode || unsafeWorkspacePath(value.Value)) {
			*violations = append(*violations, "service "+name+" build "+key+" must stay inside the workspace")
		}
	}
	for _, key := range []string{"ssh", "secrets", "additional_contexts"} {
		if hasItems(nodeMapValue(build, key)) {
			*violations = append(*violations, "service "+name+" cannot set build "+key)
		}
	}
	if privileged := nodeMapValue(build, "privileged"); privileged != nil && !explicitFalse(privileged) {
		*violations = append(*violations, "service "+name+" cannot use privileged build settings")
	}
	if network := nodeMapValue(build, "network"); network != nil && strings.EqualFold(strings.TrimSpace(network.Value), "host") {
		*violations = append(*violations, "service "+name+" cannot use host build networking")
	}
}

func validateServicePathList(name string, paths *yaml.Node, key string, violations *[]string) {
	if paths == nil {
		return
	}
	if paths.Kind != yaml.SequenceNode {
		paths = &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{paths}}
	}
	for _, item := range paths.Content {
		if item.Kind != yaml.ScalarNode || unsafeWorkspacePath(item.Value) {
			*violations = append(*violations, "service "+name+" "+key+" must stay inside the workspace")
		}
	}
}

func hasItems(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	if node.Kind == yaml.SequenceNode || node.Kind == yaml.MappingNode {
		return len(node.Content) > 0
	}
	return strings.TrimSpace(node.Value) != ""
}

func explicitFalse(node *yaml.Node) bool {
	return node != nil && node.Tag == "!!bool" && strings.EqualFold(strings.TrimSpace(node.Value), "false")
}

func explicitTrue(node *yaml.Node) bool {
	return node != nil && node.Tag == "!!bool" && strings.EqualFold(strings.TrimSpace(node.Value), "true")
}

func explicitBoolean(node *yaml.Node) bool {
	return node != nil && node.Tag == "!!bool"
}

func validateSecurityOptions(name string, options *yaml.Node, violations *[]string) {
	if options == nil {
		return
	}
	if options.Kind != yaml.SequenceNode {
		*violations = append(*violations, "service "+name+" can only set security_opt as a list")
		return
	}
	for _, option := range options.Content {
		if option.Kind != yaml.ScalarNode || strings.ToLower(strings.TrimSpace(option.Value)) != "no-new-privileges:true" {
			*violations = append(*violations, "service "+name+" can only use no-new-privileges in security_opt")
		}
	}
}

func containsAlias(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	if node.Kind == yaml.AliasNode {
		return true
	}
	for _, child := range node.Content {
		if containsAlias(child) {
			return true
		}
	}
	return false
}

func hasNestedItems(node *yaml.Node, key string) bool {
	if node == nil || node.Kind != yaml.MappingNode {
		return false
	}
	if hasItems(nodeMapValue(node, key)) {
		return true
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if hasNestedItems(node.Content[i+1], key) {
			return true
		}
	}
	return false
}

func isRootUser(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "root" || value == "0" || value == "0:0" {
		return true
	}
	return strings.HasPrefix(value, "0:")
}

func unsafeHostPath(value string) bool {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "~") || value == "." || value == ".." || strings.HasPrefix(value, "./") || strings.HasPrefix(value, "../") || strings.Contains(value, "/") || strings.Contains(value, "$") {
		return true
	}
	clean := path.Clean(value)
	return clean == ".." || strings.HasPrefix(clean, "../") || (len(value) > 1 && value[1] == ':')
}

func unsafeWorkspacePath(value string) bool {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "~") || strings.Contains(value, "$") || (len(value) > 1 && value[1] == ':') {
		return true
	}
	clean := path.Clean(value)
	return clean == ".." || strings.HasPrefix(clean, "../")
}
