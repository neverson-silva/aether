package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestInlineManagedTemplateConfigs(t *testing.T) {
	baseDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(baseDir, "template-mounts"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "template-mounts", "0"), []byte("port = 6379\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	content, changed, err := inlineManagedTemplateConfigs(`services:
  dragonfly:
    image: dragonflydb/dragonfly
    configs:
      - source: aether-template-file-0
        target: /etc/dragonfly.conf
configs:
  aether-template-file-0:
    file: ./template-mounts/0
`, baseDir)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || strings.Contains(content, "file: ./template-mounts/0") || !strings.Contains(content, "port = 6379") {
		t.Fatalf("managed template config was not inlined: changed=%v content=%s", changed, content)
	}
}

func TestInlineManagedTemplateConfigsRejectsWorkspaceEscape(t *testing.T) {
	_, _, err := inlineManagedTemplateConfigs(`services:
  dragonfly:
    image: dragonflydb/dragonfly
configs:
  aether-template-file-0:
    file: ./template-mounts/../../secret
`, t.TempDir())
	if err == nil {
		t.Fatal("workspace escape was accepted")
	}
}

func TestDockerExecuteBuildsExplicitProjectCommand(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "compose.yml")
	if err := os.WriteFile(file, []byte("services:\n  app:\n    image: nginx:alpine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	command := filepath.Join(dir, "docker")
	if err := os.WriteFile(command, []byte("#!/bin/sh\nprintf '%s' \"$*\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	adapter := &Docker{Binary: command, Host: "unix:///var/run/docker.sock"}
	output, err := adapter.Execute(context.Background(), Project{Directory: dir, File: file, EnvFile: filepath.Join(dir, ".env"), Name: "aether-test"}, "config", "--quiet")
	if err != nil {
		t.Fatal(err)
	}
	want := "compose --project-directory " + dir + " --project-name aether-test --env-file " + filepath.Join(dir, ".env") + " -f " + file + " config --quiet"
	if output != want {
		t.Fatalf("command = %q, want %q", output, want)
	}
}

func TestDockerExecuteWithLogsStreamsOutput(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "compose.yml")
	if err := os.WriteFile(file, []byte("services:\n  app:\n    image: nginx:alpine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	command := filepath.Join(dir, "docker")
	if err := os.WriteFile(command, []byte("#!/bin/sh\nprintf 'build one\\nbuild two\\n'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	var lines []string
	adapter := &Docker{Binary: command}
	_, err := adapter.ExecuteWithLogs(context.Background(), Project{Directory: dir, File: file, Name: "aether-test"}, func(line string) {
		lines = append(lines, line)
	}, "compose", "up")
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || lines[0] != "build one\nbuild two" {
		t.Fatalf("streamed lines = %#v", lines)
	}
}

func TestDockerExecuteRetriesTransientNetworkEndpointError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "compose.yml")
	if err := os.WriteFile(file, []byte("services:\n  app:\n    image: nginx:alpine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "attempted")
	command := filepath.Join(dir, "docker")
	script := fmt.Sprintf("#!/bin/sh\nif [ -f %q ]; then\n  printf 'started\\n'\n  exit 0\nfi\ntouch %q\nprintf 'Error response from daemon: Container cannot be created with multiple network endpoints: first, second\\n' >&2\nexit 1\n", marker, marker)
	if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}

	var lines []string
	output, err := (&Docker{Binary: command}).ExecuteWithLogs(context.Background(), Project{Directory: dir, File: file, Name: "aether-test"}, func(line string) {
		lines = append(lines, line)
	}, "up", "-d")
	if err != nil {
		t.Fatal(err)
	}
	if output != "started\n" {
		t.Fatalf("output = %q, want successful retry output", output)
	}
	if strings.Contains(strings.ToLower(strings.Join(lines, "\n")), "multiple network endpoints") {
		t.Fatalf("transient network error leaked to logs: %#v", lines)
	}
}

func TestDetachExternalNetworkAttachments(t *testing.T) {
	content := `version: "3.8"
services:
  web:
    image: nginx:alpine
    networks:
      default: {}
      aether-ingress:
        aliases:
          - app-example-web
  worker:
    image: nginx:alpine
    networks:
      - aether-ingress
networks:
  default: {}
  aether-ingress:
    name: aether-ingress
    external: true
`
	fallback, attachments, err := detachExternalNetworkAttachments(content)
	if err != nil {
		t.Fatal(err)
	}
	if len(attachments) != 2 {
		t.Fatalf("attachments = %#v, want two attachments", attachments)
	}
	if attachments[0].Service != "web" || attachments[0].Network != "aether-ingress" || len(attachments[0].Aliases) != 1 || attachments[0].Aliases[0] != "app-example-web" {
		t.Fatalf("web attachment = %#v", attachments[0])
	}
	if strings.Contains(fallback, "aether-ingress") {
		t.Fatalf("fallback still contains external network: %s", fallback)
	}
	if err := ValidatePolicy(fallback); err != nil {
		t.Fatalf("fallback Compose rejected: %v", err)
	}
}

func TestDockerExecuteFallsBackToNetworkConnect(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "compose.yml")
	content := `services:
  web:
    image: nginx:alpine
    networks:
      aether-ingress:
        aliases:
          - app-example-web
networks:
  aether-ingress:
    name: aether-ingress
    external: true
`
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	command := filepath.Join(dir, "docker")
	script := `#!/bin/sh
args="$*"
case "$args" in
  *"network inspect aether-ingress"*)
    exit 0
    ;;
  *"compose"*".compose-network-fallback-"*" up "*)
    exit 0
    ;;
  *".compose-network-fallback-"*" ps -q web"*)
    printf 'container-id\n'
    exit 0
    ;;
  *"network connect"*)
    exit 0
    ;;
  *"compose"*" up "*)
    printf 'Error response from daemon: Container cannot be created with multiple network endpoints: first, second\n' >&2
    exit 1
    ;;
esac
printf 'unexpected command: %s\n' "$args" >&2
exit 1
`
	if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}

	output, err := (&Docker{Binary: command}).Execute(context.Background(), Project{Directory: dir, File: file, Name: "aether-test"}, "up", "-d")
	if err != nil {
		t.Fatal(err)
	}
	if output != "" {
		t.Fatalf("output = %q, want no fallback control output", output)
	}
}

func TestDockerExecuteRejectsUnsafeComposeBeforeLaunchingRuntime(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "compose.yml")
	if err := os.WriteFile(file, []byte("services:\n  web:\n    image: nginx\n    privileged: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "launched")
	command := filepath.Join(dir, "docker")
	if err := os.WriteFile(command, []byte("#!/bin/sh\ntouch "+marker+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := (&Docker{Binary: command}).Execute(context.Background(), Project{Directory: dir, File: file, Name: "aether-test"}, "compose", "up")
	if err == nil {
		t.Fatal("unsafe compose accepted")
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("runtime launched after policy rejection: %v", statErr)
	}
}

func TestNormalizeUnsupportedExposeRanges(t *testing.T) {
	content, err := normalizeUnsupportedExposeRanges(`services:
  agent:
    expose:
      - 8090
      - 50000-50100/udp
`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, "50000-50100") || !strings.Contains(content, "8090") {
		t.Fatalf("unsupported expose range was not removed: %s", content)
	}
}

func TestMarkExistingNetworksAsExternal(t *testing.T) {
	content := `services:
  api:
    image: nginx:alpine
    networks:
      - shared
      - private
networks:
  shared:
    name: waf_net
  private:
    name: private_net
`
	updated, changed, err := markExistingNetworks(content, func(name string) bool {
		return name == "waf_net"
	})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected network configuration to change")
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(updated), &document); err != nil {
		t.Fatal(err)
	}
	networks := document["networks"].(map[string]any)
	shared := networks["shared"].(map[string]any)
	if shared["external"] != true {
		t.Fatalf("shared network = %#v", shared)
	}
	private := networks["private"].(map[string]any)
	if _, ok := private["external"]; ok {
		t.Fatalf("private network unexpectedly external: %#v", private)
	}
	if _, ok := private["internal"]; ok {
		t.Fatalf("private network unexpectedly internal: %#v", private)
	}
}

func TestMarkExistingNetworksPreservesExplicitInternalNetworks(t *testing.T) {
	content := `networks:
  private:
    internal: true
`
	updated, changed, err := markExistingNetworks(content, func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatalf("explicit internal network should not change: %s", updated)
	}
}

func TestInjectPublishedNetworkPreservesComposeNetworks(t *testing.T) {
	content := `services:
  api:
    image: example/api
  worker:
    image: example/worker
    networks:
      - private
networks:
  private:
    internal: true
`
	updated, changed, err := injectPublishedNetwork(content, "aether-workload-host")
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected published network to be injected")
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(updated), &document); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePolicy(updated); err != nil {
		t.Fatalf("injected network violates compose policy: %v", err)
	}
	networks := document["networks"].(map[string]any)
	published := networks["aether-workload-host"].(map[string]any)
	if published["name"] != "aether-workload-host" || published["external"] != true {
		t.Fatalf("published network = %#v", published)
	}
	private := networks["private"].(map[string]any)
	if private["internal"] != true {
		t.Fatalf("private network lost internal setting: %#v", private)
	}
	services := document["services"].(map[string]any)
	api := services["api"].(map[string]any)
	apiNetworks := api["networks"].(map[string]any)
	if _, ok := apiNetworks["default"]; !ok {
		t.Fatalf("api lost the default network: %#v", apiNetworks)
	}
	if _, ok := apiNetworks["aether-workload-host"]; !ok {
		t.Fatalf("api was not attached to published network: %#v", apiNetworks)
	}
	worker := services["worker"].(map[string]any)
	workerNetworks := worker["networks"].([]any)
	if len(workerNetworks) != 2 || workerNetworks[0] != "private" || workerNetworks[1] != "aether-workload-host" {
		t.Fatalf("worker networks = %#v", workerNetworks)
	}
}

func TestInjectPublishedNetworkPreservesNetworkLists(t *testing.T) {
	content := `services:
  api:
    image: example/api
    networks:
      - default
`
	updated, changed, err := injectPublishedNetwork(content, "aether-workload-host")
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected published network to be injected")
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(updated), &document); err != nil {
		t.Fatal(err)
	}
	service := document["services"].(map[string]any)["api"].(map[string]any)
	networks := service["networks"].([]any)
	if len(networks) != 2 || networks[0] != "default" || networks[1] != "aether-workload-host" {
		t.Fatalf("service networks = %#v", networks)
	}
}
