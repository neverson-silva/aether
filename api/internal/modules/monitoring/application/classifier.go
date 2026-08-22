package application

import (
	"regexp"
	"strings"

	"aether/internal/modules/monitoring/domain"
)

type rawContainer struct {
	ID       string
	Name     string
	State    string
	Labels   map[string]string
	CPU      float64
	MemUsage uint64
	MemLimit uint64
	MemPerc  float64
	NetIn    uint64
	NetOut   uint64
	BlockIn  uint64
	BlockOut uint64
	HasStats bool
}

var aetherServiceNames = map[string]string{
	"aether-api":      "api",
	"aether-postgres": "postgres",
	"aether-redis":    "redis",
	"aether-web":      "web",
	"aether-registry": "registry",
	"aether-traefik":  "proxy",
}

var userContainerPattern = regexp.MustCompile(`^aether-[0-9a-f]{8}-\d+$`)

// Classify determines ownership (aether/user/unknown) using a single source
// of truth: deploy-time labels (aether.owner, aether.service-*). Legacy
// containers without labels fall back to deterministic, centralized name
// patterns. No ownership rule is duplicated anywhere else in the codebase.
func Classify(raw rawContainer) (owner, serviceType, serviceID, projectID, name string) {
	name = raw.Name
	if owner = raw.Labels["aether.owner"]; owner != "" {
		serviceType = raw.Labels["aether.service-type"]
		serviceID = raw.Labels["aether.service-id"]
		projectID = raw.Labels["aether.project-id"]
		if n := raw.Labels["aether.service-name"]; n != "" {
			name = n
		}
		if owner == domain.OwnerAether && serviceType == "" {
			serviceType = raw.Labels["aether.service"]
		}
		if serviceType == "" {
			serviceType = "container"
		}
		return owner, serviceType, serviceID, projectID, name
	}

	// Legacy fallback: platform containers (pre-label installs).
	if svc, ok := aetherServiceNames[raw.Name]; ok {
		return domain.OwnerAether, svc, "", "", raw.Name
	}
	if strings.HasPrefix(raw.Name, "aether-") && strings.Contains(raw.Name, "-test") {
		return domain.OwnerAether, "test", "", "", raw.Name
	}
	// Legacy fallback: user application containers (pre-label deploys).
	if userContainerPattern.MatchString(raw.Name) {
		return domain.OwnerUser, "app", "", "", raw.Name
	}
	return domain.OwnerUnknown, "container", "", "", raw.Name
}
