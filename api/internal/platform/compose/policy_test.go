package compose

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestValidatePolicyAllowsNormalCompose(t *testing.T) {
	content := `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
    volumes:
      - web-data:/var/cache/nginx
volumes:
  web-data: {}
networks:
  private: {}
`
	if err := ValidatePolicy(content); err != nil {
		t.Fatalf("normal compose rejected: %v", err)
	}
}

func TestValidatePolicyAllowsImplicitNamedResourceDefinitions(t *testing.T) {
	content := `services:
  web:
    image: nginx:alpine
    volumes:
      - web-data:/var/cache/nginx
volumes:
  web-data:
`
	if err := ValidatePolicy(content); err != nil {
		t.Fatalf("implicit named resource definition rejected: %v", err)
	}
}

func TestNormalizeNamedResourceDefinitions(t *testing.T) {
	content, err := NormalizeNamedResourceDefinitions("services:\n  web:\n    image: nginx:alpine\nvolumes:\n  web-data:\nnetworks:\n  private:\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, "web-data:\n") || strings.Contains(content, "private:\n") {
		t.Fatalf("named resources were not normalized: %s", content)
	}
	if err := ValidatePolicy(content); err != nil {
		t.Fatalf("normalized Compose rejected: %v", err)
	}
}

func TestValidatePolicyRejectsPrivilegedRuntimeOptions(t *testing.T) {
	tests := map[string]string{
		"privileged": `services:
  web:
    image: nginx
    privileged: true
`,
		"host network": `services:
  web:
    image: nginx
    network_mode: host
`,
		"host pid": `services:
  web:
    image: nginx
    pid: host
`,
		"runtime override": `services:
  web:
    image: nginx
    runtime: runc
`,
		"credential spec": `services:
  web:
    image: nginx
    credential_spec: config-name
`,
		"devices": `services:
  web:
    image: nginx
    devices:
      - /dev/sda:/dev/sda
`,
		"capabilities": `services:
  web:
    image: nginx
    cap_add: [SYS_ADMIN]
`,
		"socket bind": `services:
  web:
    image: nginx
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
`,
		"relative bind": `services:
  web:
    image: nginx
    volumes:
      - ../host:/host
`,
		"external volume": `services:
  web:
    image: nginx
    volumes:
      - shared:/data
volumes:
  shared:
    external: true
`,
		"external network": `services:
  web:
    image: nginx
networks:
  shared:
    external: true
`,
		"volume driver bind": `services:
  web:
    image: nginx
    volumes:
      - shared:/data
volumes:
  shared:
    driver: local
    driver_opts:
      type: none
      device: /
      o: bind
`,
		"host network driver": `services:
  web:
    image: nginx
    networks: [hostnet]
networks:
  hostnet:
    driver: host
`,
		"root user": `services:
  web:
    image: nginx
    user: "0:0"
`,
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			err := ValidatePolicy(content)
			if err == nil {
				t.Fatal("dangerous compose accepted")
			}
			var policyErr *PolicyError
			if !errors.As(err, &policyErr) {
				t.Fatalf("error type = %T, want PolicyError", err)
			}
		})
	}
}

func TestValidatePolicyRejectsMutableLatestImage(t *testing.T) {
	content := `services:
  app:
    image: example/app:latest
`
	if err := ValidatePolicy(content); err == nil {
		t.Fatal("expected latest image tag to be rejected")
	}
}

func TestValidatePolicyRejectsAlternateVolumeSyntax(t *testing.T) {
	content := `services:
  web:
    image: nginx
    volumes:
      - type: bind
        source: /etc
        target: /host-etc
`
	err := ValidatePolicy(content)
	if err == nil || !strings.Contains(err.Error(), "named volumes") {
		t.Fatalf("bind volume error = %v", err)
	}
}

func TestValidatePolicyAllowsOnlyNoNewPrivileges(t *testing.T) {
	allowed := `services:
  web:
    image: nginx
    security_opt:
      - no-new-privileges:true
`
	if err := ValidatePolicy(allowed); err != nil {
		t.Fatalf("safe security option rejected: %v", err)
	}
	rejected := `services:
  web:
    image: nginx
    security_opt:
      - label=disable
`
	if err := ValidatePolicy(rejected); err == nil {
		t.Fatal("unsafe security option accepted")
	}
}

func TestValidatePolicyRejectsUnboundedResources(t *testing.T) {
	for _, content := range []string{
		`services:
  web:
    image: nginx
    restart: always
`,
		`services:
  web:
    image: nginx
    mem_limit: 4g
`,
		`services:
  web:
    image: nginx
    storage_opt:
      size: 100g
`,
	} {
		if err := ValidatePolicy(content); err == nil {
			t.Fatalf("unsafe resources accepted: %s", content)
		}
	}
}

func TestValidatePolicyRejectsUnsafePublishedPorts(t *testing.T) {
	for _, content := range []string{
		`services:
  web:
    image: nginx
    ports: ["80:8080"]
`,
		`services:
  web:
    image: nginx
    ports:
      - target: 8080
        published: 70000
`,
		`services:
  web:
    image: nginx
    ports: "8080:80"
`,
	} {
		if err := ValidatePolicy(content); err == nil {
			t.Fatalf("unsafe published ports accepted: %s", content)
		}
	}
	if err := ValidatePolicy(`services:
  web:
    image: nginx
    ports: ["8080:80", "80"]
`); err != nil {
		t.Fatalf("valid ports rejected: %v", err)
	}
}

func TestValidatePolicyRejectsExcessivePublishedPorts(t *testing.T) {
	ports := make([]string, 65)
	for i := range ports {
		ports[i] = strconv.Itoa(2000+i) + ":80"
	}
	content := "services:\n  web:\n    image: nginx\n    ports:\n"
	for _, port := range ports {
		content += "      - \"" + port + "\"\n"
	}
	if err := ValidatePolicy(content); err == nil {
		t.Fatal("excessive published ports accepted")
	}
}

func TestValidatePolicyRejectsWorkspaceEscapeInputs(t *testing.T) {
	tests := []string{
		`services:
  web:
    build: /var/lib/aether
`,
		`services:
  web:
    build:
      context: ../../platform
      dockerfile: Dockerfile
`,
		`services:
  web:
    image: nginx
    env_file: /var/lib/aether/.env
`,
	}
	for _, content := range tests {
		if err := ValidatePolicy(content); err == nil {
			t.Fatalf("workspace escape input accepted: %s", content)
		}
	}
}
