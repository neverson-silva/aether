package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
)

type Execer interface {
	Run(ctx context.Context, args ...string) (string, error)
	RunIO(ctx context.Context, args ...string) (io.ReadCloser, error)
}

type osExec struct{ bin string }

func killGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

func (o osExec) newCommand(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, o.bin, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return killGroup(cmd) }
	cmd.WaitDelay = 5 * time.Second
	return cmd
}

func (o osExec) Run(ctx context.Context, args ...string) (string, error) {
	cmd := o.newCommand(ctx, args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s: %s", o.bin, msg)
	}
	return strings.TrimSpace(out.String()), nil
}

func (o osExec) RunIO(ctx context.Context, args ...string) (io.ReadCloser, error) {
	cmd := o.newCommand(ctx, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return stdout, nil
}

type cliDriver struct {
	name string
	exec Execer
}

func NewPodman() Driver {
	return &cliDriver{name: "podman", exec: osExec{bin: "podman"}}
}

func NewDriver(name string) (Driver, error) {
	switch name {
	case "podman":
		return NewPodman(), nil
	}
	return nil, fmt.Errorf("runtime não suportado: %s (runtime oficial: podman)", name)
}

func (d *cliDriver) Name() string { return d.name }

func (d *cliDriver) Info(ctx context.Context) (Info, error) {
	version, err := d.exec.Run(ctx, "version", "--format", "{{.Server.Version}}")
	if err != nil {
		version, err = d.exec.Run(ctx, "version", "--format", "{{.Version}}")
		if err != nil {
			return Info{}, err
		}
	}
	store, _ := d.exec.Run(ctx, "info", "--format", "{{.Driver}}")
	rootless := false
	if d.name == "podman" {
		rl, _ := d.exec.Run(ctx, "info", "--format", "{{.Host.Security.Rootless}}")
		rootless = strings.TrimSpace(rl) == "true"
	}
	return Info{
		Driver:        d.name,
		Version:       version,
		StorageDriver: store,
		Rootless:      rootless,
		Capabilities:  []string{"cgroupv2"},
	}, nil
}

func (d *cliDriver) Pull(ctx context.Context, image string) error {
	_, err := d.exec.Run(ctx, "pull", "-q", image)
	return err
}

func (d *cliDriver) Exists(ctx context.Context, image string) (bool, error) {
	_, err := d.exec.Run(ctx, "image", "inspect", image)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (d *cliDriver) Run(ctx context.Context, spec RunSpec) (string, error) {
	args := []string{"run", "-d", "--name", spec.Name}
	for k, v := range spec.Labels {
		args = append(args, "-l", k+"="+v)
	}
	for _, e := range spec.Env {
		args = append(args, "-e", e)
	}
	for _, p := range spec.Ports {
		if p.HostPort != "" {
			args = append(args, "-p", "127.0.0.1:"+p.HostPort+":"+p.ContainerPort)
		} else {
			args = append(args, "-p", "127.0.0.1::"+p.ContainerPort)
		}
	}
	for _, n := range spec.Networks {
		args = append(args, "--network", n)
	}
	if spec.NetworkAlias != "" && len(spec.Networks) > 0 {
		args = append(args, "--network-alias", spec.NetworkAlias)
	}
	for _, v := range spec.Volumes {
		args = append(args, "-v", v.Source+":"+v.Target)
	}
	if spec.MemMB > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dm", spec.MemMB))
	}
	if spec.CPUs != "" {
		args = append(args, "--cpus", spec.CPUs)
	}
	if spec.PidsLimit > 0 {
		args = append(args, "--pids-limit", fmt.Sprintf("%d", spec.PidsLimit))
	}
	if spec.Restart != "" {
		args = append(args, "--restart", spec.Restart)
	}
	if spec.ReadOnly {
		args = append(args, "--read-only")
	}
	args = append(args, spec.Image)
	args = append(args, spec.Cmd...)
	id, err := d.exec.Run(ctx, args...)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (d *cliDriver) Start(ctx context.Context, id string) error {
	_, err := d.exec.Run(ctx, "start", id)
	return err
}

func (d *cliDriver) Stop(ctx context.Context, id string) error {
	_, err := d.exec.Run(ctx, "stop", "--time", "10", id)
	return err
}

func (d *cliDriver) Restart(ctx context.Context, id string) error {
	_, err := d.exec.Run(ctx, "restart", id)
	return err
}

func (d *cliDriver) UpdateResources(ctx context.Context, id string, memMB int64, cpus string) error {
	args := []string{"update"}
	if memMB > 0 {
		args = append(args, "--memory", strconv.FormatInt(memMB, 10)+"m")
	}
	if cpus != "" {
		args = append(args, "--cpus", cpus)
	}
	args = append(args, id)
	_, err := d.exec.Run(ctx, args...)
	return err
}

func (d *cliDriver) Remove(ctx context.Context, id string, force bool) error {
	args := []string{"rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, id)
	_, err := d.exec.Run(ctx, args...)
	return err
}

func (d *cliDriver) Inspect(ctx context.Context, id string) (ContainerInfo, error) {
	raw, err := d.exec.Run(ctx, "inspect", "--format", "{{json .}}", id)
	if err != nil {
		return ContainerInfo{}, err
	}
	var c struct {
		ID    string `json:"Id"`
		Name  string `json:"Name"`
		State struct {
			Status    string `json:"Status"`
			StartedAt string `json:"StartedAt"`
		} `json:"State"`
	}
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return ContainerInfo{}, err
	}
	started, _ := time.Parse(time.RFC3339Nano, c.State.StartedAt)
	return ContainerInfo{
		ID:        c.ID,
		Name:      strings.TrimPrefix(c.Name, "/"),
		State:     c.State.Status,
		StartedAt: started,
	}, nil
}

func (d *cliDriver) Ports(ctx context.Context, id string) (map[string]string, error) {
	raw, err := d.exec.Run(ctx, "port", id)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			container := strings.TrimSuffix(parts[0], "/tcp")
			out[container] = parts[2]
		}
	}
	return out, nil
}

func (d *cliDriver) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, id)
	return d.exec.RunIO(ctx, args...)
}

func (d *cliDriver) Stats(ctx context.Context, id string) (Stats, error) {
	raw, err := d.exec.Run(ctx, "stats", "--no-stream", "--format", "{{json .}}", id)
	if err != nil {
		return Stats{}, err
	}
	var s struct {
		CPUPerc     json.RawMessage `json:"CPUPerc"`
		MemUsage    json.RawMessage `json:"MemUsage"`
		MemPerc     json.RawMessage `json:"MemPerc"`
		PIDs        json.RawMessage `json:"PIDs"`
		NetIO       json.RawMessage `json:"NetIO"`
		BlockIO     json.RawMessage `json:"BlockIO"`
		CPUPercent  json.RawMessage `json:"cpu_percent"`
		MemUsageP   json.RawMessage `json:"mem_usage"`
		MemLimit    json.RawMessage `json:"mem_limit"`
		MemPercP    json.RawMessage `json:"mem_percent"`
		NetInput    json.RawMessage `json:"net_input"`
		NetOutput   json.RawMessage `json:"net_output"`
		BlockInput  json.RawMessage `json:"block_input"`
		BlockOutput json.RawMessage `json:"block_output"`
		Pids        json.RawMessage `json:"pids"`
	}
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return Stats{}, err
	}
	var st Stats
	if len(s.CPUPerc) > 0 {
		st.CPUPercent = parseNumStr(rawStr(s.CPUPerc))
	} else {
		st.CPUPercent = parseNumStr(rawStr(s.CPUPercent))
	}
	if len(s.MemUsage) > 0 {
		used, limit := parseMemUsage(rawStr(s.MemUsage))
		st.MemBytes, st.MemLimit = used, limit
	} else {
		st.MemBytes = uint64(parseNumStr(rawStr(s.MemUsageP)))
		st.MemLimit = uint64(parseNumStr(rawStr(s.MemLimit)))
	}
	if len(s.PIDs) > 0 {
		fmt.Sscanf(strings.TrimSpace(rawStr(s.PIDs)), "%d", &st.Pids)
	} else {
		st.Pids = int64(parseNumStr(rawStr(s.Pids)))
	}
	if len(s.NetIO) > 0 {
		st.NetRxBytes, st.NetTxBytes = parseIOUsage(rawStr(s.NetIO))
	} else {
		st.NetRxBytes = uint64(parseNumStr(rawStr(s.NetInput)))
		st.NetTxBytes = uint64(parseNumStr(rawStr(s.NetOutput)))
	}
	if len(s.BlockIO) > 0 {
		st.IOReadBytes, st.IOWriteBytes = parseIOUsage(rawStr(s.BlockIO))
	} else {
		st.IOReadBytes = uint64(parseNumStr(rawStr(s.BlockInput)))
		st.IOWriteBytes = uint64(parseNumStr(rawStr(s.BlockOutput)))
	}
	return st, nil
}

func rawStr(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		var v string
		if json.Unmarshal(raw, &v) == nil {
			return v
		}
	}
	return s
}

func parseNumStr(s string) float64 {
	var v float64
	fmt.Sscanf(strings.TrimSpace(s), "%f", &v)
	return v
}

func (d *cliDriver) Build(ctx context.Context, dir, dockerfile, tag string) (string, error) {
	args := []string{"build", "-t", tag, "-f", dockerfile, "--pull", dir}
	if _, err := d.exec.Run(ctx, args...); err != nil {
		return "", err
	}
	return tag, nil
}

func (d *cliDriver) BuildWithWriter(ctx context.Context, dir, dockerfile, tag string, w io.Writer) (string, error) {
	args := []string{"build", "-t", tag, "-f", dockerfile, "--pull", dir}
	rc, err := d.exec.RunIO(ctx, args...)
	if err != nil {
		return "", err
	}
	defer rc.Close()
	if w != nil {
		_, _ = io.Copy(w, rc)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return tag, nil
}

func (d *cliDriver) NetworkCreate(ctx context.Context, name string) error {
	_, err := d.exec.Run(ctx, "network", "create", name)
	return err
}

func (d *cliDriver) NetworkRemove(ctx context.Context, name string) error {
	_, err := d.exec.Run(ctx, "network", "rm", name)
	return err
}

func (d *cliDriver) VolumeCreate(ctx context.Context, name string, sizeMB int64) error {
	args := []string{"volume", "create", name}
	if sizeMB > 0 {
		args = append(args, "--opt", "size="+strconv.FormatInt(sizeMB/1024, 10)+"g")
	}
	_, err := d.exec.Run(ctx, args...)
	return err
}

func (d *cliDriver) VolumeRemove(ctx context.Context, name string) error {
	_, err := d.exec.Run(ctx, "volume", "rm", "-f", name)
	return err
}

func parsePercent(s string) float64 {
	s = strings.TrimSpace(strings.TrimSuffix(s, "%"))
	var v float64
	fmt.Sscanf(s, "%f", &v)
	return v
}

func parseMemUsage(s string) (uint64, uint64) {
	parts := strings.SplitN(s, "/", 2)
	used, limit := uint64(0), uint64(0)
	if len(parts) > 0 {
		used = parseMemPart(parts[0])
	}
	if len(parts) > 1 {
		limit = parseMemPart(parts[1])
	}
	return used, limit
}

func parseIOUsage(s string) (uint64, uint64) {
	parts := strings.SplitN(s, "/", 2)
	a, b := uint64(0), uint64(0)
	if len(parts) > 0 {
		a = parseMemPart(parts[0])
	}
	if len(parts) > 1 {
		b = parseMemPart(parts[1])
	}
	return a, b
}

func parseMemPart(s string) uint64 {
	s = strings.TrimSpace(s)
	var v float64
	var unit string
	fmt.Sscanf(s, "%f%s", &v, &unit)
	mult := uint64(1)
	switch strings.ToLower(unit) {
	case "kib":
		mult = 1 << 10
	case "kb", "k":
		mult = 1000
	case "mib":
		mult = 1 << 20
	case "mb", "m":
		mult = 1000 * 1000
	case "gib":
		mult = 1 << 30
	case "gb", "g":
		mult = 1000 * 1000 * 1000
	case "tib":
		mult = 1 << 40
	case "tb", "t":
		mult = 1000 * 1000 * 1000 * 1000
	}
	return uint64(v * float64(mult))
}

func (d *cliDriver) Exec(ctx context.Context, id string, req ExecRequest) (*ExecResult, error) {
	args := []string{"exec", id}
	args = append(args, req.Command...)
	cmd := exec.CommandContext(ctx, d.name, args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	code := 0
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			return nil, err
		}
	}
	res := &ExecResult{Stdout: out.Bytes(), ExitCode: code}
	if errb.Len() > 0 {
		res.Stdout = append(res.Stdout, errb.Bytes()...)
	}
	return res, nil
}

type cliExecStream struct {
	cmd  *exec.Cmd
	pipe io.WriteCloser
	out  io.ReadCloser
}

func (s *cliExecStream) Write(p []byte) (int, error) { return s.pipe.Write(p) }
func (s *cliExecStream) Close() error                { return s.pipe.Close() }
func (s *cliExecStream) Stdout() io.Reader           { return s.out }
func (s *cliExecStream) Wait() (int, error) {
	err := s.cmd.Wait()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}
func (s *cliExecStream) Resize(cols, rows uint16) error { return nil }

func (d *cliDriver) ExecStream(ctx context.Context, id string, req ExecRequest) (ExecStream, error) {
	args := []string{"exec", "-i"}
	if req.TTY {
		args = append(args, "-t")
	}
	args = append(args, id)
	args = append(args, req.Command...)
	cmd := exec.CommandContext(ctx, d.name, args...)
	if req.TTY {
		master, err := pty.Start(cmd)
		if err != nil {
			return nil, err
		}
		return &cliPTYStream{cmd: cmd, master: master, out: master}, nil
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &cliExecStream{cmd: cmd, pipe: stdin, out: stdout}, nil
}

type cliPTYStream struct {
	cmd    *exec.Cmd
	master *os.File
	out    io.Reader
}

func (s *cliPTYStream) Write(p []byte) (int, error) { return s.master.Write(p) }
func (s *cliPTYStream) Stdout() io.Reader           { return s.out }
func (s *cliPTYStream) Close() error {
	s.master.Close()
	return s.cmd.Process.Kill()
}
func (s *cliPTYStream) Wait() (int, error) {
	s.master.Close()
	err := s.cmd.Wait()
	if s.cmd.ProcessState != nil {
		return s.cmd.ProcessState.ExitCode(), err
	}
	return -1, err
}
func (s *cliPTYStream) Resize(cols, rows uint16) error {
	return pty.Setsize(s.master, &pty.Winsize{Cols: cols, Rows: rows})
}
