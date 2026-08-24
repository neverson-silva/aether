package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

type Status struct {
	ready atomic.Bool
}

func (s *Status) SetReady(value bool) {
	s.ready.Store(value)
}

func (s *Status) Ready() bool {
	return s.ready.Load()
}

type Server struct {
	http    *http.Server
	metrics interface{ Render() string }
}

func NewServer(addr string, status *Status) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		if !status.Ready() {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	server := &Server{http: &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}}
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		if server.metrics == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(server.metrics.Render()))
	})
	return server
}

func (s *Server) SetMetrics(metrics interface{ Render() string }) {
	s.metrics = metrics
}

func (s *Server) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.http.Shutdown(shutdownCtx)
	}()
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
