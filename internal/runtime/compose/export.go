package compose

import (
	"fmt"
	"strings"
)

// ExportKubernetes gera um manifest Kubernetes (Deployment + Service) a partir
// do mesmo DeploymentSpec — sem alterar a UI ou o modelo.
func ExportKubernetes(spec *DeploymentSpec) (string, error) {
	if spec == nil || spec.Service == "" {
		return "", fmt.Errorf("spec vazio")
	}
	var b strings.Builder
	image := spec.Image
	b.WriteString("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: " + spec.Service + "\nspec:\n")
	b.WriteString("  replicas: 1\n  selector:\n    matchLabels:\n      app: " + spec.Service + "\n")
	b.WriteString("  template:\n    metadata:\n      labels:\n        app: " + spec.Service + "\n")
	b.WriteString("    spec:\n      containers:\n        - name: " + spec.Service + "\n")
	if image != "" {
		b.WriteString("          image: " + image + "\n")
	}
	if spec.Resources != nil {
		b.WriteString("          resources:\n")
		if spec.Resources.CPUs > 0 {
			b.WriteString("            limits:\n              cpu: \"" + fCPU(spec.Resources.CPUs) + "\"\n")
		}
		if spec.Resources.Memory != "" {
			b.WriteString("            limits:\n              memory: " + spec.Resources.Memory + "\n")
		}
	}
	if len(spec.Environment) > 0 {
		b.WriteString("          env:\n")
		for k, v := range spec.Environment {
			b.WriteString("            - name: " + k + "\n              value: " + shellQuote(v) + "\n")
		}
	}
	if len(spec.Ports) > 0 {
		b.WriteString("          ports:\n")
		for _, p := range spec.Ports {
			b.WriteString("            - containerPort: " + p.Container + "\n")
		}
	}
	// Service (expõe a primeira porta)
	if len(spec.Ports) > 0 {
		b.WriteString("---\napiVersion: v1\nkind: Service\nmetadata:\n  name: " + spec.Service + "\nspec:\n")
		b.WriteString("  selector:\n    app: " + spec.Service + "\n")
		b.WriteString("  ports:\n")
		for _, p := range spec.Ports {
			b.WriteString("    - port: " + p.Container + "\n      targetPort: " + p.Container + "\n")
		}
	}
	return b.String(), nil
}

// ExportNomad gera um job Nomad (HCL) a partir do mesmo spec.
func ExportNomad(spec *DeploymentSpec) (string, error) {
	if spec == nil || spec.Service == "" {
		return "", fmt.Errorf("spec vazio")
	}
	var b strings.Builder
	b.WriteString("job \"" + spec.Service + "\" {\n")
	b.WriteString("  datacenters = [\"dc1\"]\n")
	b.WriteString("  type = \"service\"\n\n")
	b.WriteString("  group \"" + spec.Service + "\" {\n")
	b.WriteString("    count = 1\n\n")
	b.WriteString("    task \"" + spec.Service + "\" {\n")
	b.WriteString("      driver = \"docker\"\n\n")
	b.WriteString("      config {\n")
	if spec.Image != "" {
		b.WriteString("        image = \"" + spec.Image + "\"\n")
	}
	if len(spec.Ports) > 0 {
		b.WriteString("        ports = [\"http\"]\n")
	}
	b.WriteString("      }\n\n")
	if len(spec.Environment) > 0 {
		b.WriteString("      env {\n")
		for k, v := range spec.Environment {
			b.WriteString("        " + k + " = \"" + shellQuote(v) + "\"\n")
		}
		b.WriteString("      }\n\n")
	}
	if len(spec.Ports) > 0 {
		b.WriteString("      resources {\n")
		b.WriteString("        cpu    = 300\n")
		b.WriteString("        memory = 512\n")
		b.WriteString("        network {\n")
		for _, p := range spec.Ports {
			b.WriteString("          port \"http\" {\n            static = " + p.Container + "\n          }\n")
		}
		b.WriteString("        }\n")
		b.WriteString("      }\n")
	}
	b.WriteString("    }\n")
	b.WriteString("  }\n")
	b.WriteString("}\n")
	return b.String(), nil
}

func fCPU(c float64) string {
	return fmt.Sprintf("%.3f", c)
}

func shellQuote(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}
