package config

import "testing"

func TestLoadUsesDedicatedBuildDockerHost(t *testing.T) {
	t.Setenv("AETHER_STATE", t.TempDir())
	t.Setenv("AETHER_DOCKER_HOST", "unix:///legacy.sock")
	t.Setenv("AETHER_BUILD_DOCKER_HOST", "unix:///docker.sock")
	t.Setenv("DOCKER_HOST", "unix:///environment.sock")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DockerHost != "unix:///legacy.sock" {
		t.Fatalf("DockerHost = %q", cfg.DockerHost)
	}
	if cfg.BuildDockerHost != "unix:///docker.sock" {
		t.Fatalf("BuildDockerHost = %q", cfg.BuildDockerHost)
	}
}

func TestLoadDefaultsBuildDockerHostToDockerSocket(t *testing.T) {
	t.Setenv("AETHER_STATE", t.TempDir())
	t.Setenv("AETHER_DOCKER_HOST", "")
	t.Setenv("AETHER_BUILD_DOCKER_HOST", "")
	t.Setenv("AETHER_IMAGE_DOCKER_HOST", "")
	t.Setenv("DOCKER_HOST", "unix:///legacy.sock")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BuildDockerHost != "unix:///var/run/docker.sock" {
		t.Fatalf("BuildDockerHost = %q", cfg.BuildDockerHost)
	}
}
