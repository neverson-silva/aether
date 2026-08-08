package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"aether/internal/runtime"
)

const version = "0.1.0"

type Agent struct {
	CoreURL string
	Token   string
	Name    string
	Labels  []string
	State   string
	client  *http.Client
	server  string
}

func Run(coreURL, token, name string, labels []string, stateDir string) error {
	if err := os.MkdirAll(stateDir, 0o750); err != nil {
		return err
	}
	a := &Agent{
		CoreURL: strings.TrimSuffix(coreURL, "/"),
		Token:   token,
		Name:    name,
		Labels:  labels,
		State:   stateDir,
	}
	ident, err := a.loadOrRegister()
	if err != nil {
		return err
	}
	cert, err := tls.X509KeyPair([]byte(ident.CertPEM), []byte(ident.KeyPEM))
	if err != nil {
		return err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM([]byte(ident.CAPEM))
	a.client = &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
				RootCAs:      pool,
				MinVersion:   tls.VersionTLS12,
			},
		},
	}
	log.Printf("[agent] conectado ao core %s como %s (%s)", a.CoreURL, a.Name, a.server)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go a.heartbeatLoop(ctx)
	return a.commandLoop(ctx)
}

func (a *Agent) loadOrRegister() (*identity, error) {
	state := filepath.Join(a.State, "agent.json")
	if data, err := os.ReadFile(state); err == nil {
		var id identity
		if json.Unmarshal(data, &id) == nil && id.ServerID != "" {
			a.server = id.ServerID
			return &id, nil
		}
	}
	hostname, _ := os.Hostname()
	cpu := goosCPUCount()
	body, _ := json.Marshal(map[string]any{
		"token":           a.Token,
		"name":            a.Name,
		"host":            hostname,
		"version":         version,
		"labels":          a.Labels,
		"cpu_cores":       cpu,
		"mem_total_bytes": memTotal(),
	})
	resp, err := a.rawPost("/agent/v1/register", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registro falhou (%d): %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var id identity
	if err := json.NewDecoder(resp.Body).Decode(&id); err != nil {
		return nil, err
	}
	if err := os.WriteFile(state, mustJSON(id), 0o600); err != nil {
		return nil, err
	}
	a.server = id.ServerID
	return &id, nil
}

type identity struct {
	ServerID string `json:"server_id"`
	CertPEM  string `json:"cert_pem"`
	KeyPEM   string `json:"key_pem"`
	CAPEM    string `json:"ca_pem"`
}

func (a *Agent) rawPost(path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", a.CoreURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	insecure := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12},
	}}
	return insecure.Do(req)
}

func (a *Agent) post(path string, body any, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", a.CoreURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", path, strings.TrimSpace(string(b)))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (a *Agent) heartbeatLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
			var out struct {
				Commands []cmd `json:"commands"`
			}
			err := a.post("/agent/v1/heartbeat", map[string]any{
				"load":    loadAvg(),
				"version": version,
			}, &out)
			if err != nil {
				log.Printf("[agent] heartbeat: %v", err)
				continue
			}
			for _, c := range out.Commands {
				go a.handleCommand(c)
			}
		}
	}
}

func (a *Agent) commandLoop(ctx context.Context) error {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			var out struct {
				Commands []cmd `json:"commands"`
			}
			if err := a.post("/agent/v1/commands", map[string]any{}, &out); err != nil {
				continue
			}
			for _, c := range out.Commands {
				a.handleCommand(c)
			}
		}
	}
}

type cmd struct {
	ID      string          `json:"id"`
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

func (a *Agent) handleCommand(c cmd) {
	log.Printf("[agent] comando recebido: %s (%s)", c.Kind, c.ID)
	switch c.Kind {
	case "deploy":
		a.deploy(c.Payload)
	default:
		log.Printf("[agent] comando desconhecido: %s", c.Kind)
	}
}

func (a *Agent) sendEvent(evType string, payload map[string]any) {
	_ = a.post("/agent/v1/events", map[string]any{
		"events": []map[string]any{{"type": evType, "payload": payload}},
	}, nil)
}

func (a *Agent) deploy(raw json.RawMessage) {
	var req struct {
		DeploymentID string `json:"deployment_id"`
		App          struct {
			ID        string                `json:"id"`
			Name      string                `json:"name"`
			Image     string                `json:"image"`
			Port      int                   `json:"port"`
			ProjectID string                `json:"project_id"`
			Env       []string              `json:"env"`
			Volumes   []runtime.VolumeMount `json:"volumes"`
			MemMB     int64                 `json:"mem_mb"`
			CPUs      string                `json:"cpus"`
			Health    map[string]any        `json:"health_check"`
			Raw       map[string]any
		}
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		log.Printf("[agent] deploy payload inválido: %v", err)
		a.sendEvent("deploy.failed", map[string]any{"deployment_id": "", "error": err.Error()})
		return
	}
	log.Printf("[agent] deploy %s (dep %s)", req.App.Image, req.DeploymentID)
	app := req.App
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	logf := func(line string) {
		a.sendEvent("deploy.log", map[string]any{"deployment_id": req.DeploymentID, "line": line})
	}
	logf("[agent] deploy " + app.Image + "\n")
	driver, err := runtime.NewDriver(autoDriver())
	if err != nil {
		a.sendEvent("deploy.failed", map[string]any{"deployment_id": req.DeploymentID, "error": err.Error()})
		return
	}
	if err := driver.Pull(ctx, app.Image); err != nil {
		a.sendEvent("deploy.failed", map[string]any{"deployment_id": req.DeploymentID, "error": "pull: " + err.Error()})
		return
	}
	network := "aether-" + app.ProjectID
	if err := driver.NetworkCreate(ctx, network); err != nil {
		if !strings.Contains(err.Error(), "exists") {
			a.sendEvent("deploy.failed", map[string]any{"deployment_id": req.DeploymentID, "error": err.Error()})
			return
		}
	}
	var volumes []runtime.VolumeMount
	for _, v := range app.Volumes {
		volName := "aether-" + app.Name + "-" + v.Target
		if err := driver.VolumeCreate(ctx, volName, 0); err != nil {
			if !strings.Contains(err.Error(), "exists") {
				a.sendEvent("deploy.failed", map[string]any{"deployment_id": req.DeploymentID, "error": err.Error()})
				return
			}
		}
		volumes = append(volumes, runtime.VolumeMount{Source: volName, Target: v.Target})
	}
	name := "aether-" + app.Name + "-" + strconv.FormatInt(time.Now().Unix(), 10)
	id, err := driver.Run(ctx, runtime.RunSpec{
		Name:     name,
		Image:    app.Image,
		Env:      app.Env,
		Ports:    []runtime.PortBinding{{HostPort: "0", ContainerPort: strconv.Itoa(app.Port)}},
		Networks: []string{network},
		Volumes:  volumes,
		MemMB:    app.MemMB,
		CPUs:     app.CPUs,
		Restart:  "unless-stopped",
		Labels:   map[string]string{"aether.app": app.ID},
	})
	if err != nil {
		a.sendEvent("deploy.failed", map[string]any{"deployment_id": req.DeploymentID, "error": err.Error()})
		return
	}
	logf("[agent] container " + id + " rodando\n")
	a.sendEvent("deploy.ready", map[string]any{"deployment_id": req.DeploymentID, "container_id": id})
}

func goosCPUCount() int {
	n, err := strconv.Atoi(strings.TrimSpace(string(os.Getenv("AETHER_CPU_OVERRIDE"))))
	if err == nil && n > 0 {
		return n
	}
	return 8
}

func autoDriver() string {
	return "podman"
}

func goosIsLinux() bool {
	if _, err := os.Stat("/proc/loadavg"); err == nil {
		return true
	}
	return false
}

func mustJSON(v any) []byte {
	b, _ := json.MarshalIndent(v, "", "  ")
	return b
}

func loadAvg() float64 {
	data, err := os.ReadFile("/proc/loadavg")
	if err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			if v, err := strconv.ParseFloat(fields[0], 64); err == nil {
				return v
			}
		}
	}
	return 0
}

func memTotal() int64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) > 1 {
					if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
						return kb * 1024
					}
				}
			}
		}
	}
	return 0
}
