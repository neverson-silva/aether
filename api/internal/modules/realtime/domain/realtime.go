package domain

import (
	"encoding/json"
	"errors"
	"time"

	"aether/internal/platform/druntime/queue"
)

var (
	ErrValidation = errors.New("invalid input")
	ErrForbidden  = errors.New("access denied")
)

type Event struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Version       int             `json:"version"`
	Aggregate     string          `json:"aggregate_type"`
	Message       string          `json:"message"`
	Payload       json.RawMessage `json:"payload"`
	TS            time.Time       `json:"ts"`
	OrgID         string          `json:"org_id,omitempty"`
	ProjectID     string          `json:"project_id,omitempty"`
	AppID         string          `json:"app_id,omitempty"`
	ServiceID     string          `json:"service_id,omitempty"`
	ResourceType  string          `json:"resource_type,omitempty"`
	ResourceID    string          `json:"resource_id,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	CausationID   string          `json:"causation_id,omitempty"`
	Seq           int64           `json:"seq"`
	Ephemeral     bool            `json:"-"`
}

func ParseEvent(data []byte) Event {
	var e Event
	_ = json.Unmarshal(data, &e)
	return e
}

type Metrics struct {
	Backend       string                   `json:"backend"`
	CacheHits     int64                    `json:"cache_hits"`
	CacheMisses   int64                    `json:"cache_misses"`
	CacheSets     int64                    `json:"cache_sets"`
	CacheErrors   int64                    `json:"cache_errors"`
	Subscribers   map[string]int           `json:"subscribers"`
	TotalChannels int                      `json:"total_channels"`
	Queues        map[string]queue.Metrics `json:"queues,omitempty"`
}

type NetSample struct {
	At time.Time `json:"at"`
	Ms float64   `json:"ms"`
	OK bool      `json:"ok"`
	H3 bool      `json:"h3"`
}

type NetAppStat struct {
	ServiceID string      `json:"service_id"`
	AppID     string      `json:"app_id,omitempty"`
	Name      string      `json:"name"`
	Addr      string      `json:"addr"`
	Samples   []NetSample `json:"samples"`
	P50       float64     `json:"p50_ms"`
	P95       float64     `json:"p95_ms"`
	Uptime    float64     `json:"uptime_pct"`
	H3        bool        `json:"http3"`
}
