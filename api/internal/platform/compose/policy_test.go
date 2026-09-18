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

func TestValidatePolicyAllowsDokployHostPortsMarker(t *testing.T) {
	content := `# dokploy: allow-host-ports
services:
  dns:
    image: adguard/adguardhome
    ports:
      - "53:53/tcp"
      - "53:53/udp"
`
	if err := ValidatePolicy(content); err != nil {
		t.Fatalf("Dokploy host ports rejected: %v", err)
	}
	normalized, err := NormalizeNamedResourceDefinitions(content)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePolicy(normalized); err != nil {
		t.Fatalf("normalized Dokploy host ports rejected: %v\n%s", err, normalized)
	}
}

func TestValidatePolicyRejectsPrivilegedHostPortsWithoutMarker(t *testing.T) {
	content := `services:
  dns:
    image: adguard/adguardhome
    ports:
      - "53:53/tcp"
`
	if err := ValidatePolicy(content); err == nil {
		t.Fatal("privileged host port accepted without marker")
	}
}

func TestValidatePolicyAllowsManagedTemplateConfigs(t *testing.T) {
	content := `services:
  web:
    image: nginx:alpine
    configs:
      - source: aether-template-file-0
        target: /etc/nginx/nginx.conf
configs:
  aether-template-file-0:
    file: ./template-mounts/0
`
	if err := ValidatePolicy(content); err != nil {
		t.Fatalf("managed template config rejected: %v", err)
	}
	if err := ValidatePolicy(strings.ReplaceAll(content, "./template-mounts/0", "../../secret")); err == nil {
		t.Fatal("template config workspace escape accepted")
	}
}

func TestValidatePolicyAllowsInlineManagedTemplateConfigs(t *testing.T) {
	content := `services:
  web:
    image: nginx:alpine
    configs:
      - source: aether-template-file-0
        target: /etc/nginx/nginx.conf
configs:
  aether-template-file-0:
    content: "worker_processes 1;"
`
	if err := ValidatePolicy(content); err != nil {
		t.Fatalf("inline managed template config rejected: %v", err)
	}
}

func TestValidatePolicyAllowsManagedIngressNetwork(t *testing.T) {
	content := `services:
  web:
    image: nginx:1.27
    networks:
      aether-ingress:
        aliases:
          - app-12345678-web
networks:
  aether-ingress:
    name: aether-ingress
    external: true
`
	if err := ValidatePolicy(content); err != nil {
		t.Fatalf("managed ingress network rejected: %v", err)
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
	content, err := NormalizeNamedResourceDefinitions("version: \"3.8\"\nservices:\n  web:\n    image: nginx:alpine\nvolumes:\n  web-data:\nnetworks:\n  private:\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, "version:") || strings.Contains(content, "web-data:\n") || strings.Contains(content, "private:\n") {
		t.Fatalf("named resources were not normalized: %s", content)
	}
	if err := ValidatePolicy(content); err != nil {
		t.Fatalf("normalized Compose rejected: %v", err)
	}
}

func TestNormalizeTemplateComposeReplacesHostStateAndUnsupportedOptions(t *testing.T) {
	content, err := NormalizeTemplateCompose(`services:
  web:
    image: example/web:1
    user: "0:0"
    shm_size: 1gb
    volumes:
      - ./data:/var/lib/web
      - type: bind
        source: /var/lib/web-config
        target: /etc/web
    deploy:
      resources:
        limits:
          cpus: "4"
volumes:
  data:
    driver: local
    driver_opts:
      type: none
`)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePolicy(content); err != nil {
		t.Fatalf("normalized template compose rejected: %v\n%s", err, content)
	}
	if strings.Contains(content, "./data:/var/lib/web") || strings.Contains(content, "shm_size:") || strings.Contains(content, "user: 0:0") {
		t.Fatalf("unsupported template settings remained: %s", content)
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

func TestNormalizeUserComposeAllowsMutableImageTags(t *testing.T) {
	content, err := NormalizeUserCompose(`services:
  app:
    image: example/app:latest
`)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePolicy(content); err != nil {
		t.Fatalf("user compose with mutable image rejected: %v", err)
	}
}

func TestNormalizeUserComposeAllowsDokployCompatibleOptions(t *testing.T) {
	content, err := NormalizeUserCompose(`services:
  waf:
    build:
      context: .
      dockerfile: Dockerfile
    image: findash/waf:latest
    restart: unless-stopped
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
    networks:
      - waf_net
volumes:
  geoip_data:
networks:
  waf_net:
    name: waf_net
    driver: bridge
`)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePolicy(content); err != nil {
		t.Fatalf("Dokploy-compatible user compose rejected: %v", err)
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
