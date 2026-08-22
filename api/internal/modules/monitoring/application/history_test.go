package application

import (
	"testing"

	"aether/internal/modules/monitoring/domain"
)

func TestHistorySmallSampleReturnsAll(t *testing.T) {
	h := NewHistory()
	for i := 0; i < 10; i++ {
		h.push(int64(1000+i*2), []domain.HistoryPoint{
			{TS: int64(1000 + i*2), HostCPU: float64(i), HostMem: 50, NetRx: 100},
		}, nil)
	}
	pts := h.AggregateHistory("5m")
	if len(pts) != 10 {
		t.Fatalf("expected all 10 points when under target, got %d", len(pts))
	}
	if pts[0].HostCPU != 0 || pts[9].HostCPU != 9 {
		t.Fatalf("points out of order: first=%v last=%v", pts[0].HostCPU, pts[9].HostCPU)
	}
}

func TestHistoryDownsampleBounded(t *testing.T) {
	h := NewHistory()
	now := int64(1_700_000_000)
	for i := 0; i < 500; i++ {
		h.push(now+int64(i*2), []domain.HistoryPoint{{TS: now + int64(i*2), HostCPU: float64(i % 100)}}, nil)
	}
	pts := h.AggregateHistory("15m")
	if len(pts) > targetPts {
		t.Fatalf("downsample exceeded target: %d", len(pts))
	}
	if len(pts) == 0 {
		t.Fatal("expected points")
	}
}

func TestHistoryWindowTrims(t *testing.T) {
	h := NewHistory()
	base := int64(1_000_000_000)
	for i := 0; i < 200; i++ {
		h.push(base+int64(i*2), []domain.HistoryPoint{{TS: base + int64(i*2), HostCPU: float64(i)}}, nil)
	}
	pts := h.AggregateHistory("5m")
	for _, p := range pts {
		if p.TS < base+int64(2*50) {
			t.Fatalf("point outside window: %d", p.TS)
		}
	}
}

func TestHistoryResourceSeries(t *testing.T) {
	h := NewHistory()
	rs := []domain.Resource{resource("r1", "user", "running", 5, 100)}
	h.push(1000, []domain.HistoryPoint{}, rs)
	h.push(1002, []domain.HistoryPoint{}, rs)
	pts := h.ResourceHistory("r1", "5m")
	if len(pts) != 2 {
		t.Fatalf("expected 2 points, got %d", len(pts))
	}
	if pts[0].CPU != 5 {
		t.Fatalf("expected cpu 5, got %v", pts[0].CPU)
	}
}

func TestHistoryEmptyWindows(t *testing.T) {
	h := NewHistory()
	if pts := h.AggregateHistory("5m"); len(pts) != 0 {
		t.Fatal("expected empty aggregate history")
	}
	if pts := h.ResourceHistory("nope", "1h"); len(pts) != 0 {
		t.Fatal("expected empty resource history")
	}
}

func TestWindowSecondsKnown(t *testing.T) {
	for _, w := range []string{"5m", "15m", "1h", "6h", "24h", "7d"} {
		if windowSeconds[w] <= 0 {
			t.Fatalf("missing window %s", w)
		}
	}
}
