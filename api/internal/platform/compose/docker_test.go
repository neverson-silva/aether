package compose

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDockerExecuteBuildsExplicitProjectCommand(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "compose.yml")
	if err := os.WriteFile(file, []byte("services: {}\n"), 0o600); err != nil {
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
