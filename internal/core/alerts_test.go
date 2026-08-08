package core

import (
	"aether/internal/domain"
	"context"
	"testing"
)

func TestAlertRulesCRUDAndEvaluation(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	user, org, err := c.CreateUserAndOrg("alerts@aether.local", "alerts", "senha-alerts")
	if err != nil {
		t.Fatal(err)
	}
	_ = user
	rule := &domain.AlertRule{
		OrgID:     org.ID,
		Name:      "high cpu",
		Metric:    "cpu",
		Threshold: 1000,
		WindowS:   30,
		Severity:  "warning",
		Enabled:   true,
	}
	if err := c.CreateAlertRule(rule); err != nil {
		t.Fatal(err)
	}
	rules, err := c.ListAlertRules(org.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].Metric != "cpu" {
		t.Fatalf("rules: %v", rules)
	}
	// threshold absurdo -> nunca dispara; avaliação deve ser inofensiva
	if err := c.EvaluateAlertRules(context.Background()); err != nil {
		t.Fatal(err)
	}
	events, err := c.ListAlertEvents(org.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("não deveria haver eventos: %v", events)
	}
	if err := c.DeleteAlertRule(rule.ID); err != nil {
		t.Fatal(err)
	}
	rules, _ = c.ListAlertRules(org.ID)
	if len(rules) != 0 {
		t.Fatal("regra não deletada")
	}
}
