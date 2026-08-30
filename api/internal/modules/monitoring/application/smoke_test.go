package application

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	hostApp "aether/internal/modules/host/application"
	"aether/internal/platform/worker"
)

func TestSmokeCollectReal(t *testing.T) {
	if os.Getenv("AETHER_SMOKE") == "" {
		t.Skip("set AETHER_SMOKE=1 to run against live Docker Engine")
	}
	rt, err := worker.NewDockerRuntime("")
	if err != nil {
		t.Fatalf("create Docker runtime: %v", err)
	}
	defer rt.Close()
	hostSvc := &hostApp.Host{LogsDir: ""}
	m := NewMonitoring(rt, hostSvc, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	m.Collect(ctx)
	time.Sleep(2 * time.Second)
	m.Collect(ctx)
	snap := m.Latest()
	out, _ := json.MarshalIndent(snap, "", "  ")
	os.WriteFile("/tmp/aether-monitoring-smoke.json", out, 0o644)
	t.Logf("resources=%d withStats=%d aetherCPU=%.2f userCPU=%.2f systemCPU=%.2f",
		snap.Collector.Resources, snap.Collector.WithStats, snap.Aether.CPUOfHost, snap.User.CPUOfHost, snap.System.CPUOfHost)
}
