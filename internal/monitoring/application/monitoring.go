package application

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	hostdomain "aether/internal/host/domain"
	"aether/internal/monitoring/domain"

	"aether/internal/worker"
)

// Runtime is the narrow podman-facing dependency of the collector.
type Runtime interface {
	ListContainers(ctx context.Context) ([]worker.ContainerInfo, error)
}

type HostStats interface {
	Stats(ctx context.Context) hostdomain.Stats
}

// HistoryStore persists samples so history survives API restarts. When nil,
// the in-memory ring buffer is used as a fallback.
type HistoryStore interface {
	InsertSample(ctx context.Context, p domain.HistoryPoint, ts time.Time) error
	InsertResourceSample(ctx context.Context, r domain.ResourcePoint, id, name, owner string, ts time.Time) error
	ListSamples(ctx context.Context, from, to time.Time) ([]domain.HistoryPoint, error)
	ListResourceSamples(ctx context.Context, id string, from, to time.Time) ([]domain.ResourcePoint, error)
	Purge(ctx context.Context, before time.Time) error
}

type prevCounters struct {
	ts       time.Time
	netIn    uint64
	netOut   uint64
	blockIn  uint64
	blockOut uint64
}

type hostPrev struct {
	ts  time.Time
	rx  uint64
	tx  uint64
	cpu float64
}

// Monitoring collects container metrics in a single batch, classifies
// ownership, aggregates Aether/User/System and keeps history (persisted when a
// HistoryStore is configured).
type Monitoring struct {
	runtime Runtime
	host    HostStats
	history *History
	store   HistoryStore
	logger  *slog.Logger

	mu        sync.RWMutex
	latest    *domain.Snapshot
	prev      map[string]prevCounters
	hostPrev  hostPrev
	cores     int
	collectN  int64
	errN      int64
	lastErr   string
	lastMS    float64
	upSince   time.Time
	tick      int64

	storageMu        sync.RWMutex
	storageCache     map[string]uint64
	lastStorageScan  time.Time
}

func NewMonitoring(runtime Runtime, host HostStats, logger *slog.Logger, store HistoryStore) *Monitoring {
	if logger == nil {
		logger = slog.Default()
	}
	return &Monitoring{
		runtime: runtime,
		host:    host,
		history: NewHistory(),
		store:   store,
		logger:  logger,
		prev:    map[string]prevCounters{},
		upSince: time.Now(),
		storageCache: map[string]uint64{},
	}
}

// Run starts the collector loop. It never returns until ctx is cancelled.
func (m *Monitoring) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.Collect(ctx)
		}
	}
}

// Collect performs one sampling cycle (host + all containers in batch).
func (m *Monitoring) Collect(ctx context.Context) {
	start := time.Now()
	hs := m.host.Stats(ctx)

	raw, err := m.runtime.ListContainers(ctx)
	if err != nil {
		m.mu.Lock()
		m.errN++
		m.lastErr = "list containers: " + err.Error()
		m.lastMS = float64(time.Since(start).Milliseconds())
		if m.latest == nil {
			m.latest = m.emptySnapshot(hs)
		}
		m.mu.Unlock()
		m.logger.Warn("monitoring: list containers", "err", err)
		return
	}

	cores := hs.RuntimeCores
	if cores < 1 {
		cores = hs.CPUCores
	}
	if cores < 1 {
		cores = 1
	}

	now := time.Now()

	// Storage attribution is expensive (podman system df -v computes writable
	// layers), so it runs on a slow cadence and is cached between scans.
	if now.Sub(m.lastStorageScan) >= 5*time.Minute {
		sctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		sizes, err := m.storageScan(sctx)
		cancel()
		if err == nil {
			m.storageMu.Lock()
			m.storageCache = sizes
			m.lastStorageScan = now
			m.storageMu.Unlock()
		} else {
			m.logger.Warn("monitoring: storage scan", "err", err)
		}
	}

	m.mu.RLock()
	prevHostSample := m.hostPrev
	prev := make(map[string]prevCounters, len(m.prev))
	for k, v := range m.prev {
		prev[k] = v
	}
	m.mu.RUnlock()

	m.storageMu.RLock()
	storageCache := m.storageCache
	m.storageMu.RUnlock()

	hostNetRx, hostNetTx := 0.0, 0.0
	if !prevHostSample.ts.IsZero() && now.After(prevHostSample.ts) {
		dt := now.Sub(prevHostSample.ts).Seconds()
		if dt > 0 {
			hostNetRx = rate(hs.Net.RxBytes, prevHostSample.rx, dt)
			hostNetTx = rate(hs.Net.TxBytes, prevHostSample.tx, dt)
		}
	}

	resources := make([]domain.Resource, 0, len(raw))
	withStats := 0
	for _, c := range raw {
		owner, st, sid, pid, name := Classify(rawContainer{
			ID: c.ID, Name: c.Name, State: c.State, Labels: c.Labels,
			CPU: c.Stats.CPUPercent, MemUsage: c.Stats.MemUsage, MemLimit: c.Stats.MemLimit,
			MemPerc: c.Stats.MemPercent, NetIn: c.Stats.NetInput, NetOut: c.Stats.NetOutput,
			BlockIn: c.Stats.BlockInput, BlockOut: c.Stats.BlockOutput, HasStats: c.HasStats,
		})
		r := domain.Resource{
			ID: c.ID, Name: name, Owner: owner, ServiceType: st, ServiceID: sid, ProjectID: pid,
			State: c.State, Active: isActive(c.State), HasStats: c.HasStats,
			CPUPercent: c.Stats.CPUPercent, CPUOfHost: c.Stats.CPUPercent / float64(cores),
			MemUsage: c.Stats.MemUsage, MemLimit: c.Stats.MemLimit, MemPercent: c.Stats.MemPercent,
			NetInput: c.Stats.NetInput, NetOutput: c.Stats.NetOutput,
			BlockInput: c.Stats.BlockInput, BlockOutput: c.Stats.BlockOutput,
		}
		if sz, ok := storageCache[c.Name]; ok {
			s := sz
			r.Storage = &s
		}
		if c.HasStats {
			withStats++
		}
		if p, ok := prev[c.ID]; ok && !p.ts.IsZero() && now.After(p.ts) {
			dt := now.Sub(p.ts).Seconds()
			if dt > 0 {
				r.NetRxRate = rate(c.Stats.NetInput, p.netIn, dt)
				r.NetTxRate = rate(c.Stats.NetOutput, p.netOut, dt)
				r.BlockRxRate = rate(c.Stats.BlockInput, p.blockIn, dt)
				r.BlockTxRate = rate(c.Stats.BlockOutput, p.blockOut, dt)
				r.HasNetRate = true
				r.HasBlockRate = true
			}
		}
		resources = append(resources, r)
		m.mu.Lock()
		m.prev[c.ID] = prevCounters{ts: now, netIn: c.Stats.NetInput, netOut: c.Stats.NetOutput, blockIn: c.Stats.BlockInput, blockOut: c.Stats.BlockOutput}
		m.mu.Unlock()
	}

	sort.Slice(resources, func(i, j int) bool {
		ai, aj := resources[i].Active, resources[j].Active
		if ai != aj {
			return ai
		}
		return resources[i].CPUOfHost > resources[j].CPUOfHost
	})

	host := domain.Host{
		CPUPercent: hs.CPUPercent, CPUCores: hs.CPUCores, RuntimeCores: hs.RuntimeCores,
		MemTotal: hs.MemTotal, MemUsed: hs.MemUsed, MemPercent: hs.MemPercent,
		DiskTotal: hs.Disk.Total, DiskUsed: hs.Disk.Used, DiskPercent: hs.Disk.Percent,
		NetRxRate: hostNetRx, NetTxRate: hostNetTx,
		Load: hs.Load, Uptime: hs.Uptime, Hostname: hs.Hostname, OS: hs.OS,
		Source: hs.Source,
	}
	aether, user, system := Aggregate(resources, host)

	collectMS := float64(time.Since(start).Milliseconds())
	m.mu.Lock()
	m.collectN++
	m.lastMS = collectMS
	m.hostPrev = hostPrev{ts: now, rx: hs.Net.RxBytes, tx: hs.Net.TxBytes, cpu: hs.CPUPercent}
	m.latest = &domain.Snapshot{
		TS: now, Host: host, Aether: aether, User: user, System: system,
		Resources: resources,
		Collector: domain.CollectorStats{
			CollectCount: m.collectN, ErrorCount: m.errN, LastCollectMS: collectMS,
			Resources: len(resources), WithStats: withStats, LastError: m.lastErr, UpSince: m.upSince.Format(time.RFC3339),
		},
	}
	m.mu.Unlock()

	m.history.push(now.Unix(), []domain.HistoryPoint{{
		TS: now.Unix(), HostCPU: host.CPUPercent, HostMem: host.MemPercent,
		AetherCPU: aether.CPUOfHost, AetherMem: aether.MemUsage, AetherMemPct: aether.MemPercent,
		UserCPU: user.CPUOfHost, UserMem: user.MemUsage, UserMemPct: user.MemPercent,
		NetRx: hostNetRx, NetTx: hostNetTx,
	}}, resources)

	m.persist(now, host.CPUPercent, host.MemPercent, aether, user, hostNetRx, hostNetTx, resources)
}

func (m *Monitoring) persist(now time.Time, hostCPU, hostMem float64, aether, user domain.Aggregate, hostNetRx, hostNetTx float64, resources []domain.Resource) {
	if m.store == nil {
		return
	}
	m.mu.Lock()
	m.tick++
	tick := m.tick
	m.mu.Unlock()

	pctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.store.InsertSample(pctx, domain.HistoryPoint{
		TS: now.Unix(), HostCPU: hostCPU, HostMem: hostMem,
		AetherCPU: aether.CPUOfHost, AetherMem: aether.MemUsage, AetherMemPct: aether.MemPercent,
		UserCPU: user.CPUOfHost, UserMem: user.MemUsage, UserMemPct: user.MemPercent,
		NetRx: hostNetRx, NetTx: hostNetTx,
	}, now); err != nil {
		m.logger.Warn("monitoring: persist aggregate", "err", err)
	}
	if tick%5 == 0 {
		for _, r := range resources {
			if !r.Active || !r.HasStats {
				continue
			}
			if err := m.store.InsertResourceSample(pctx, domain.ResourcePoint{
				TS: now.Unix(), CPU: r.CPUOfHost, Mem: r.MemUsage, NetRx: r.NetRxRate, NetTx: r.NetTxRate,
			}, r.ID, r.Name, r.Owner, now); err != nil {
				m.logger.Warn("monitoring: persist resource", "id", r.ID, "err", err)
			}
		}
	}
	if tick%450 == 0 {
		if err := m.store.Purge(pctx, now.Add(-7*24*time.Hour)); err != nil {
			m.logger.Warn("monitoring: purge", "err", err)
		}
	}
}

func (m *Monitoring) emptySnapshot(hs hostdomain.Stats) *domain.Snapshot {
	return &domain.Snapshot{
		TS: time.Now(),
		Host: domain.Host{
			CPUPercent: hs.CPUPercent, CPUCores: hs.CPUCores, RuntimeCores: hs.RuntimeCores,
			MemTotal: hs.MemTotal, MemUsed: hs.MemUsed, MemPercent: hs.MemPercent,
			DiskTotal: hs.Disk.Total, DiskUsed: hs.Disk.Used, DiskPercent: hs.Disk.Percent,
			Load: hs.Load, Uptime: hs.Uptime, Hostname: hs.Hostname, OS: hs.OS,
			Source: hs.Source,
		},
		System: domain.Aggregate{Available: true},
		Resources: []domain.Resource{},
		Collector: domain.CollectorStats{CollectCount: m.collectN, ErrorCount: m.errN, Resources: 0, UpSince: m.upSince.Format(time.RFC3339)},
	}
}

// Latest returns the most recent snapshot.
func (m *Monitoring) Latest() *domain.Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.latest == nil {
		return &domain.Snapshot{TS: time.Now(), System: domain.Aggregate{Available: true}}
	}
	return m.latest
}

func (m *Monitoring) History(window string) []domain.HistoryPoint {
	secs := windowSeconds[window]
	if secs <= 0 {
		return []domain.HistoryPoint{}
	}
	if m.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		now := time.Now()
		pts, err := m.store.ListSamples(ctx, now.Add(-time.Duration(secs)*time.Second), now)
		if err == nil {
			return downsampleHistoryTo(pts, targetPts)
		}
		m.logger.Warn("monitoring: history from store", "err", err)
	}
	return m.history.AggregateHistory(window)
}

func (m *Monitoring) ResourceHistory(id, window string) []domain.ResourcePoint {
	secs := windowSeconds[window]
	if secs <= 0 {
		return []domain.ResourcePoint{}
	}
	if m.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		now := time.Now()
		pts, err := m.store.ListResourceSamples(ctx, id, now.Add(-time.Duration(secs)*time.Second), now)
		if err == nil {
			return downsampleResourceTo(pts, targetPts)
		}
		m.logger.Warn("monitoring: resource history from store", "id", id, "err", err)
	}
	return m.history.ResourceHistory(id, window)
}

func (m *Monitoring) CollectorStats() domain.CollectorStats {
	s := m.Latest()
	return s.Collector
}

func rate(cur, prev uint64, dt float64) float64 {
	if cur < prev {
		return 0
	}
	return float64(cur-prev) / dt
}
