package compose

import (
	"fmt"
	"strconv"
	"strings"
)

// Validate verifica consistência básica de um DeploymentSpec.
func Validate(spec *DeploymentSpec) []string {
	var errs []string
	if spec == nil {
		return []string{"spec vazio"}
	}
	if spec.Service == "" {
		errs = append(errs, "service name obrigatório")
	}
	if spec.Image == "" && spec.Build == nil {
		errs = append(errs, "spec precisa de image ou build")
	}
	for _, p := range spec.Ports {
		if p.Container == "" {
			errs = append(errs, "porta sem container port")
			continue
		}
		if _, err := strconv.Atoi(strings.TrimSuffix(p.Container, "/tcp")); err != nil {
			errs = append(errs, fmt.Sprintf("porta inválida: %q", p.Container))
		}
	}
	return errs
}

// ServiceName sanitiza um nome para ser usado como chave de serviço no compose.
func ServiceName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + 32)
		default:
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		out = "app"
	}
	return out
}
