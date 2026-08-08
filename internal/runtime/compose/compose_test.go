package compose

import (
	"strings"
	"testing"
)

func TestGenerateValidCompose(t *testing.T) {
	spec := &DeploymentSpec{
		Service:  "web",
		Image:    "nginx:alpine",
		Ports:    []PortMapping{{Host: "8080", Container: "80"}},
		Networks: []string{"aether-x"},
		Restart:  "unless-stopped",
	}
	yaml, err := Generate(spec)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"services:", "web:", "nginx:alpine", "8080:80/tcp", "networks:"} {
		if !strings.Contains(yaml, want) {
			t.Errorf("compose sem %q:\n%s", want, yaml)
		}
	}
}

func TestParseRoundTrip(t *testing.T) {
	in := `services:
  api:
    image: nginx:alpine
    ports:
      - "9090:80/tcp"
    restart: always
`
	spec, err := Parse([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if spec.Image != "nginx:alpine" {
		t.Errorf("image: %q", spec.Image)
	}
	if len(spec.Ports) != 1 || spec.Ports[0].Host != "9090" || spec.Ports[0].Container != "80" {
		t.Errorf("ports: %+v", spec.Ports)
	}
}

func TestExportKubernetes(t *testing.T) {
	spec := &DeploymentSpec{Service: "web", Image: "nginx:alpine", Ports: []PortMapping{{Container: "80"}}}
	out, err := ExportKubernetes(spec)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"kind: Deployment", "image: nginx:alpine", "kind: Service", "containerPort: 80"} {
		if !strings.Contains(out, want) {
			t.Errorf("k8s sem %q:\n%s", want, out)
		}
	}
}

func TestExportNomad(t *testing.T) {
	spec := &DeploymentSpec{Service: "web", Image: "nginx:alpine", Ports: []PortMapping{{Container: "80"}}}
	out, err := ExportNomad(spec)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"job \"web\"", "driver = \"docker\"", "image = \"nginx:alpine\""} {
		if !strings.Contains(out, want) {
			t.Errorf("nomad sem %q:\n%s", want, out)
		}
	}
}
