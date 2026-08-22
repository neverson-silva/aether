package application

import (
	"strings"
	"testing"

	"aether/internal/modules/specs/domain"
)

func TestExportCompose(t *testing.T) {
	s := &Specs{}
	spec := domain.Spec{Name: "web", Image: "nginx:alpine", Port: 80, Env: map[string]string{"A": "1"}}
	out, err := s.ExportCompose(spec)
	if err != nil || !strings.Contains(out, "nginx:alpine") {
		t.Fatalf("compose: %v %s", err, out)
	}
}

func TestExportKubernetes(t *testing.T) {
	s := &Specs{}
	out, err := s.ExportKubernetes(domain.Spec{Name: "web", Image: "nginx", Port: 80})
	if err != nil || !strings.Contains(out, "kind: Deployment") {
		t.Fatalf("k8s: %v %s", err, out)
	}
}

func TestExportNomad(t *testing.T) {
	s := &Specs{}
	out, err := s.ExportNomad(domain.Spec{Name: "web", Image: "nginx", Port: 8080})
	if err != nil || !strings.Contains(out, `image = "nginx"`) {
		t.Fatalf("nomad: %v %s", err, out)
	}
}
