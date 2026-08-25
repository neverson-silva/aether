package domain

import "testing"

func TestEvaluateTriggerMatchesServiceAndSharedPaths(t *testing.T) {
	decision := EvaluateTrigger(BuildTriggerRules{
		Branch:        "main",
		AutoDeploy:    true,
		RootDirectory: "apps/web",
		WatchPaths:    []string{"packages/ui/**"},
	}, []string{"packages/ui/src/button.tsx"}, true)
	if !decision.Trigger || decision.Reason != ReasonWatchPathMatched {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestEvaluateTriggerMatchesRootFile(t *testing.T) {
	decision := EvaluateTrigger(BuildTriggerRules{AutoDeploy: true, WatchRootFiles: true}, []string{"pnpm-lock.yaml", "packages/ui/button.tsx"}, true)
	if !decision.Trigger || decision.Reason != ReasonRootFileMatched {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestEvaluateTriggerIgnoresFilesAndNestedRootPatterns(t *testing.T) {
	decision := EvaluateTrigger(BuildTriggerRules{
		AutoDeploy:    true,
		RootDirectory: "apps/web",
		IgnorePaths:   []string{"**/*.md"},
	}, []string{"README.md", "apps/api/main.go"}, true)
	if decision.Trigger || decision.Reason != ReasonNoRelevantChanges {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestEvaluateTriggerFallsBackWhenFilesAreUnknown(t *testing.T) {
	decision := EvaluateTrigger(BuildTriggerRules{AutoDeploy: true}, nil, false)
	if !decision.Trigger || decision.Reason != ReasonChangedFilesUnknown {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestEvaluateTriggerMatchesAnyFileWithoutWatchFilters(t *testing.T) {
	decision := EvaluateTrigger(BuildTriggerRules{AutoDeploy: true}, []string{"src/main.ts"}, true)
	if !decision.Trigger || decision.Reason != ReasonWatchPathMatched {
		t.Fatalf("expected automatic deployment for an unfiltered file change: %+v", decision)
	}
}
