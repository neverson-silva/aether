package nats

import (
	"encoding/json"
	"testing"
	"time"

	"aether/internal/platform/druntime/queue"
	"aether/internal/platform/messaging"
)

func TestDecodeJobEnvelope(t *testing.T) {
	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	payload, err := json.Marshal(queue.Job{Payload: []byte(`{"value":"ok"}`)})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(messaging.Envelope{
		ID:            "job-1",
		Type:          "backup.create",
		SchemaVersion: 1,
		OrgID:         "org-1",
		ResourceID:    "resource-1",
		CreatedAt:     createdAt,
		Payload:       payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	job, err := decodeJob(raw)
	if err != nil {
		t.Fatal(err)
	}
	if job.ID != "job-1" || job.Type != "backup.create" || job.OrgID != "org-1" || job.DeploymentID != "resource-1" {
		t.Fatalf("unexpected decoded job: %+v", job)
	}
	if !job.CreatedAt.Equal(createdAt) || string(job.Payload) != `{"value":"ok"}` {
		t.Fatalf("unexpected decoded metadata: %+v", job)
	}
}

func TestDecodeLegacyJob(t *testing.T) {
	raw, err := json.Marshal(queue.Job{ID: "legacy-1", Type: "cleanup"})
	if err != nil {
		t.Fatal(err)
	}
	job, err := decodeJob(raw)
	if err != nil {
		t.Fatal(err)
	}
	if job.ID != "legacy-1" || job.Type != "cleanup" {
		t.Fatalf("unexpected legacy job: %+v", job)
	}
}

func TestValidateServerVersion(t *testing.T) {
	for _, version := range []string{"2.14.0", "2.14.2", "v2.15.1"} {
		if err := validateServerVersion(version); err != nil {
			t.Fatalf("version %s should be supported: %v", version, err)
		}
	}
	for _, version := range []string{"2.13.9", "1.0.0", "invalid"} {
		if err := validateServerVersion(version); err == nil {
			t.Fatalf("version %s should be rejected", version)
		}
	}
}
