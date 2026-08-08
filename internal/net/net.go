package net

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"aether/internal/config"
)

type Route struct {
	Target    string
	TLS       bool
	CertPath  string
	KeyPath   string
	Challenge bool
}

type Engine struct {
	cfg    *config.Config
	mu     sync.RWMutex
	routes map[string]Route
	proxy  *ProxySupervisor
}

func NewEngine(cfg *config.Config) *Engine {
	e := &Engine{
		cfg:    cfg,
		routes: map[string]Route{},
	}
	e.proxy = NewProxySupervisor(cfg.TraefikBin, filepath.Join(cfg.StateDir, "proxy"))
	return e
}

func (e *Engine) SetRoute(host, target string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.routes[host] = Route{Target: target}
}

func (e *Engine) SetRouteTLS(host, target, certPath, keyPath string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.routes[host] = Route{Target: target, TLS: true, CertPath: certPath, KeyPath: keyPath}
}

func (e *Engine) SetChallengeRoute(host string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.routes[host] = Route{Challenge: true, Target: "http://" + e.cfg.ChallengeAddr}
}

func (e *Engine) RemoveRoute(host string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.routes, host)
}

func (e *Engine) GetRoute(host string) (Route, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	r, ok := e.routes[host]
	return r, ok
}

func (e *Engine) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/config", e.serveConfig)
	return mux
}

func (e *Engine) serveConfig(w http.ResponseWriter, r *http.Request) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	routers := map[string]any{}
	services := map[string]any{}
	for host, rt := range e.routes {
		svcName := "svc-" + sanitize(host)
		services[svcName] = map[string]any{
			"loadBalancer": map[string]any{
				"servers": []map[string]string{{"url": rt.Target}},
			},
		}
		if rt.Challenge {
			routers["challenge-"+sanitize(host)] = map[string]any{
				"rule":        "Host(`" + host + "`) && PathPrefix(`/.well-known/acme-challenge/`)",
				"service":     svcName,
				"entryPoints": []string{"web"},
				"priority":    1000,
			}
			continue
		}
		entry := "web"
		router := map[string]any{
			"rule":        "Host(`" + host + "`)",
			"service":     svcName,
			"entryPoints": []string{entry},
		}
		if rt.TLS {
			entry = "websecure"
			router["entryPoints"] = []string{entry}
			router["tls"] = map[string]any{
				"certificates": []map[string]string{
					{"certFile": rt.CertPath, "keyFile": rt.KeyPath},
				},
			}
		}
		routers[sanitize(host)] = router
	}
	doc := map[string]any{
		"http": map[string]any{
			"routers":  routers,
			"services": services,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func sanitize(s string) string {
	return strings.NewReplacer(".", "-", "*", "wild", ":", "-").Replace(s)
}

func (e *Engine) StartProxy(ctx context.Context) error {
	return e.proxy.Start(ctx, e.cfg.ProxyEndpoint, e.cfg.StateDir)
}

func (e *Engine) StopProxy() {
	e.proxy.Stop()
}

func (e *Engine) ProxyRunning() bool {
	return e.proxy.Running()
}

type ProxySupervisor struct {
	bin     string
	dir     string
	mu      sync.Mutex
	cmd     *exec.Cmd
	stopped bool
}

func NewProxySupervisor(bin, dir string) *ProxySupervisor {
	return &ProxySupervisor{bin: bin, dir: dir}
}

func (p *ProxySupervisor) Start(ctx context.Context, endpoint, stateDir string) error {
	if p.bin == "" {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd != nil && p.cmd.Process != nil {
		return nil
	}
	if err := os.MkdirAll(p.dir, 0o750); err != nil {
		return err
	}
	staticPath := filepath.Join(p.dir, "traefik-static.yml")
	static := fmt.Sprintf(`api:
  insecure: true
  dashboard: false
entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"
providers:
  http:
    endpoint: "http://%s/config"
    pollInterval: "2s"
log:
  level: INFO
`, endpoint)
	if err := os.WriteFile(staticPath, []byte(static), 0o640); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, p.bin, "--configFile", staticPath)
	cmd.Stdout = logWriter{prefix: "[proxy] "}
	cmd.Stderr = logWriter{prefix: "[proxy] "}
	if err := cmd.Start(); err != nil {
		return err
	}
	p.cmd = cmd
	p.stopped = false
	go p.supervise(cmd)
	return nil
}

func (p *ProxySupervisor) supervise(cmd *exec.Cmd) {
	for {
		err := cmd.Wait()
		p.mu.Lock()
		stopped := p.stopped
		p.cmd = nil
		p.mu.Unlock()
		if stopped {
			return
		}
		log.Printf("[proxy] saiu: %v — reiniciando", err)
		time.Sleep(3 * time.Second)
		newCmd := exec.Command(cmd.Args[0], cmd.Args[1:]...)
		newCmd.Stdout = logWriter{prefix: "[proxy] "}
		newCmd.Stderr = logWriter{prefix: "[proxy] "}
		if err := newCmd.Start(); err != nil {
			log.Printf("[proxy] falha ao reiniciar: %v", err)
			time.Sleep(10 * time.Second)
			p.mu.Lock()
			stopped = p.stopped
			p.mu.Unlock()
			if stopped {
				return
			}
			continue
		}
		p.mu.Lock()
		p.cmd = newCmd
		p.mu.Unlock()
		cmd = newCmd
	}
}

func (p *ProxySupervisor) Stop() {
	p.mu.Lock()
	p.stopped = true
	cmd := p.cmd
	p.cmd = nil
	p.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
		cmd.Wait()
	}
}

func (p *ProxySupervisor) Running() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cmd != nil && p.cmd.Process != nil
}

type logWriter struct{ prefix string }

func (l logWriter) Write(b []byte) (int, error) {
	log.Print(l.prefix + strings.TrimRight(string(b), "\n"))
	return len(b), nil
}

func PortFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}
