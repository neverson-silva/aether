package compose

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
)

// Generate serializa um DeploymentSpec em um docker-compose.yml completo.
func Generate(spec *DeploymentSpec) (string, error) {
	return GenerateWith(spec, "")
}

// GenerateWith serializa o spec definindo um container_name fixo.
func GenerateWith(spec *DeploymentSpec, containerName string) (string, error) {
	if spec == nil {
		return "", fmt.Errorf("spec vazio")
	}
	if spec.Service == "" {
		return "", fmt.Errorf("spec sem service name")
	}
	svc := ServiceDef{
		Image:         spec.Image,
		Build:         spec.Build,
		ContainerName: containerName,
		NetworkAlias:  spec.NetworkAlias,
		Command:       spec.Command,
		Entrypoint:    spec.Entrypoint,
		Environment:   spec.Environment,
		Secrets:       spec.Secrets,
		Networks:      spec.Networks,
		Labels:        spec.Labels,
		Restart:       restartOrDefault(spec.Restart),
		Healthcheck:   spec.Healthcheck,
	}
	for _, p := range spec.Ports {
		svc.Ports = append(svc.Ports, portString(p))
	}
	for _, v := range spec.Volumes {
		svc.Volumes = append(svc.Volumes, volumeString(v))
	}
	if spec.Resources != nil {
		svc.Deploy = &DeployBlock{Resources: &ResourceLimits{Limits: spec.Resources}}
	}

	doc := ComposeDocument{
		Services: map[string]ServiceDef{spec.Service: svc},
	}
	if len(spec.Networks) > 0 {
		doc.Networks = map[string]any{}
		for _, n := range spec.Networks {
			doc.Networks[n] = map[string]any{}
		}
	}
	// volumes nomeados declarados no topo
	topVols := map[string]any{}
	for _, v := range spec.Volumes {
		if v.Source != "" && !looksLikePath(v.Source) {
			topVols[v.Source] = map[string]any{}
		}
	}
	if len(topVols) > 0 {
		doc.Volumes = topVols
	}

	var buf bytes.Buffer
	enc := newYAMLEncoder(&buf)
	if err := enc.Encode(doc); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func restartOrDefault(r string) string {
	if r == "" {
		return "unless-stopped"
	}
	return r
}

// GenerateMulti serializa vários specs em um compose de múltiplos serviços
// (usado pelo Marketplace). Cada spec é um serviço; volumes nomeados declarados
// no topo são coletados automaticamente.
func GenerateMulti(specs []*DeploymentSpec) (string, error) {
	if len(specs) == 0 {
		return "", fmt.Errorf("specs vazios")
	}
	doc := ComposeDocument{Services: map[string]ServiceDef{}}
	topVols := map[string]any{}
	for _, spec := range specs {
		if spec == nil || spec.Service == "" {
			continue
		}
		svc := ServiceDef{
			Image:        spec.Image,
			Build:        spec.Build,
			NetworkAlias: spec.NetworkAlias,
			Command:      spec.Command,
			Entrypoint:   spec.Entrypoint,
			Environment:  spec.Environment,
			Secrets:      spec.Secrets,
			Networks:     spec.Networks,
			Labels:       spec.Labels,
			Restart:      restartOrDefault(spec.Restart),
			Healthcheck:  spec.Healthcheck,
		}
		for _, p := range spec.Ports {
			svc.Ports = append(svc.Ports, portString(p))
		}
		for _, v := range spec.Volumes {
			svc.Volumes = append(svc.Volumes, volumeString(v))
			if v.Source != "" && !looksLikePath(v.Source) {
				topVols[v.Source] = map[string]any{}
			}
		}
		if spec.Resources != nil {
			svc.Deploy = &DeployBlock{Resources: &ResourceLimits{Limits: spec.Resources}}
		}
		doc.Services[spec.Service] = svc
	}
	if len(doc.Services) == 0 {
		return "", fmt.Errorf("nenhum serviço válido")
	}
	if len(topVols) > 0 {
		doc.Volumes = topVols
	}
	var buf bytes.Buffer
	enc := newYAMLEncoder(&buf)
	if err := enc.Encode(doc); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func portString(p PortMapping) string {
	proto := p.Protocol
	if proto == "" {
		proto = "tcp"
	}
	if p.Host == "" {
		return fmt.Sprintf("%s/%s", p.Container, proto)
	}
	return fmt.Sprintf("%s:%s/%s", p.Host, p.Container, proto)
}

func volumeString(v VolumeSpec) string {
	if v.Source == "" {
		return v.Target
	}
	ro := ""
	if v.ReadOnly {
		ro = ":ro"
	}
	return v.Source + ":" + v.Target + ro
}

func looksLikePath(s string) bool {
	return len(s) > 0 && (s[0] == '/' || s[0] == '.' || s[0] == '~' || s[0] == '$')
}

// SortedKeys ajuda a manter a saída determinística.
func SortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func itoa(n int) string { return strconv.Itoa(n) }
