package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const presenceTTL = 60 * time.Second

type presenceReq struct {
	Scope string `json:"scope"`
}

func (s *Server) handlePresenceJoin(w http.ResponseWriter, r *http.Request) {
	var req presenceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Scope) == "" {
		writeErr(w, 400, "scope obrigatório")
		return
	}
	member := claimsFrom(r).Subject
	if member == "" {
		member = claimsFrom(r).OrgID
	}
	if err := s.core.RT.Presence.Join(r.Context(), req.Scope, member, presenceTTL); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "joined", "scope": req.Scope})
}

func (s *Server) handlePresenceHeartbeat(w http.ResponseWriter, r *http.Request) {
	var req presenceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Scope) == "" {
		writeErr(w, 400, "scope obrigatório")
		return
	}
	member := claimsFrom(r).Subject
	if member == "" {
		member = claimsFrom(r).OrgID
	}
	if err := s.core.RT.Presence.Heartbeat(r.Context(), req.Scope, member, presenceTTL); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) handlePresenceLeave(w http.ResponseWriter, r *http.Request) {
	var req presenceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	member := claimsFrom(r).Subject
	if member == "" {
		member = claimsFrom(r).OrgID
	}
	if err := s.core.RT.Presence.Leave(r.Context(), req.Scope, member); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "left"})
}

func (s *Server) handlePresenceCount(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		writeErr(w, 400, "scope obrigatório")
		return
	}
	count, err := s.core.RT.Presence.Count(r.Context(), scope)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	members, _ := s.core.RT.Presence.Members(r.Context(), scope)
	writeJSON(w, 200, map[string]any{"scope": scope, "count": count, "members": members})
}
