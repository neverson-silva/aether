package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const agentFixture = `{"ts":1787049000,"source":"host-agent","cpu_percent":16.8,"cpu_cores":10,` +
	`"mem_total":25769803776,"mem_used":19120000000,"mem_percent":74.2,` +
	`"disk_total":994662584320,"disk_used":12636901376,"disk_percent":1.3,` +
	`"net_rx_bytes":9464094501,"net_tx_bytes":3429158092,"load":[3.01,2.72,2.38],` +
	`"uptime":121984,"hostname":"MacBook-Pro-de-Neverson.local","os":"darwin 26.6.1"}`

func writeAgent(t *testing.T, content string, modTime time.Time) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "host-stats.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, modTime, modTime); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAgentStatsFreshUsesHostValues(t *testing.T) {
	h := &Host{AgentFile: writeAgent(t, agentFixture, time.Now())}
	stats := h.Stats(context.Background())
	if stats.Source != "host-agent" {
		t.Fatalf("source = %q, want mac-host-agent", stats.Source)
	}
	if stats.MemTotal != 25769803776 {
		t.Fatalf("mem total = %d, want Mac total", stats.MemTotal)
	}
	if stats.Disk.Total != 994662584320 {
		t.Fatalf("disk total = %d, want Mac disk", stats.Disk.Total)
	}
	if stats.CPUCores != 10 {
		t.Fatalf("cpu cores = %d, want 10", stats.CPUCores)
	}
	if stats.Hostname != "MacBook-Pro-de-Neverson.local" {
		t.Fatalf("hostname = %q", stats.Hostname)
	}
	if stats.RuntimeCores == 0 {
		t.Fatal("runtime cores must be populated for container CPU normalization")
	}
}

func TestAgentStatsStaleFallsBackToRuntime(t *testing.T) {
	h := &Host{AgentFile: writeAgent(t, agentFixture, time.Now().Add(-5*time.Minute))}
	stats := h.Stats(context.Background())
	if stats.Source != "runtime" {
		t.Fatalf("source = %q, want runtime fallback", stats.Source)
	}
	if stats.CPUCores == 0 || stats.MemTotal == 0 {
		t.Fatalf("fallback stats empty: %+v", stats)
	}
}

func TestAgentStatsMissingFallsBack(t *testing.T) {
	h := &Host{AgentFile: filepath.Join(t.TempDir(), "nope.json")}
	stats := h.Stats(context.Background())
	if stats.Source != "runtime" {
		t.Fatalf("source = %q, want runtime", stats.Source)
	}
}

func TestAgentStatsEmptyAgentFileConfig(t *testing.T) {
	h := &Host{}
	stats := h.Stats(context.Background())
	if stats.Source != "runtime" {
		t.Fatalf("source = %q, want runtime", stats.Source)
	}
}

func TestAgentStatsInvalidJSONFallsBack(t *testing.T) {
	h := &Host{AgentFile: writeAgent(t, "not json", time.Now())}
	stats := h.Stats(context.Background())
	if stats.Source != "runtime" {
		t.Fatalf("source = %q, want runtime", stats.Source)
	}
}
