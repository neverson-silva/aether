package http

import (
	"testing"
	"time"

	"aether/internal/platform/worker"
)

func TestLatestRunningContainer(t *testing.T) {
	createdAt := time.Date(2026, time.September, 13, 12, 0, 0, 0, time.UTC)
	items := []worker.ContainerInfo{
		{ID: "old-running", State: "running", CreatedAt: createdAt},
		{ID: "build-container", State: "exited", CreatedAt: createdAt.Add(time.Minute)},
		{ID: "latest-running", State: "running", CreatedAt: createdAt.Add(2 * time.Minute)},
	}

	item, ok := latestRunningContainer(items)
	if !ok {
		t.Fatal("expected a running container")
	}
	if item.ID != "latest-running" {
		t.Fatalf("expected latest-running, got %s", item.ID)
	}
}

func TestLatestRunningContainerWithoutRunningItems(t *testing.T) {
	item, ok := latestRunningContainer([]worker.ContainerInfo{{ID: "stopped", State: "exited"}})
	if ok {
		t.Fatalf("expected no running container, got %s", item.ID)
	}
}

func TestExplicitlyStoppedServiceStatus(t *testing.T) {
	tests := []struct {
		name             string
		storedStatus     string
		activeDeployment bool
		want             bool
	}{
		{name: "manually stopped service", storedStatus: "stopped", want: true},
		{name: "stopped while deployment is active", storedStatus: "stopped", activeDeployment: true, want: false},
		{name: "unexpectedly exited container", storedStatus: "running", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := explicitlyStopped(test.storedStatus, test.activeDeployment); got != test.want {
				t.Fatalf("explicitlyStopped() = %t, want %t", got, test.want)
			}
		})
	}
}
