package core

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"aether/internal/domain"
	"aether/internal/runtime"
	"aether/internal/security"
)

type agentHandlers struct {
	core *Core
}

func (c *Core) StartAgentServer() error {
	ca, err := security.EnsureCA(c.Cfg.CertsDir)
	if err != nil {
		return err
	}
	c.CA = ca
	srvCert, err := ca.ServerTLS()
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	h := &agentHandlers{core: c}
	mux.HandleFunc("POST /agent/v1/register", h.register)
	mux.HandleFunc("POST /agent/v1/heartbeat", h.auth(h.heartbeat))
	mux.HandleFunc("GET /agent/v1/commands", h.auth(h.commands))
	mux.HandleFunc("POST /agent/v1/events", h.auth(h.events))
	mux.HandleFunc("POST /agent/v1/exec", h.auth(h.exec))
	ln, err := net.Listen("tcp", c.Cfg.AgentAddr)
	if err != nil {
		return fmt.Errorf("agent listener: %w", err)
	}
	c.agentAddr = ln.Addr().String()
	srv := &http.Server{
		Handler: mux,
		TLSConfig: &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{srvCert},
			ClientAuth:   tls.RequestClientCert,
		},
	}
	c.agentSrv = srv
	go func() {
		if err := srv.ServeTLS(ln, "", ""); err != nil && err != http.ErrServerClosed {
			log.Printf("[agent] server: %v", err)
		}
	}()
	go c.watchServerHealth(context.Background())
	return nil
}

func (h *agentHandlers) agentID(r *http.Request) (string, error) {
	if len(r.TLS.PeerCertificates) == 0 {
		return "", fmt.Errorf("cert ausente")
	}
	cn := r.TLS.PeerCertificates[0].Subject.CommonName
	if !strings.HasPrefix(cn, "agent:") {
		return "", fmt.Errorf("CN inválido: %s", cn)
	}
	return strings.TrimPrefix(cn, "agent:"), nil
}

func (h *agentHandlers) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := h.agentID(r)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		srv, err := h.core.Store.GetServer(id)
		if err != nil || srv.Status == "removed" {
			http.Error(w, "server desconhecido", http.StatusUnauthorized)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), serverIDKey{}, id)))
	}
}

type serverIDKey struct{}

func serverIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(serverIDKey{}).(string)
	return id
}

type registerReq struct {
	Token    string   `json:"token"`
	Name     string   `json:"name"`
	Host     string   `json:"host"`
	Version  string   `json:"version"`
	Labels   []string `json:"labels"`
	CPUCores int      `json:"cpu_cores"`
	MemTotal int64    `json:"mem_total_bytes"`
}

func (h *agentHandlers) register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo inválido", http.StatusBadRequest)
		return
	}
	if req.Token == "" {
		http.Error(w, "token ausente", http.StatusBadRequest)
		return
	}
	sum := sha256.Sum256([]byte(req.Token))
	bound, err := h.core.Store.ConsumeServerToken(hex.EncodeToString(sum[:]))
	if err != nil {
		http.Error(w, "token inválido ou expirado", http.StatusUnauthorized)
		return
	}
	id := bound
	if id == "" {
		id = "srv-" + domain.NewID()
	}
	now := time.Now().UTC()
	srv := &domain.Server{
		ID:            id,
		Name:          req.Name,
		Host:          req.Host,
		Role:          "agent",
		Status:        "registered",
		Version:       req.Version,
		Labels:        req.Labels,
		CPUCores:      req.CPUCores,
		MemTotalBytes: req.MemTotal,
		LastHeartbeat: now,
		CreatedAt:     now,
	}
	if err := h.core.Store.CreateServer(srv); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	ident, err := h.core.CA.SignAgent(id, 365)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("[agent] server %s registrado (%s)", id, req.Name)
	h.core.Bus.Publish(context.Background(), "system", "", "server.registered", map[string]any{
		"server_id": id, "name": req.Name,
	}, nil)
	writeJSON(w, 200, map[string]any{
		"server_id": id,
		"cert_pem":  string(ident.CertPEM),
		"key_pem":   string(ident.KeyPEM),
		"ca_pem":    string(ident.CAPEM),
	})
}

type heartbeatReq struct {
	Load     float64 `json:"load"`
	CPUCores int     `json:"cpu_cores"`
	MemTotal int64   `json:"mem_total_bytes"`
	Version  string  `json:"version"`
}

func (h *agentHandlers) heartbeat(w http.ResponseWriter, r *http.Request) {
	var req heartbeatReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	id := serverIDFrom(r.Context())
	srv, _ := h.core.Store.GetServer(id)
	if srv != nil && srv.Status == "unhealthy" {
		h.core.Bus.Publish(context.Background(), "system", "", "server.recovered", map[string]any{
			"server_id": id, "name": srv.Name,
		}, nil)
	}
	if err := h.core.Store.UpdateServerHeartbeat(id, req.Load, req.CPUCores, req.MemTotal, "healthy", req.Version); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cmds, err := h.core.Store.DequeueServerCommands(id, 10)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, 200, map[string]any{"commands": cmds})
}

func (h *agentHandlers) commands(w http.ResponseWriter, r *http.Request) {
	cmds, err := h.core.Store.DequeueServerCommands(serverIDFrom(r.Context()), 10)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, 200, map[string]any{"commands": cmds})
}

type agentEvent struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

func (h *agentHandlers) events(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Events []agentEvent `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo inválido", http.StatusBadRequest)
		return
	}
	id := serverIDFrom(r.Context())
	for _, ev := range req.Events {
		h.core.applyAgentEvent(id, ev)
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (h *agentHandlers) exec(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command []string `json:"command"`
		Timeout int      `json:"timeout"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo inválido", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(req.Timeout)*time.Second)
	defer cancel()
	res, err := h.core.Driver.Exec(ctx, serverIDFrom(r.Context()), runtime.ExecRequest{Command: req.Command})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, 200, map[string]any{"stdout": string(res.Stdout), "exit_code": res.ExitCode})
}

func (c *Core) watchServerHealth(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			servers, err := c.Store.ListServers()
			if err != nil {
				continue
			}
			cutoff := time.Now().UTC().Add(-30 * time.Second)
			for _, srv := range servers {
				if srv.Status != "healthy" && srv.Status != "registered" {
					continue
				}
				if srv.LastHeartbeat.IsZero() || srv.LastHeartbeat.Before(cutoff) {
					_ = c.Store.UpdateServerHeartbeat(srv.ID, srv.Load, srv.CPUCores, srv.MemTotalBytes, "unhealthy", srv.Version)
					log.Printf("[agent] server %s marcado unhealthy", srv.Name)
					c.Bus.Publish(ctx, "system", "", "server.marked_unhealthy", map[string]any{
						"server_id": srv.ID, "name": srv.Name,
					}, nil)
				}
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
