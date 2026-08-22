package application

import (
	"sync"

	"aether/internal/modules/monitoring/domain"
)

// History keeps a raw ring buffer of the last 60 minutes of samples as a fast
// in-memory fallback. When a HistoryStore is configured, history is read from
// Postgres (survives API restarts) and this ring is only used as a fallback.
type History struct {
	mu          sync.RWMutex
	aggregate   []domain.HistoryPoint
	perResource map[string][]domain.ResourcePoint
	cap         int
}

const (
	historyCap = 1800 // 1h at 2s sampling
	sampleSecs = 2
	targetPts  = 120
)

var windowSeconds = map[string]int{
	"5m":  300,
	"15m": 900,
	"1h":  3600,
	"6h":  21600,
	"24h": 86400,
	"7d":  604800,
}

func NewHistory() *History {
	return &History{
		aggregate:   make([]domain.HistoryPoint, 0, historyCap),
		perResource: map[string][]domain.ResourcePoint{},
		cap:         historyCap,
	}
}

func (h *History) push(ts int64, points []domain.HistoryPoint, resources []domain.Resource) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(points) > 0 {
		h.aggregate = append(h.aggregate, points...)
		if len(h.aggregate) > h.cap {
			h.aggregate = h.aggregate[len(h.aggregate)-h.cap:]
		}
	}
	for _, r := range resources {
		ser := h.perResource[r.ID]
		ser = append(ser, domain.ResourcePoint{TS: ts, CPU: r.CPUOfHost, Mem: r.MemUsage, NetRx: r.NetRxRate, NetTx: r.NetTxRate})
		if len(ser) > h.cap {
			ser = ser[len(ser)-h.cap:]
		}
		h.perResource[r.ID] = ser
	}
}

func (h *History) AggregateHistory(window string) []domain.HistoryPoint {
	h.mu.RLock()
	raw := h.aggregate
	h.mu.RUnlock()
	return downsampleHistory(raw, window)
}

func (h *History) ResourceHistory(id, window string) []domain.ResourcePoint {
	h.mu.RLock()
	raw := h.perResource[id]
	h.mu.RUnlock()
	return downsampleResource(raw, window)
}

func downsampleHistory(raw []domain.HistoryPoint, window string) []domain.HistoryPoint {
	secs := windowSeconds[window]
	if secs <= 0 {
		return []domain.HistoryPoint{}
	}
	now := int64(0)
	if len(raw) > 0 {
		now = raw[len(raw)-1].TS
	}
	from := now - int64(secs)
	start := 0
	for i, p := range raw {
		if p.TS >= from {
			start = i
			break
		}
	}
	return downsampleHistoryTo(raw[start:], targetPts)
}

func downsampleHistoryTo(raw []domain.HistoryPoint, target int) []domain.HistoryPoint {
	if len(raw) <= target {
		return raw
	}
	step := (len(raw) + target - 1) / target
	out := make([]domain.HistoryPoint, 0, len(raw)/step+1)
	for i := 0; i < len(raw); i += step {
		end := i + step
		if end > len(raw) {
			end = len(raw)
		}
		chunk := raw[i:end]
		var p domain.HistoryPoint
		p.TS = chunk[len(chunk)-1].TS
		for _, c := range chunk {
			p.HostCPU += c.HostCPU
			p.HostMem += c.HostMem
			p.AetherCPU += c.AetherCPU
			p.AetherMem += c.AetherMem
			p.AetherMemPct += c.AetherMemPct
			p.UserCPU += c.UserCPU
			p.UserMem += c.UserMem
			p.UserMemPct += c.UserMemPct
			p.NetRx += c.NetRx
			p.NetTx += c.NetTx
		}
		n := float64(len(chunk))
		p.HostCPU /= n
		p.HostMem /= n
		p.AetherCPU /= n
		p.AetherMemPct /= n
		p.UserCPU /= n
		p.UserMemPct /= n
		p.NetRx /= n
		p.NetTx /= n
		if len(chunk) > 0 {
			p.AetherMem /= uint64(len(chunk))
			p.UserMem /= uint64(len(chunk))
		}
		out = append(out, p)
	}
	return out
}

func downsampleResource(raw []domain.ResourcePoint, window string) []domain.ResourcePoint {
	secs := windowSeconds[window]
	if secs <= 0 {
		return []domain.ResourcePoint{}
	}
	now := int64(0)
	if len(raw) > 0 {
		now = raw[len(raw)-1].TS
	}
	from := now - int64(secs)
	start := 0
	for i, p := range raw {
		if p.TS >= from {
			start = i
			break
		}
	}
	return downsampleResourceTo(raw[start:], targetPts)
}

func downsampleResourceTo(raw []domain.ResourcePoint, target int) []domain.ResourcePoint {
	if len(raw) <= target {
		return raw
	}
	step := (len(raw) + target - 1) / target
	out := make([]domain.ResourcePoint, 0, len(raw)/step+1)
	for i := 0; i < len(raw); i += step {
		end := i + step
		if end > len(raw) {
			end = len(raw)
		}
		chunk := raw[i:end]
		var p domain.ResourcePoint
		p.TS = chunk[len(chunk)-1].TS
		for _, c := range chunk {
			p.CPU += c.CPU
			p.Mem += c.Mem
			p.NetRx += c.NetRx
			p.NetTx += c.NetTx
		}
		n := float64(len(chunk))
		p.CPU /= n
		p.NetRx /= n
		p.NetTx /= n
		if len(chunk) > 0 {
			p.Mem /= uint64(len(chunk))
		}
		out = append(out, p)
	}
	return out
}
