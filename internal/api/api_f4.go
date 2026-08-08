package api

import (
	"encoding/json"
	"net/http"

	"aether/internal/domain"
)

func (s *Server) handleGetPolicy(w http.ResponseWriter, r *http.Request) {
	p, err := s.core.Store.GetPolicy(r.PathValue("id"))
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, p)
}

func (s *Server) handleSavePolicy(w http.ResponseWriter, r *http.Request) {
	var p domain.AppPolicy
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	p.AppID = r.PathValue("id")
	if err := s.core.SavePolicy(&p); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, p)
}

func (s *Server) handlePolicyEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.core.PolicyEvents(r.PathValue("id"))
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, events)
}

func (s *Server) handleListGitOps(w http.ResponseWriter, r *http.Request) {
	list, err := s.core.Store.ListGitOps(claimsFrom(r).OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, list)
}

func (s *Server) handleCreateGitOps(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		RepoURL   string `json:"repo_url"`
		Branch    string `json:"branch"`
		Path      string `json:"path"`
		ApplyMode string `json:"apply_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if req.Name == "" || req.RepoURL == "" {
		writeErr(w, 400, "name e repo_url obrigatórios")
		return
	}
	g, err := s.core.CreateGitOps(claimsFrom(r).OrgID, req.Name, req.RepoURL, req.Branch, req.Path, req.ApplyMode)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, g)
}

func (s *Server) handleSyncGitOps(w http.ResponseWriter, r *http.Request) {
	g, err := s.core.Store.GetGitOps(r.PathValue("id"))
	if err != nil {
		writeErr(w, 404, "gitops não encontrado")
		return
	}
	if err := s.core.SyncGitOps(g); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, g)
}

func (s *Server) handleDeleteGitOps(w http.ResponseWriter, r *http.Request) {
	if err := s.core.DeleteGitOps(r.PathValue("id")); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleListMirrors(w http.ResponseWriter, r *http.Request) {
	list, err := s.core.Store.ListMirrors()
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, list)
}

func (s *Server) handleCreateMirror(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string `json:"name"`
		Source        string `json:"source"`
		Dest          string `json:"dest"`
		DestTLSVerify bool   `json:"dest_tls_verify"`
		TagsFilter    string `json:"tags_filter"`
		Schedule      string `json:"schedule"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	m, err := s.core.CreateMirror(req.Name, req.Source, req.Dest, req.DestTLSVerify, req.TagsFilter, req.Schedule)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, m)
}

func (s *Server) handleRunMirror(w http.ResponseWriter, r *http.Request) {
	list, err := s.core.Store.ListMirrors()
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	for i := range list {
		if list[i].ID == r.PathValue("id") {
			m := &list[i]
			go s.core.RunMirror(r.Context(), m)
			writeJSON(w, 202, map[string]string{"status": "started"})
			return
		}
	}
	writeErr(w, 404, "mirror não encontrado")
}

func (s *Server) handleDeleteMirror(w http.ResponseWriter, r *http.Request) {
	if err := s.core.Store.DeleteMirror(r.PathValue("id")); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleNetQ(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.core.NetQStats())
}

func (s *Server) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	list, err := s.core.Store.ListSnapshots(claimsFrom(r).OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, list)
}

func (s *Server) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AppID  string `json:"app_id"`
		Volume string `json:"volume"`
		Name   string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if req.Volume == "" || req.Name == "" {
		writeErr(w, 400, "volume e name obrigatórios")
		return
	}
	sn, err := s.core.CreateVolumeSnapshot(r.Context(), claimsFrom(r).OrgID, req.AppID, req.Volume, req.Name)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, sn)
}

func (s *Server) handleRestoreSnapshot(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Volume string `json:"volume"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	volume := req.Volume
	if volume == "" {
		volume = "auto"
	}
	if err := s.core.RestoreVolumeSnapshot(r.Context(), r.PathValue("id"), volume); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "restored"})
}

func (s *Server) handleDeleteSnapshot(w http.ResponseWriter, r *http.Request) {
	if err := s.core.Store.DeleteSnapshot(r.PathValue("id")); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}
