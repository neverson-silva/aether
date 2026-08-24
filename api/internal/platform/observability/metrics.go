package observability

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	mu                   sync.Mutex
	activeJobs           atomic.Int64
	completedJobs        atomic.Int64
	failedJobs           atomic.Int64
	jobDurationNanos     atomic.Int64
	reconciliationErrors atomic.Int64
	publishErrors        atomic.Int64
	collectionCount      atomic.Int64
	collectionErrors     atomic.Int64
	collectionDurationNs atomic.Int64
	jobTypes             map[string]*jobTypeMetrics
}

type jobTypeMetrics struct {
	active    atomic.Int64
	completed atomic.Int64
	failed    atomic.Int64
	duration  atomic.Int64
}

func NewMetrics() *Metrics {
	return &Metrics{jobTypes: make(map[string]*jobTypeMetrics)}
}

func (m *Metrics) ObserveJob(jobType string, started time.Time, failed bool) {
	duration := time.Since(started)
	m.activeJobs.Add(-1)
	m.jobDurationNanos.Add(duration.Nanoseconds())
	if failed {
		m.failedJobs.Add(1)
	} else {
		m.completedJobs.Add(1)
	}
	m.mu.Lock()
	stats := m.jobTypes[jobType]
	if stats == nil {
		stats = &jobTypeMetrics{}
		m.jobTypes[jobType] = stats
	}
	m.mu.Unlock()
	stats.active.Add(-1)
	stats.duration.Add(duration.Nanoseconds())
	if failed {
		stats.failed.Add(1)
	} else {
		stats.completed.Add(1)
	}
}

func (m *Metrics) StartJob(jobType string) func(bool) {
	m.activeJobs.Add(1)
	m.mu.Lock()
	stats := m.jobTypes[jobType]
	if stats == nil {
		stats = &jobTypeMetrics{}
		m.jobTypes[jobType] = stats
	}
	m.mu.Unlock()
	stats.active.Add(1)
	started := time.Now()
	return func(failed bool) { m.ObserveJob(jobType, started, failed) }
}

func (m *Metrics) ObserveReconciliation(failed bool) {
	if failed {
		m.reconciliationErrors.Add(1)
	}
}

func (m *Metrics) ObservePublish(failed bool) {
	if failed {
		m.publishErrors.Add(1)
	}
}

func (m *Metrics) ObserveCollection(duration time.Duration, failed bool) {
	m.collectionCount.Add(1)
	m.collectionDurationNs.Add(duration.Nanoseconds())
	if failed {
		m.collectionErrors.Add(1)
	}
}

func (m *Metrics) Render() string {
	var out string
	out += fmt.Sprintf("aether_jobs_active %d\n", m.activeJobs.Load())
	out += fmt.Sprintf("aether_jobs_completed_total %d\n", m.completedJobs.Load())
	out += fmt.Sprintf("aether_jobs_failed_total %d\n", m.failedJobs.Load())
	out += fmt.Sprintf("aether_jobs_duration_seconds_total %.6f\n", float64(m.jobDurationNanos.Load())/float64(time.Second))
	out += fmt.Sprintf("aether_scheduler_reconciliation_errors_total %d\n", m.reconciliationErrors.Load())
	out += fmt.Sprintf("aether_nats_publish_errors_total %d\n", m.publishErrors.Load())
	out += fmt.Sprintf("aether_monitoring_collections_total %d\n", m.collectionCount.Load())
	out += fmt.Sprintf("aether_monitoring_collection_errors_total %d\n", m.collectionErrors.Load())
	out += fmt.Sprintf("aether_monitoring_collection_duration_seconds_total %.6f\n", float64(m.collectionDurationNs.Load())/float64(time.Second))
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, stats := range m.jobTypes {
		out += fmt.Sprintf("aether_job_active{type=%q} %d\n", name, stats.active.Load())
		out += fmt.Sprintf("aether_job_completed_total{type=%q} %d\n", name, stats.completed.Load())
		out += fmt.Sprintf("aether_job_failed_total{type=%q} %d\n", name, stats.failed.Load())
		out += fmt.Sprintf("aether_job_duration_seconds_total{type=%q} %.6f\n", name, float64(stats.duration.Load())/float64(time.Second))
	}
	return out
}
