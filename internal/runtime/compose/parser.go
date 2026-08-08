package compose

import (
	"bytes"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

func newYAMLEncoder(w io.Writer) *yaml.Encoder {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	return enc
}

// Parse converte um docker-compose.yml em um DeploymentSpec (importação).
// Suporta um único serviço (o primeiro do map services).
func Parse(data []byte) (*DeploymentSpec, error) {
	var doc struct {
		Version  string                 `yaml:"version"`
		Services map[string]*ServiceDef `yaml:"services"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse compose: %w", err)
	}
	if len(doc.Services) == 0 {
		return nil, fmt.Errorf("compose sem services")
	}
	// primeiro serviço (ordem determinística para single-service)
	var name string
	var svc *ServiceDef
	for n, s := range doc.Services {
		if s == nil {
			continue
		}
		name = n
		svc = s
		break
	}
	if svc == nil {
		return nil, fmt.Errorf("compose sem serviço válido")
	}

	spec := &DeploymentSpec{
		Service:     name,
		Image:       svc.Image,
		Build:       svc.Build,
		Command:     svc.Command,
		Entrypoint:  svc.Entrypoint,
		Environment: svc.Environment,
		Secrets:     svc.Secrets,
		Networks:    svc.Networks,
		Labels:      svc.Labels,
		Restart:     svc.Restart,
		Healthcheck: svc.Healthcheck,
	}
	for _, p := range svc.Ports {
		spec.Ports = append(spec.Ports, parsePort(p))
	}
	for _, v := range svc.Volumes {
		spec.Volumes = append(spec.Volumes, parseVolume(v))
	}
	if svc.Deploy != nil && svc.Deploy.Resources != nil {
		spec.Resources = svc.Deploy.Resources.Limits
	}
	return spec, nil
}

func parsePort(p any) PortMapping {
	switch v := p.(type) {
	case string:
		// "8080:80/tcp" | "80/tcp"
		proto := "tcp"
		rest := v
		for i := len(v) - 1; i >= 0; i-- {
			if v[i] == '/' {
				proto = v[i+1:]
				rest = v[:i]
				break
			}
		}
		host, container, _ := cutLast(rest, ":")
		return PortMapping{Host: host, Container: container, Protocol: proto}
	case map[string]any:
		pm := PortMapping{}
		if h, ok := v["published"].(string); ok {
			pm.Host = h
		}
		if c, ok := v["target"].(int); ok {
			pm.Container = itoa(c)
		} else if c, ok := v["target"].(string); ok {
			pm.Container = c
		}
		return pm
	}
	return PortMapping{}
}

func parseVolume(v any) VolumeSpec {
	switch s := v.(type) {
	case string:
		// "vol:/data:ro" | "/host:/data" | "/data"
		parts := splitVolume(s)
		switch len(parts) {
		case 1:
			return VolumeSpec{Target: parts[0]}
		default:
			vs := VolumeSpec{Source: parts[0], Target: parts[1]}
			for _, f := range parts[2:] {
				if f == "ro" {
					vs.ReadOnly = true
				}
			}
			return vs
		}
	case map[string]any:
		vs := VolumeSpec{}
		if s, ok := s["source"].(string); ok {
			vs.Source = s
		}
		if t, ok := s["target"].(string); ok {
			vs.Target = t
		}
		return vs
	}
	return VolumeSpec{}
}

func splitVolume(s string) []string {
	var parts []string
	var cur bytes.Buffer
	for i := 0; i < len(s); i++ {
		if s[i] == ':' && (i == 0 || s[i-1] != '\\') {
			parts = append(parts, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(s[i])
	}
	parts = append(parts, cur.String())
	return parts
}

func cutLast(s, sep string) (string, string, bool) {
	for i := len(s) - len(sep); i >= 0; i-- {
		if s[i:i+len(sep)] == sep {
			return s[:i], s[i+len(sep):], true
		}
	}
	return "", s, false
}
