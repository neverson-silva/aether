package observability

import (
	"strings"
	"testing"
	"time"
)

func TestMetricsTrackJobsCollectionsAndFailures(t *testing.T) {
	metrics := NewMetrics()
	finish := metrics.StartJob("backup.create")
	finish(false)
	failed := metrics.StartJob("restore.execute")
	failed(true)
	metrics.ObserveReconciliation(true)
	metrics.ObservePublish(true)
	metrics.ObserveCollection(2*time.Second, false)
	metrics.ObserveCollection(time.Second, true)

	rendered := metrics.Render()
	for _, expected := range []string{
		"aether_jobs_active 0",
		"aether_jobs_completed_total 1",
		"aether_jobs_failed_total 1",
		"aether_scheduler_reconciliation_errors_total 1",
		"aether_nats_publish_errors_total 1",
		"aether_monitoring_collections_total 2",
		"aether_monitoring_collection_errors_total 1",
		`aether_job_completed_total{type="backup.create"} 1`,
		`aether_job_failed_total{type="restore.execute"} 1`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("metrics output does not contain %q: %s", expected, rendered)
		}
	}
}
