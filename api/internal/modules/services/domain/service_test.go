package domain

import "testing"

func TestNormalizeComposeDegraded(t *testing.T) {
	got := NormalizeCompose([]ContainerState{
		{Status: "running"},
		{Status: "exited"},
		{Status: "running"},
	}, false)
	if got != StatusDegraded {
		t.Fatalf("expected %q, got %q", StatusDegraded, got)
	}
}

func TestNormalizeComposeRecovery(t *testing.T) {
	got := NormalizeCompose([]ContainerState{{Status: "running"}, {Status: "healthy"}}, false)
	if got != StatusRunning {
		t.Fatalf("expected %q, got %q", StatusRunning, got)
	}
}

func TestNormalizeAppDoesNotTrustSuccessfulDeployment(t *testing.T) {
	got := NormalizeApp("success", "exited", nil)
	if got != StatusStopped {
		t.Fatalf("expected %q, got %q", StatusStopped, got)
	}
}

func TestNormalizeDatabaseHealth(t *testing.T) {
	healthy := false
	got := NormalizeDatabase("running", &healthy)
	if got != StatusDegraded {
		t.Fatalf("expected %q, got %q", StatusDegraded, got)
	}
}

func TestNormalizeDatabaseRunningHealthOK(t *testing.T) {
	healthy := true
	if got := NormalizeDatabase("running", &healthy); got != StatusRunning {
		t.Fatalf("expected %q, got %q", StatusRunning, got)
	}
}

func TestNormalizeComposeStatusMatrix(t *testing.T) {
	tests := []struct {
		name   string
		states []ContainerState
		deploy bool
		want   Status
	}{
		{name: "deploying", states: []ContainerState{{Status: "running"}}, deploy: true, want: StatusDeploying},
		{name: "stopped", states: []ContainerState{{Status: "created"}, {Status: "stopped"}}, want: StatusStopped},
		{name: "exited", states: []ContainerState{{Status: "exited"}, {Status: "stopped"}}, want: StatusFailed},
		{name: "failed", states: []ContainerState{{Status: "failed"}, {Status: "dead"}}, want: StatusFailed},
		{name: "unknown", states: nil, want: StatusUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := NormalizeCompose(test.states, test.deploy); got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}

func TestNormalizeAppStatusMatrix(t *testing.T) {
	if got := NormalizeApp("deploying", "", nil); got != StatusDeploying {
		t.Fatalf("expected %q, got %q", StatusDeploying, got)
	}
	if got := NormalizeApp("failed", "", nil); got != StatusFailed {
		t.Fatalf("expected %q, got %q", StatusFailed, got)
	}
	if got := NormalizeApp("success", "", nil); got != StatusUnknown {
		t.Fatalf("expected %q, got %q", StatusUnknown, got)
	}
}

func TestNormalizeDatabaseStatusMatrix(t *testing.T) {
	if got := NormalizeDatabase("stopped", nil); got != StatusStopped {
		t.Fatalf("expected %q, got %q", StatusStopped, got)
	}
	if got := NormalizeDatabase("", nil); got != StatusUnknown {
		t.Fatalf("expected %q, got %q", StatusUnknown, got)
	}
}

func TestCapabilitiesExposeLifecycleActionsForEveryServiceKind(t *testing.T) {
	for _, kind := range []Kind{KindApp, KindCompose, KindDatabase} {
		capabilities := CapabilitiesFor(kind)
		if !capabilities.CanDeploy || !capabilities.CanStart || !capabilities.CanStop || !capabilities.CanRestart || !capabilities.CanOpenTerminal {
			t.Fatalf("service kind %q does not expose the common lifecycle capabilities: %+v", kind, capabilities)
		}
	}
}
