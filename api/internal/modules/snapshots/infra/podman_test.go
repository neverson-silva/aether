package infra

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPodmanExecutorRejectsUnsafeSources(t *testing.T) {
	executor := PodmanExecutor{OutputDir: t.TempDir()}
	if _, _, err := executor.Create(context.Background(), "volume/name", "snapshot"); err == nil {
		t.Fatal("expected unsafe volume to be rejected")
	}
	if _, _, err := executor.Create(context.Background(), "volume", "snapshot/name"); err == nil {
		t.Fatal("expected unsafe snapshot name to be rejected")
	}
}

func TestPodmanExecutorDeleteOnlyRemovesOutputFile(t *testing.T) {
	directory := t.TempDir()
	executor := PodmanExecutor{OutputDir: directory}
	path := filepath.Join(directory, "snapshot.tar.gz")
	if err := os.WriteFile(path, []byte("snapshot"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := executor.Delete(context.Background(), path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected snapshot archive to be removed, got %v", err)
	}
	if err := executor.Delete(context.Background(), filepath.Join(directory, "..", "other")); err == nil {
		t.Fatal("expected path outside output directory to be rejected")
	}
}
