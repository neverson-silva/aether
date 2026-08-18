package application

import (
	"testing"

	"aether/internal/monitoring/domain"
)

func TestParseSystemDF(t *testing.T) {
	out := `Images space usage:

REPOSITORY TAG IMAGE ID CREATED SIZE SHARED SIZE UNIQUE SIZE CONTAINERS
docker.io/library/alpine latest 1991bd789d71 2 months 8.951MB 8.94MB 11.89kB 0

Containers space usage:

CONTAINER ID IMAGE COMMAND LOCAL VOLUMES SIZE CREATED STATUS NAMES
819beade5423 7e7dbab8d3b4 postgres 1 3.198MB 6 days running aether-test-pg
9d5219574648 7e7dbab8d3b4 nginx -g daemon off; 0 11.83kB 6 days exited aether-0c6f9c06-6

Local Volumes space usage:

VOLUME NAME LINKS SIZE
aether-test-pg-data 1 334.3MB
aether-pg-data 1 69.03kB
`
	cs, vs, vls := parseSystemDF(out)
	if len(cs) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(cs))
	}
	want, _ := parseDfBytes("3.198MB")
	if cs[0].name != "aether-test-pg" || cs[0].size != want {
		t.Fatalf("container row wrong: %+v", cs[0])
	}
	want11, _ := parseDfBytes("11.83kB")
	if cs[1].size != want11 {
		t.Fatalf("command with spaces row wrong: %+v", cs[1])
	}
	want334, _ := parseDfBytes("334.3MB")
	if len(vs) != 2 || vs[0].name != "aether-test-pg-data" || vs[0].size != want334 {
		t.Fatalf("volume row wrong: %+v", vs)
	}
	if len(vls) != 2 || vls[0].links != 1 {
		t.Fatalf("volume links wrong: %+v", vls)
	}
}

func TestParseDfBytes(t *testing.T) {
	cases := map[string]uint64{
		"0B": 0, "25.02kB": 25620, "1.101MB": 1154482, "334.3MB": 350538956,
		"1.5GB": 1610612736, "2TB": 2199023255552,
	}
	for in, want := range cases {
		got, ok := parseDfBytes(in)
		if !ok || got != want {
			t.Fatalf("%s: got %d ok=%v want %d", in, got, ok, want)
		}
	}
	if _, ok := parseDfBytes("nope"); ok {
		t.Fatal("expected parse failure")
	}
}

func TestAggregateStorageIncludesStopped(t *testing.T) {
	stopped := uint64(1000)
	resources := []domain.Resource{
		{ID: "c1", Owner: domain.OwnerUser, State: "exited", Active: false, Storage: &stopped},
		{ID: "c2", Owner: domain.OwnerUser, State: "running", Active: true, Storage: &stopped},
		{ID: "c3", Owner: domain.OwnerAether, State: "running", Active: true},
	}
	_, user, _ := Aggregate(resources, domain.Host{})
	if user.StorageUsage != 2000 {
		t.Fatalf("storage must include stopped containers, got %d", user.StorageUsage)
	}
	if user.RunningCount != 1 || user.Count != 2 {
		t.Fatalf("counts wrong: %+v", user)
	}
}
