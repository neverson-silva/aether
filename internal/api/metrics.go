package api

import "net/http"

func (s *Server) handleRuntimeMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	m := s.core.RT.Metrics(ctx)
	writeJSON(w, 200, m)
}
