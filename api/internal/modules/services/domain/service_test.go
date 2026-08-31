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
	if got := NormalizeApp("success", "", nil); got != StatusRunning {
		t.Fatalf("expected %q, got %q", StatusRunning, got)
	}
}

func TestProjectStatusUsesLatestDeploymentWithoutContainer(t *testing.T) {
	tests := []struct {
		name       string
		deployment string
		want       Status
	}{
		{name: "ready", deployment: "ready", want: StatusRunning},
		{name: "failed", deployment: "failed", want: StatusFailed},
		{name: "cancelled", deployment: "cancelled", want: StatusStopped},
		{name: "building", deployment: "building", want: StatusDeploying},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ProjectStatusWithDeployment(KindApp, nil, test.deployment, false, true); got != test.want {
				t.Fatalf("status = %q, want %q", got, test.want)
			}
		})
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

func TestProjectStatusUsesPendingForNeverDeployedServices(t *testing.T) {
	for _, kind := range []Kind{KindApp, KindCompose, KindDatabase} {
		if got := ProjectStatus(kind, nil, false, false); got != StatusPending {
			t.Fatalf("kind %q status = %q, want %q", kind, got, StatusPending)
		}
	}
}

func TestProjectStatusDeploymentTakesPrecedenceOverRuntime(t *testing.T) {
	got := ProjectStatus(KindApp, []ContainerState{{Status: "running"}}, true, true)
	if got != StatusDeploying {
		t.Fatalf("status = %q, want %q", got, StatusDeploying)
	}
}

func TestProjectStatusRuntimeMatrix(t *testing.T) {
	healthy := true
	unhealthy := false
	tests := []struct {
		name   string
		state  ContainerState
		status Status
	}{
		{name: "running", state: ContainerState{Status: "running", Healthy: &healthy}, status: StatusRunning},
		{name: "stopped", state: ContainerState{Status: "stopped"}, status: StatusStopped},
		{name: "exited", state: ContainerState{Status: "exited"}, status: StatusStopped},
		{name: "restarting", state: ContainerState{Status: "restarting"}, status: StatusDegraded},
		{name: "unhealthy", state: ContainerState{Status: "running", Healthy: &unhealthy}, status: StatusDegraded},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ProjectStatus(KindApp, []ContainerState{test.state}, false, true); got != test.status {
				t.Fatalf("status = %q, want %q", got, test.status)
			}
		})
	}
}

func TestProjectStatusComposeAggregateMatrix(t *testing.T) {
	healthy := true
	unhealthy := false
	tests := []struct {
		name   string
		states []ContainerState
		status Status
	}{
		{name: "all running", states: []ContainerState{{Status: "running", Healthy: &healthy}, {Status: "running"}}, status: StatusRunning},
		{name: "mixed", states: []ContainerState{{Status: "running"}, {Status: "stopped"}}, status: StatusDegraded},
		{name: "unhealthy", states: []ContainerState{{Status: "running", Healthy: &unhealthy}}, status: StatusDegraded},
		{name: "all stopped", states: []ContainerState{{Status: "stopped"}, {Status: "stopped"}}, status: StatusStopped},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ProjectStatus(KindCompose, test.states, false, true); got != test.status {
				t.Fatalf("status = %q, want %q", got, test.status)
			}
		})
	}
}
