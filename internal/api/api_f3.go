package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleRegistryGet(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.core.RegistrySettings()
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, cfg)
}

func (s *Server) handleRegistryEnable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	cfg, err := s.core.RegistryEnable(r.Context(), req.Enabled)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, cfg)
}

func (s *Server) handleRegistryImages(w http.ResponseWriter, r *http.Request) {
	imgs, err := s.core.RegistryImages(r.Context())
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, imgs)
}

func (s *Server) handleRegistryImageDelete(w http.ResponseWriter, r *http.Request) {
	repo := r.PathValue("repo")
	tag := r.PathValue("tag")
	if err := s.core.RegistryDelete(r.Context(), repo, tag); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleListWebhooks(w http.ResponseWriter, r *http.Request) {
	orgID := claimsFrom(r).OrgID
	hooks, err := s.core.Store.ListOutWebhooks(orgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, hooks)
}

func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string   `json:"name"`
		URL    string   `json:"url"`
		Secret string   `json:"secret"`
		Events []string `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	if req.Name == "" || req.URL == "" || len(req.Events) == 0 {
		writeErr(w, 400, "campos obrigatórios: name, url, events")
		return
	}
	hook, err := s.core.CreateOutWebhook(claimsFrom(r).OrgID, req.Name, req.URL, req.Secret, req.Events)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, hook)
}

func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	if err := s.core.Store.DeleteOutWebhook(r.PathValue("id")); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Write([]byte(s.core.MetricsText(r.Context())))
}

func (s *Server) handleListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := s.core.Store.ListServers()
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, servers)
}

func (s *Server) handleServerToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Name == "" {
		writeErr(w, 400, "name obrigatório")
		return
	}
	token, err := s.core.NewServerTokenFor(req.Name)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, map[string]string{"token": token, "core": s.core.AgentURL()})
}

func (s *Server) handleDeleteServer(w http.ResponseWriter, r *http.Request) {
	if err := s.core.Store.DeleteServer(r.PathValue("id")); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}
