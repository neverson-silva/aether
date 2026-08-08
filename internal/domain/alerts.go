package domain

import "time"

type AlertRule struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	Metric    string    `json:"metric"`
	Threshold float64   `json:"threshold"`
	WindowS   int       `json:"window_s"`
	Severity  string    `json:"severity"`
	Enabled   bool      `json:"enabled"`
	TargetApp string    `json:"target_app"`
	CreatedAt time.Time `json:"created_at"`
}

type AlertEvent struct {
	ID         string    `json:"id"`
	OrgID      string    `json:"org_id"`
	RuleID     string    `json:"rule_id"`
	AppID      string    `json:"app_id"`
	AppName    string    `json:"app_name"`
	Severity   string    `json:"severity"`
	Message    string    `json:"message"`
	Value      float64   `json:"value"`
	Threshold  float64   `json:"threshold"`
	Metric     string    `json:"metric"`
	CreatedAt  time.Time `json:"created_at"`
	ResolvedAt time.Time `json:"resolved_at"`
}

func (e *AlertEvent) Resolved() bool {
	return !e.ResolvedAt.IsZero()
}
