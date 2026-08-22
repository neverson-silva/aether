package application

import (
	"testing"

	"aether/internal/modules/monitoring/domain"
)

func TestClassifyUserByLabel(t *testing.T) {
	owner, st, sid, pid, name := Classify(rawContainer{
		Name: "aether-abc12345-2",
		Labels: map[string]string{
			"aether.owner":        "user",
			"aether.service-type": "app",
			"aether.service-id":   "11111111-1111-1111-1111-111111111111",
			"aether.project-id":   "22222222-2222-2222-2222-222222222222",
			"aether.service-name": "checkout-api",
		},
	})
	if owner != domain.OwnerUser || st != "app" || sid != "11111111-1111-1111-1111-111111111111" || pid != "22222222-2222-2222-2222-222222222222" || name != "checkout-api" {
		t.Fatalf("unexpected classification: %q %q %q %q %q", owner, st, sid, pid, name)
	}
}

func TestClassifyAetherByLabel(t *testing.T) {
	owner, st, _, _, name := Classify(rawContainer{
		Name: "aether-api",
		Labels: map[string]string{
			"aether.owner":   "aether",
			"aether.service": "api",
		},
	})
	if owner != domain.OwnerAether || st != "api" || name != "aether-api" {
		t.Fatalf("unexpected classification: %q %q %q", owner, st, name)
	}
}

func TestClassifyUnknownWhenNoInfo(t *testing.T) {
	owner, st, _, _, _ := Classify(rawContainer{Name: "some-random-78g5", Labels: map[string]string{}})
	if owner != domain.OwnerUnknown {
		t.Fatalf("expected unknown, got %q", owner)
	}
	if st != "container" {
		t.Fatalf("expected container type, got %q", st)
	}
}

func TestClassifyLegacyPlatformFallback(t *testing.T) {
	cases := map[string]string{
		"aether-api":      "api",
		"aether-postgres": "postgres",
		"aether-redis":    "redis",
		"aether-traefik":  "proxy",
	}
	for name, svc := range cases {
		owner, st, _, _, _ := Classify(rawContainer{Name: name, Labels: map[string]string{}})
		if owner != domain.OwnerAether || st != svc {
			t.Fatalf("%s: got owner=%q type=%q", name, owner, st)
		}
	}
}

func TestClassifyLegacyUserFallback(t *testing.T) {
	owner, st, _, _, _ := Classify(rawContainer{Name: "aether-1234abcd-7", Labels: map[string]string{}})
	if owner != domain.OwnerUser || st != "app" {
		t.Fatalf("expected user/app, got %q %q", owner, st)
	}
}

func TestClassifyTestInfra(t *testing.T) {
	for _, name := range []string{"aether-test-pg", "aether-redis-test", "aether-test-something"} {
		owner, st, _, _, _ := Classify(rawContainer{Name: name, Labels: map[string]string{}})
		if owner != domain.OwnerAether || st != "test" {
			t.Fatalf("%s: expected aether/test, got %q %q", name, owner, st)
		}
	}
}

func resource(id, owner, state string, cpu float64, mem uint64) domain.Resource {
	return domain.Resource{
		ID: id, Name: id, Owner: owner, State: state, Active: isActive(state),
		CPUOfHost: cpu, MemUsage: mem,
	}
}

func TestAggregateSplit(t *testing.T) {
	resources := []domain.Resource{
		resource("c1", domain.OwnerAether, "running", 8, 2000),
		resource("c2", domain.OwnerUser, "running", 34, 9000),
		resource("c3", domain.OwnerUser, "running", 5, 1000),
		resource("c4", domain.OwnerUser, "exited", 90, 99999),
	}
	host := domain.Host{CPUPercent: 50, MemUsed: 20000, MemTotal: 32000, MemPercent: 62.5}
	aether, user, system := Aggregate(resources, host)
	if aether.CPUOfHost != 8 {
		t.Fatalf("aether cpu = %v", aether.CPUOfHost)
	}
	if user.CPUOfHost != 39 {
		t.Fatalf("user cpu = %v (stopped container must be excluded)", user.CPUOfHost)
	}
	if user.RunningCount != 2 || user.Count != 3 {
		t.Fatalf("user counts = %d/%d", user.RunningCount, user.Count)
	}
	if system.CPUOfHost != 3 {
		t.Fatalf("system cpu = %v", system.CPUOfHost)
	}
	if user.MemUsage != 10000 {
		t.Fatalf("user mem = %v (stopped excluded)", user.MemUsage)
	}
	if system.MemUsage != 8000 {
		t.Fatalf("system mem = %v (host - aether - user)", system.MemUsage)
	}
}

func TestAggregateZeroResources(t *testing.T) {
	aether, user, system := Aggregate(nil, domain.Host{CPUPercent: 10, MemUsed: 1000, MemTotal: 8000})
	if !aether.Available || !user.Available {
		t.Fatal("aggregates should be available even with zero resources")
	}
	if aether.CPUOfHost != 0 || user.CPUOfHost != 0 {
		t.Fatal("empty buckets must be zero")
	}
	if system.CPUOfHost != 10 || system.MemUsage != 1000 {
		t.Fatalf("system = %v cpu / %v mem", system.CPUOfHost, system.MemUsage)
	}
}

func TestAggregateHostLessThanAttributed(t *testing.T) {
	// Working-set memory can exceed host "used"; system bucket must clamp to 0.
	resources := []domain.Resource{
		resource("c1", domain.OwnerUser, "running", 40, 20000),
	}
	host := domain.Host{CPUPercent: 30, MemUsed: 10000, MemTotal: 32000}
	_, user, system := Aggregate(resources, host)
	if user.MemUsage != 20000 {
		t.Fatalf("user mem = %v", user.MemUsage)
	}
	if system.MemUsage != 0 {
		t.Fatalf("system mem must clamp to 0, got %v", system.MemUsage)
	}
	if system.CPUOfHost != 0 {
		t.Fatalf("system cpu must clamp to 0, got %v", system.CPUOfHost)
	}
}

func TestAggregateLargeNumberOfResources(t *testing.T) {
	resources := make([]domain.Resource, 0, 500)
	for i := 0; i < 500; i++ {
		resources = append(resources, resource("c"+string(rune('a'+i%26)), domain.OwnerUser, "running", 1, 100))
	}
	_, user, _ := Aggregate(resources, domain.Host{CPUPercent: 99, MemUsed: 100000, MemTotal: 1 << 30})
	if user.Count != 500 || user.RunningCount != 500 {
		t.Fatalf("counts = %d/%d", user.Count, user.RunningCount)
	}
	if user.CPUOfHost < 499 || user.CPUOfHost > 501 {
		t.Fatalf("user cpu = %v", user.CPUOfHost)
	}
}

func TestIsActive(t *testing.T) {
	if !isActive("running") || !isActive("restarting") {
		t.Fatal("running/restarting are active")
	}
	for _, s := range []string{"exited", "paused", "created", "dead", "stopped"} {
		if isActive(s) {
			t.Fatalf("%s must be inactive", s)
		}
	}
}
