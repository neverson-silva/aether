package core

import (
	"context"
	"fmt"
	"time"

	"aether/internal/domain"
)

func (c *Core) CreateAlertRule(r *domain.AlertRule) error {
	if r.ID == "" {
		r.ID = domain.NewID()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	if r.WindowS <= 0 {
		r.WindowS = 30
	}
	enabled := 0
	if r.Enabled {
		enabled = 1
	}
	switch r.Metric {
	case "cpu", "memory", "memory_pct", "disk":
	default:
		return fmt.Errorf("metric inválida (cpu|memory|memory_pct|disk)")
	}
	switch r.Severity {
	case "info", "warning", "critical":
	default:
		r.Severity = "warning"
	}
	_, err := c.DB.Exec(`INSERT INTO alert_rules(id,org_id,name,metric,threshold,window_s,severity,enabled,target_app,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		r.ID, r.OrgID, r.Name, r.Metric, r.Threshold, r.WindowS, r.Severity, enabled, r.TargetApp,
		r.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (c *Core) ListAlertRules(orgID string) ([]domain.AlertRule, error) {
	rows, err := c.DB.Query(`SELECT id,org_id,name,metric,threshold,window_s,severity,enabled,target_app,created_at FROM alert_rules WHERE org_id=? ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AlertRule{}
	for rows.Next() {
		var r domain.AlertRule
		var enabled int
		var created string
		if err := rows.Scan(&r.ID, &r.OrgID, &r.Name, &r.Metric, &r.Threshold, &r.WindowS, &r.Severity, &enabled, &r.TargetApp, &created); err != nil {
			return nil, err
		}
		r.Enabled = enabled == 1
		r.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (c *Core) SetAlertRuleEnabled(id string, enabled bool) error {
	v := 0
	if enabled {
		v = 1
	}
	_, err := c.DB.Exec(`UPDATE alert_rules SET enabled=? WHERE id=?`, v, id)
	return err
}

func (c *Core) DeleteAlertRule(id string) error {
	_, err := c.DB.Exec(`DELETE FROM alert_rules WHERE id=?`, id)
	return err
}

func (c *Core) ListAlertEvents(orgID string, limit int) ([]domain.AlertEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := c.DB.Query(`SELECT id,org_id,rule_id,app_id,app_name,severity,message,value,threshold,metric,created_at,resolved_at FROM alert_events WHERE org_id=? ORDER BY created_at DESC LIMIT ?`, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AlertEvent{}
	for rows.Next() {
		var e domain.AlertEvent
		var created, resolved string
		if err := rows.Scan(&e.ID, &e.OrgID, &e.RuleID, &e.AppID, &e.AppName, &e.Severity, &e.Message, &e.Value, &e.Threshold, &e.Metric, &created, &resolved); err != nil {
			return nil, err
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, created)
		e.ResolvedAt, _ = time.Parse(time.RFC3339, resolved)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (c *Core) ResolveAlertEvent(id string) error {
	_, err := c.DB.Exec(`UPDATE alert_events SET resolved_at=? WHERE id=?`, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (c *Core) EvaluateAlertRules(ctx context.Context) error {
	orgs, err := c.Store.ListOrgs()
	if err != nil {
		return err
	}
	for _, org := range orgs {
		rules, err := c.ListAlertRules(org.ID)
		if err != nil {
			continue
		}
		for _, rule := range rules {
			if !rule.Enabled {
				continue
			}
			apps := []domain.App{}
			if rule.TargetApp != "" {
				app, err := c.Store.GetApp(rule.TargetApp)
				if err == nil {
					apps = []domain.App{*app}
				}
			} else {
				apps, _ = c.Store.ListApps(org.ID)
			}
			for _, app := range apps {
				c.evaluateRule(ctx, rule, app)
			}
		}
	}
	return nil
}

func (c *Core) evaluateRule(ctx context.Context, rule domain.AlertRule, app domain.App) {
	deploys, err := c.Store.ListDeployments(app.ID, 1)
	if err != nil || len(deploys) == 0 || deploys[0].ContainerID == "" {
		return
	}
	sctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	stats, err := c.Driver.Stats(sctx, deploys[0].ContainerID)
	if err != nil || stats.MemBytes <= 0 {
		return
	}
	var value float64
	switch rule.Metric {
	case "cpu":
		value = stats.CPUPercent
	case "memory_pct":
		if stats.MemLimit > 0 {
			value = float64(stats.MemBytes) / float64(stats.MemLimit) * 100
		} else {
			return
		}
	case "memory":
		value = float64(stats.MemBytes) / (1024 * 1024)
	default:
		return
	}
	if value < rule.Threshold {
		return
	}
	var resolved string
	err = c.DB.QueryRow(`SELECT resolved_at FROM alert_events WHERE org_id=? AND rule_id=? AND app_id=? AND resolved_at='' ORDER BY created_at DESC LIMIT 1`,
		rule.OrgID, rule.ID, app.ID).Scan(&resolved)
	if err == nil {
		return
	}
	label := map[string]string{"cpu": "CPU", "memory": "Memory (MiB)", "memory_pct": "Memory %"}[rule.Metric]
	msg := fmt.Sprintf("%s usage %.1f%s above threshold %.1f%s for %s", label, value, unitSuffix(rule.Metric), rule.Threshold, unitSuffix(rule.Metric), app.Name)
	e := &domain.AlertEvent{
		ID:        domain.NewID(),
		OrgID:     rule.OrgID,
		RuleID:    rule.ID,
		AppID:     app.ID,
		AppName:   app.Name,
		Severity:  rule.Severity,
		Message:   msg,
		Value:     value,
		Threshold: rule.Threshold,
		Metric:    rule.Metric,
		CreatedAt: time.Now().UTC(),
	}
	_, _ = c.DB.Exec(`INSERT INTO alert_events(id,org_id,rule_id,app_id,app_name,severity,message,value,threshold,metric,created_at,resolved_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		e.ID, e.OrgID, e.RuleID, e.AppID, e.AppName, e.Severity, e.Message, e.Value, e.Threshold, e.Metric,
		e.CreatedAt.UTC().Format(time.RFC3339), "")
	c.notify.emit(rule.OrgID, "alert."+rule.Severity, msg, map[string]any{
		"app_id": app.ID, "rule_id": rule.ID, "metric": rule.Metric, "value": e.Value, "threshold": e.Threshold,
	})
}

func unitSuffix(metric string) string {
	switch metric {
	case "cpu":
		return "%"
	case "memory_pct":
		return "%"
	default:
		return ""
	}
}

func (c *Core) StartAlertLoop(ctx context.Context) {
	interval := 30 * time.Second
	if env := c.Cfg.AlertIntervalSeconds; env > 0 {
		interval = time.Duration(env) * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.EvaluateAlertRules(context.Background()); err != nil {
					_ = err
				}
			}
		}
	}()
}
