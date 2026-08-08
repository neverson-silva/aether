package runtime

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

var errFake = errors.New("fake error")

type fakeExec struct {
	calls []string
	errs  map[string]error
}

func (f *fakeExec) Run(ctx context.Context, args ...string) (string, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	if f.errs != nil {
		if err, ok := f.errs[args[0]]; ok {
			return "", err
		}
	}
	switch args[0] {
	case "version":
		return "99.0", nil
	case "info":
		return "overlay2", nil
	case "image", "inspect":
		return `{"Id":"img1","Name":"/x"}`, nil
	case "run":
		return "cid123", nil
	case "port":
		return "80/tcp -> 127.0.0.1:34567", nil
	case "stats":
		return `{"CPUPerc":"0.50%","MemUsage":"5MiB / 8GiB","PIDs":"3","NetIO":"1.2kB / 3.4kB","BlockIO":"5MiB / 2MiB"}`, nil
	}
	return "ok", nil
}

func (f *fakeExec) RunIO(ctx context.Context, args ...string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("log line\n")), nil
}

func TestDriverRunSpecMapping(t *testing.T) {
	fe := &fakeExec{}
	d := &cliDriver{name: "docker", exec: fe}
	ctx := context.Background()
	id, err := d.Run(ctx, RunSpec{
		Name:     "aether-web-1",
		Image:    "nginx:alpine",
		Cmd:      []string{"nginx", "-g", "daemon off;"},
		Env:      []string{"FOO=bar"},
		Ports:    []PortBinding{{HostPort: "20001", ContainerPort: "80"}},
		Networks: []string{"aether-proj"},
		Volumes:  []VolumeMount{{Source: "aether-web-data", Target: "/data"}},
		MemMB:    256,
		CPUs:     "0.5",
		Restart:  "on-failure",
		Labels:   map[string]string{"aether.app": "app1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "cid123" {
		t.Fatalf("id divergiu: %s", id)
	}
	runCall := fe.calls[len(fe.calls)-1]
	for _, want := range []string{
		"run", "-d", "--name", "aether-web-1",
		"-l", "aether.app=app1",
		"-e", "FOO=bar",
		"-p", "127.0.0.1:20001:80",
		"--network", "aether-proj",
		"-v", "aether-web-data:/data",
		"--memory", "256m",
		"--cpus", "0.5",
		"--restart", "on-failure",
		"nginx:alpine",
	} {
		if !strings.Contains(runCall, want) {
			t.Fatalf("run sem %q: %s", want, runCall)
		}
	}
}

func TestDriverPortAutoAssign(t *testing.T) {
	fe := &fakeExec{}
	d := &cliDriver{name: "docker", exec: fe}
	ports, err := d.Ports(context.Background(), "cid123")
	if err != nil {
		t.Fatal(err)
	}
	if ports["80"] != "127.0.0.1:34567" {
		t.Fatalf("porta divergiu: %v", ports)
	}
}

func TestDriverStatsParse(t *testing.T) {
	fe := &fakeExec{}
	d := &cliDriver{name: "docker", exec: fe}
	st, err := d.Stats(context.Background(), "cid123")
	if err != nil {
		t.Fatal(err)
	}
	if st.CPUPercent != 0.5 {
		t.Fatalf("cpu divergiu: %f", st.CPUPercent)
	}
	if st.MemBytes != 5*(1<<20) {
		t.Fatalf("mem divergiu: %d", st.MemBytes)
	}
	if st.Pids != 3 {
		t.Fatalf("pids divergiu: %d", st.Pids)
	}
	if st.NetRxBytes != 1200 || st.NetTxBytes != 3400 {
		t.Fatalf("net divergiu: %d %d", st.NetRxBytes, st.NetTxBytes)
	}
	if st.IOReadBytes != 5*(1<<20) || st.IOWriteBytes != 2*(1<<20) {
		t.Fatalf("io divergiu: %d %d", st.IOReadBytes, st.IOWriteBytes)
	}
}

func TestDriverErrorWrapping(t *testing.T) {
	fe := &fakeExec{errs: map[string]error{"stop": errFake}}
	d := &cliDriver{name: "docker", exec: fe}
	err := d.Stop(context.Background(), "x")
	if err == nil || !strings.Contains(err.Error(), "fake") {
		t.Fatalf("erro deveria propagar: %v", err)
	}
}
