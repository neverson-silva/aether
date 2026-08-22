package application

import (
	"context"
	"testing"
)

func TestStatsPopulated(t *testing.T) {
	h := &Host{}
	stats := h.Stats(context.Background())
	if stats.Hostname == "" || stats.CPUCores == 0 {
		t.Fatalf("stats inesperado: %+v", stats)
	}
	if stats.MemTotal == 0 || stats.Uptime == 0 {
		t.Fatalf("mem/uptime zerados: %+v", stats)
	}
}

func TestParseLogLine(t *testing.T) {
	ts, typ, detail := parseLogLine("2026-08-08 12:00:00 INFO request done")
	if typ != "INFO" || detail != "request done" || ts == "" {
		t.Fatalf("parse inesperado: ts=%q typ=%q detail=%q", ts, typ, detail)
	}
}
