package messaging

import "time"

type Envelope struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	SchemaVersion int       `json:"schema_version"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	CausationID   string    `json:"causation_id,omitempty"`
	OrgID         string    `json:"org_id,omitempty"`
	ResourceID    string    `json:"resource_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	Payload       []byte    `json:"payload"`
}
