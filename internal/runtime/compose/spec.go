// Package compose implementa a arquitetura "Deployment Spec First".
// O DeploymentSpec é a fonte de verdade tipada e independente de runtime;
// o generator serializa em docker-compose.yml e o parser faz o caminho inverso
// (importação). Outros runtimes (kubernetes, swarm, nomad) poderão consumir o
// mesmo modelo sem alterar a UI.
package compose

// BuildSpec descreve a construção da imagem (Dockerfile).
type BuildSpec struct {
	Context    string   `yaml:"context,omitempty" json:"context,omitempty"`
	Dockerfile string   `yaml:"dockerfile,omitempty" json:"dockerfile,omitempty"`
	Args       []string `yaml:"args,omitempty" json:"args,omitempty"`
}

// PortMapping representa uma exposição de porta host->container.
type PortMapping struct {
	Host      string `yaml:"host,omitempty" json:"host,omitempty"`
	Container string `yaml:"container" json:"container"`
	Protocol  string `yaml:"protocol,omitempty" json:"protocol,omitempty"`
}

// VolumeSpec é um volume ou bind mount.
type VolumeSpec struct {
	Source   string `yaml:"source,omitempty" json:"source,omitempty"`
	Target   string `yaml:"target" json:"target"`
	ReadOnly bool   `yaml:"read_only,omitempty" json:"read_only,omitempty"`
}

// HealthcheckSpec espelha o healthcheck do container.
type HealthcheckSpec struct {
	Test        []string `yaml:"test,omitempty" json:"test,omitempty"`
	Interval    string   `yaml:"interval,omitempty" json:"interval,omitempty"`
	Timeout     string   `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retries     int      `yaml:"retries,omitempty" json:"retries,omitempty"`
	StartPeriod string   `yaml:"start_period,omitempty" json:"start_period,omitempty"`
}

// ResourcesSpec limita CPU/memória.
type ResourcesSpec struct {
	CPUs   float64 `yaml:"cpus,omitempty" json:"cpus,omitempty"`
	Memory string  `yaml:"memory,omitempty" json:"memory,omitempty"`
}

// SecretRef referencia um segredo do deployment.
type SecretRef struct {
	Name  string `yaml:"name" json:"name"`
	Value string `yaml:"value,omitempty" json:"value,omitempty"`
}

// DeploymentSpec é a única fonte de verdade da configuração de um serviço.
type DeploymentSpec struct {
	Service      string            `yaml:"service" json:"service"`
	Build        *BuildSpec        `yaml:"build,omitempty" json:"build,omitempty"`
	Image        string            `yaml:"image,omitempty" json:"image,omitempty"`
	Command      []string          `yaml:"command,omitempty" json:"command,omitempty"`
	Entrypoint   []string          `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
	Ports        []PortMapping     `yaml:"ports,omitempty" json:"ports,omitempty"`
	Expose       []string          `yaml:"expose,omitempty" json:"expose,omitempty"`
	Environment  map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`
	Secrets      []SecretRef       `yaml:"secrets,omitempty" json:"secrets,omitempty"`
	Volumes      []VolumeSpec      `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Networks     []string          `yaml:"networks,omitempty" json:"networks,omitempty"`
	NetworkAlias string            `yaml:"network_alias,omitempty" json:"network_alias,omitempty"`
	Domains      []string          `yaml:"domains,omitempty" json:"domains,omitempty"`
	Labels       map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Restart      string            `yaml:"restart,omitempty" json:"restart,omitempty"`
	Healthcheck  *HealthcheckSpec  `yaml:"healthcheck,omitempty" json:"healthcheck,omitempty"`
	Resources    *ResourcesSpec    `yaml:"resources,omitempty" json:"resources,omitempty"`
	Runtime      string            `yaml:"runtime,omitempty" json:"runtime,omitempty"`
	Strategy     string            `yaml:"deploy_strategy,omitempty" json:"deploy_strategy,omitempty"`
}

// ComposeDocument é o documento docker-compose completo (top-level).
type ComposeDocument struct {
	Version  string                `yaml:"version,omitempty" json:"version,omitempty"`
	Services map[string]ServiceDef `yaml:"services" json:"services"`
	Networks map[string]any        `yaml:"networks,omitempty" json:"networks,omitempty"`
	Volumes  map[string]any        `yaml:"volumes,omitempty" json:"volumes,omitempty"`
}

// ServiceDef é a serialização de um DeploymentSpec dentro do services map.
type ServiceDef struct {
	Image         string            `yaml:"image,omitempty" json:"image,omitempty"`
	Build         *BuildSpec        `yaml:"build,omitempty" json:"build,omitempty"`
	ContainerName string            `yaml:"container_name,omitempty" json:"container_name,omitempty"`
	Command       []string          `yaml:"command,omitempty" json:"command,omitempty"`
	Entrypoint    []string          `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
	Ports         []any             `yaml:"ports,omitempty" json:"ports,omitempty"`
	Expose        []string          `yaml:"expose,omitempty" json:"expose,omitempty"`
	Environment   map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`
	Secrets       []SecretRef       `yaml:"secrets,omitempty" json:"secrets,omitempty"`
	Volumes       []any             `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Networks      []string          `yaml:"networks,omitempty" json:"networks,omitempty"`
	NetworkAlias  string            `yaml:"network_alias,omitempty" json:"network_alias,omitempty"`
	Labels        map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Restart       string            `yaml:"restart,omitempty" json:"restart,omitempty"`
	Healthcheck   *HealthcheckSpec  `yaml:"healthcheck,omitempty" json:"healthcheck,omitempty"`
	Deploy        *DeployBlock      `yaml:"deploy,omitempty" json:"deploy,omitempty"`
}

// DeployBlock contém recursos/replicas (docker compose deploy).
type DeployBlock struct {
	Resources *ResourceLimits `yaml:"resources,omitempty" json:"resources,omitempty"`
	Replicas  int             `yaml:"replicas,omitempty" json:"replicas,omitempty"`
}

// ResourceLimits segue a spec do compose: deploy.resources.limits.
type ResourceLimits struct {
	Limits *ResourcesSpec `yaml:"limits,omitempty" json:"limits,omitempty"`
}
