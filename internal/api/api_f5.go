package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"aether/internal/core"
	"aether/internal/domain"
	"aether/internal/planner"
)

func (s *Server) handleGetBranding(w http.ResponseWriter, r *http.Request) {
	b, err := s.core.GetBranding(claimsFrom(r).OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, b)
}

func (s *Server) handleSaveBranding(w http.ResponseWriter, r *http.Request) {
	var b domain.Branding
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	b.OrgID = claimsFrom(r).OrgID
	if err := s.core.SaveBranding(&b); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, b)
}

func (s *Server) handleListPipelines(w http.ResponseWriter, r *http.Request) {
	list, err := s.core.Store.ListPipelines(claimsFrom(r).OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, list)
}

func (s *Server) handleCreatePipeline(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AppID   string                 `json:"app_id"`
		Name    string                 `json:"name"`
		Trigger string                 `json:"trigger"`
		Stages  []domain.PipelineStage `json:"stages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if req.Name == "" || len(req.Stages) == 0 {
		writeErr(w, 400, "name e stages obrigatórios")
		return
	}
	p, err := s.core.CreatePipeline(claimsFrom(r).OrgID, req.AppID, req.Name, req.Trigger, req.Stages)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, p)
}

func (s *Server) handleDeletePipeline(w http.ResponseWriter, r *http.Request) {
	if err := s.core.Store.DeletePipeline(r.PathValue("id")); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleRunPipeline(w http.ResponseWriter, r *http.Request) {
	run, err := s.core.RunPipeline(r.Context(), r.PathValue("id"), "manual")
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, run)
}

func (s *Server) handleListPipelineRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := s.core.Store.ListPipelineRuns(r.PathValue("id"), 30)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, runs)
}

func (s *Server) handleListClusters(w http.ResponseWriter, r *http.Request) {
	list, err := s.core.Store.ListClusters(claimsFrom(r).OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	servers, _ := s.core.Store.ListServers()
	out := make([]map[string]any, 0, len(list))
	for _, c := range list {
		var members []domain.Server
		for _, srv := range servers {
			if srv.ClusterID == c.ID {
				members = append(members, srv)
			}
		}
		out = append(out, map[string]any{
			"id": c.ID, "org_id": c.OrgID, "name": c.Name, "labels": c.Labels,
			"created_at": c.CreatedAt, "servers": members,
		})
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleCreateCluster(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string   `json:"name"`
		Labels []string `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if req.Name == "" {
		writeErr(w, 400, "name obrigatório")
		return
	}
	c := &domain.Cluster{
		ID:     "cl-" + domain.NewID(),
		OrgID:  claimsFrom(r).OrgID,
		Name:   req.Name,
		Labels: req.Labels,
	}
	if err := s.core.Store.CreateCluster(c); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, c)
}

func (s *Server) handleDeleteCluster(w http.ResponseWriter, r *http.Request) {
	if err := s.core.Store.DeleteCluster(r.PathValue("id")); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleClusterAddServer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ServerID string `json:"server_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if err := s.core.Store.SetServerCluster(req.ServerID, r.PathValue("id")); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "added"})
}

func (s *Server) handleClusterRemoveServer(w http.ResponseWriter, r *http.Request) {
	if err := s.core.Store.SetServerCluster(r.PathValue("serverID"), ""); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "removed"})
}

func (s *Server) handleListSSO(w http.ResponseWriter, r *http.Request) {
	list, err := s.core.Store.ListOIDCProviders(claimsFrom(r).OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, list)
}

func (s *Server) handleCreateSSO(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		Issuer       string `json:"issuer"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Scopes       string `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if req.Name == "" || req.Issuer == "" || req.ClientID == "" {
		writeErr(w, 400, "name, issuer e client_id obrigatórios")
		return
	}
	if _, err := s.core.DiscoverOIDC(r.Context(), req.Issuer); err != nil {
		writeErr(w, 400, "issuer inválido: "+err.Error())
		return
	}
	p, err := s.core.CreateOIDCProvider(claimsFrom(r).OrgID, req.Name, req.Issuer, req.ClientID, req.ClientSecret, req.Scopes)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, p)
}

func (s *Server) handleSSOAuthURL(w http.ResponseWriter, r *http.Request) {
	p, err := s.core.Store.GetOIDCProvider(r.PathValue("id"))
	if err != nil {
		writeErr(w, 404, "provider não encontrado")
		return
	}
	u, err := s.core.OIDCAuthURL(p)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"url": u})
}

func (s *Server) handleSSOCallback(w http.ResponseWriter, r *http.Request) {
	p, err := s.core.Store.GetOIDCProvider(r.PathValue("id"))
	if err != nil {
		writeErr(w, 404, "provider não encontrado")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		writeErr(w, 400, "code ausente: "+r.URL.Query().Get("error"))
		return
	}
	user, err := s.core.OIDCExchange(r.Context(), p, code)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	u, token, err := s.core.OIDCLogin(user)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "aether_sso_token", Value: token, Path: "/", HttpOnly: true, MaxAge: 86400})
	writeJSON(w, 200, map[string]any{
		"token": token, "user": u, "org_id": p.OrgID,
	})
}

func (s *Server) handleDeleteSSO(w http.ResponseWriter, r *http.Request) {
	if err := s.core.Store.DeleteOIDCProvider(r.PathValue("id")); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleSystemSummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.core.SystemSummary(r.Context(), claimsFrom(r).OrgID))
}

func (s *Server) handleAllCronJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.core.AllCronJobs(claimsFrom(r).OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, jobs)
}

func (s *Server) handleCertificates(w http.ResponseWriter, r *http.Request) {
	certs, err := s.core.Certificates(claimsFrom(r).OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, certs)
}

func (s *Server) triggeredBy(r *http.Request) string {
	if user, err := s.core.Store.GetUser(claimsFrom(r).Subject); err == nil {
		return user.Email
	}
	return "manual"
}

func (s *Server) handleAppStart(w http.ResponseWriter, r *http.Request) {
	if err := s.core.AppStart(r.PathValue("id")); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "started", "state": s.core.AppState(r.PathValue("id"))})
}

func (s *Server) handleAppStop(w http.ResponseWriter, r *http.Request) {
	if err := s.core.AppStop(r.PathValue("id")); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "stopped", "state": s.core.AppState(r.PathValue("id"))})
}

func (s *Server) handleAppRestart(w http.ResponseWriter, r *http.Request) {
	if err := s.core.AppRestart(r.PathValue("id")); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "restarted", "state": s.core.AppState(r.PathValue("id"))})
}

func (s *Server) handleAppRebuild(w http.ResponseWriter, r *http.Request) {
	dep, err := s.core.AppRebuild(r.PathValue("id"), s.triggeredBy(r))
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 202, dep)
}

func (s *Server) handleAppStates(w http.ResponseWriter, r *http.Request) {
	orgID := claimsFrom(r).OrgID
	states := map[string]string{}
	projects, _ := s.core.Store.ListProjects(orgID)
	for _, p := range projects {
		apps, _ := s.core.Store.ListAppsByProject(p.ID)
		for _, a := range apps {
			states[a.ID] = s.core.AppState(a.ID)
		}
	}
	writeJSON(w, 200, states)
}

func (s *Server) handleAppStatesStream(w http.ResponseWriter, r *http.Request) {
	flusher, wok := w.(http.Flusher)
	if !wok {
		writeErr(w, 500, "stream não suportado")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(200)

	ctx := r.Context()
	orgID := claimsFrom(r).OrgID
	allowed := map[string]bool{}
	projects, _ := s.core.Store.ListProjects(orgID)
	for _, p := range projects {
		apps, _ := s.core.Store.ListAppsByProject(p.ID)
		for _, a := range apps {
			allowed[a.ID] = true
		}
	}
	for id := range allowed {
		state := s.core.AppState(id)
		fmt.Fprintf(w, "event: state\ndata: %s\n\n", appStateJSON(id, state))
	}
	flusher.Flush()

	updates := make(chan [2]string, 256)
	stop, err := s.core.WatchAppStates("app:state:*", func(_ context.Context, appID string, state string) {
		if !allowed[appID] {
			return
		}
		select {
		case updates <- [2]string{appID, state}:
		default:
		}
	})
	if err != nil {
		return
	}
	defer stop()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case u := <-updates:
			fmt.Fprintf(w, "event: state\ndata: %s\n\n", appStateJSON(u[0], u[1]))
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func appStateJSON(appID, state string) string {
	b, _ := json.Marshal(map[string]string{"app_id": appID, "state": state})
	return string(b)
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if req.Name == "" || req.Email == "" || len(req.Password) < 8 {
		writeErr(w, 400, "name, email e senha (mín. 8) obrigatórios")
		return
	}
	user, token, err := s.core.Register(req.Email, req.Name, req.Password)
	if err != nil {
		if errors.Is(err, core.ErrAlreadyRegistered) {
			writeErr(w, 409, "platform already registered")
			return
		}
		writeErr(w, 500, err.Error())
		return
	}
	s.setAuthCookie(w, token)
	writeJSON(w, 201, map[string]any{
		"token": token,
		"user": map[string]any{
			"id": user.ID, "email": user.Email, "name": user.Name,
		},
	})
}

func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.core.AuthStatus())
}

func (s *Server) handlePublicSSO(w http.ResponseWriter, r *http.Request) {
	providers, err := s.core.Store.ListOIDCProviders("")
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	out := make([]map[string]string, 0, len(providers))
	for _, p := range providers {
		if p.Enabled {
			out = append(out, map[string]string{"id": p.ID, "name": p.Name})
		}
	}
	writeJSON(w, 200, out)
}

func (s *Server) handlePublicSSOAuthURL(w http.ResponseWriter, r *http.Request) {
	p, err := s.core.Store.GetOIDCProvider(r.PathValue("id"))
	if err != nil {
		writeErr(w, 404, "provider não encontrado")
		return
	}
	u, err := s.core.OIDCAuthURL(p)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"url": u})
}

func (s *Server) handleRenameProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if req.Name == "" {
		writeErr(w, 400, "name obrigatório")
		return
	}
	p, err := s.core.RenameProject(claimsFrom(r).OrgID, r.PathValue("id"), req.Name)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, p)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	if err := s.core.DeleteProject(claimsFrom(r).OrgID, r.PathValue("id")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleListEnvironments(w http.ResponseWriter, r *http.Request) {
	envs, err := s.core.EnvSummaries(r.PathValue("projectID"))
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, envs)
}

func (s *Server) handleCreateEnvironment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Color       string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	env, err := s.core.CreateEnvironment(r.PathValue("projectID"), req.Name, req.Description, req.Color)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, env)
}

func (s *Server) handleUpdateEnvironment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Color       string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	env, err := s.core.UpdateEnvironment(r.PathValue("projectID"), r.PathValue("environmentID"), req.Name, req.Description, req.Color)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, env)
}

func (s *Server) handleDeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	if err := s.core.DeleteEnvironment(r.PathValue("projectID"), r.PathValue("environmentID")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleSetDefaultEnvironment(w http.ResponseWriter, r *http.Request) {
	if err := s.core.SetDefaultEnvironment(r.PathValue("projectID"), r.PathValue("environmentID")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "default"})
}

func (s *Server) handleListEnvVars(w http.ResponseWriter, r *http.Request) {
	includeSecrets := r.URL.Query().Get("secrets") == "1"
	vars, err := s.core.ListEnvironmentVars(r.PathValue("projectID"), r.PathValue("environmentID"), includeSecrets)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, vars)
}

func (s *Server) handleSetEnvVar(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key    string `json:"key"`
		Value  string `json:"value"`
		Secret bool   `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	v, err := s.core.SetEnvironmentVar(r.PathValue("projectID"), r.PathValue("environmentID"), req.Key, req.Value, req.Secret)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, v)
}

func (s *Server) handleDeleteEnvVar(w http.ResponseWriter, r *http.Request) {
	if err := s.core.DeleteEnvironmentVar(r.PathValue("projectID"), r.PathValue("environmentID"), r.PathValue("key")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleReplaceEnvVars(w http.ResponseWriter, r *http.Request) {
	var req map[string]struct {
		Value  string `json:"value"`
		Secret bool   `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	n, err := s.core.ReplaceEnvironmentVars(r.PathValue("projectID"), r.PathValue("environmentID"), req)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]int{"saved": n})
}

func (s *Server) handleExportEnvVars(w http.ResponseWriter, r *http.Request) {
	vars, err := s.core.ListEnvironmentVars(r.PathValue("projectID"), r.PathValue("environmentID"), true)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(core.EnvTextOf(vars, nil)))
}

func (s *Server) handleImportEnvVars(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	entries, err := core.ParseEnvText(string(body))
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	n, err := s.core.ReplaceEnvironmentVars(r.PathValue("projectID"), r.PathValue("environmentID"), entries)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]int{"imported": n})
}

func (s *Server) handleEnvVarAudit(w http.ResponseWriter, r *http.Request) {
	audit, err := s.core.Store.ListVariableAudit(r.PathValue("environmentID"), 50)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, audit)
}

func (s *Server) handleListProjectVars(w http.ResponseWriter, r *http.Request) {
	includeSecrets := r.URL.Query().Get("secrets") == "1"
	vars, err := s.core.ListProjectVars(r.PathValue("projectID"), includeSecrets)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, vars)
}

func (s *Server) handleSetProjectVar(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key    string `json:"key"`
		Value  string `json:"value"`
		Secret bool   `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	v, err := s.core.SetProjectVar(r.PathValue("projectID"), req.Key, req.Value, req.Secret)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, v)
}

func (s *Server) handleDeleteProjectVar(w http.ResponseWriter, r *http.Request) {
	if err := s.core.DeleteProjectVar(r.PathValue("projectID"), r.PathValue("key")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleReplaceProjectVars(w http.ResponseWriter, r *http.Request) {
	var req map[string]struct {
		Value  string `json:"value"`
		Secret bool   `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	n, err := s.core.ReplaceProjectVars(r.PathValue("projectID"), req)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]int{"saved": n})
}

func (s *Server) handleExportProjectVars(w http.ResponseWriter, r *http.Request) {
	vars, err := s.core.ListProjectVars(r.PathValue("projectID"), true)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	text := ""
	for _, v := range vars {
		text += v.Key + "=" + v.Value + "\n"
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(text))
}

func (s *Server) handleImportProjectVars(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	entries, err := core.ParseEnvText(string(body))
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	n, err := s.core.ReplaceProjectVars(r.PathValue("projectID"), entries)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]int{"imported": n})
}

func (s *Server) handleProjectVarAudit(w http.ResponseWriter, r *http.Request) {
	audit, err := s.core.Store.ListVariableAuditByProject(r.PathValue("projectID"), 50)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, audit)
}

func (s *Server) handleDetectApp(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	if app.SourceType != domain.SourceGit || app.GitURL == "" {
		writeErr(w, 400, "detecção requer um repositório git")
		return
	}
	res, err := s.core.DetectFramework(r.Context(), app.GitURL, app.GitBranch)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, res)
}

func (s *Server) handleCompareDeployments(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	depA, err := s.core.Store.GetDeployment(r.URL.Query().Get("a"))
	if err != nil {
		writeErr(w, 400, "deployment a inválido")
		return
	}
	depB, err := s.core.Store.GetDeployment(r.URL.Query().Get("b"))
	if err != nil {
		writeErr(w, 400, "deployment b inválido")
		return
	}
	if depA.AppID != app.ID || depB.AppID != app.ID {
		writeErr(w, 400, "deployments não pertencem ao app")
		return
	}
	parse := func(snap string) map[string]string {
		out := map[string]string{}
		for _, line := range strings.Split(snap, "\n") {
			if i := strings.Index(line, "="); i > 0 {
				out[line[:i]] = line[i+1:]
			}
		}
		return out
	}
	envA, envB := parse(depA.EnvSnapshot), parse(depB.EnvSnapshot)
	added, removed, changed := []string{}, []string{}, []string{}
	for k, v := range envB {
		va, ok := envA[k]
		if !ok {
			added = append(added, k)
		} else if va != v {
			changed = append(changed, k)
		}
	}
	for k := range envA {
		if _, ok := envB[k]; !ok {
			removed = append(removed, k)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(changed)
	diff := map[string]any{
		"image":       map[string]string{"from": depA.ImageRef, "to": depB.ImageRef},
		"commit":      map[string]string{"from": depA.Commit, "to": depB.Commit},
		"status_a":    string(depA.Status),
		"status_b":    string(depB.Status),
		"env_added":   added,
		"env_removed": removed,
		"env_changed": changed,
	}
	writeJSON(w, 200, diff)
}

func (s *Server) handleValidateCompose(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	writeJSON(w, 200, core.ValidateComposeYAML(req.Content))
}

func (s *Server) handleDetectRepo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GitURL   string `json:"git_url"`
		Branch   string `json:"git_branch"`
		UploadID string `json:"upload_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if req.GitURL == "" {
		writeErr(w, 400, "git_url é obrigatório")
		return
	}
	res, err := s.core.DetectFramework(r.Context(), req.GitURL, req.Branch)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, res)
}

func (s *Server) handleAlertRules(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	rules, err := s.core.ListAlertRules(claims.OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, rules)
}

func (s *Server) handleCreateAlertRule(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	var req struct {
		Name      string  `json:"name"`
		Metric    string  `json:"metric"`
		Threshold float64 `json:"threshold"`
		WindowS   int     `json:"window_s"`
		Severity  string  `json:"severity"`
		TargetApp string  `json:"target_app"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	rule := &domain.AlertRule{OrgID: claims.OrgID, Name: req.Name, Metric: req.Metric, Threshold: req.Threshold, WindowS: req.WindowS, Severity: req.Severity, TargetApp: req.TargetApp, Enabled: true}
	if err := s.core.CreateAlertRule(rule); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, rule)
}

func (s *Server) handlePatchAlertRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if req.Enabled != nil {
		if err := s.core.SetAlertRuleEnabled(r.PathValue("id"), *req.Enabled); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteAlertRule(w http.ResponseWriter, r *http.Request) {
	if err := s.core.DeleteAlertRule(r.PathValue("id")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) handleAlertEvents(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	events, err := s.core.ListAlertEvents(claims.OrgID, limit)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, events)
}

func (s *Server) handleResolveAlert(w http.ResponseWriter, r *http.Request) {
	if err := s.core.ResolveAlertEvent(r.PathValue("id")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) handleSnapshotSchedules(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	schedules, err := s.core.ListSnapshotSchedules(claims.OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, schedules)
}

func (s *Server) handleCreateSnapshotSchedule(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	var req struct {
		AppID      string `json:"app_id"`
		Volume     string `json:"volume"`
		NamePrefix string `json:"name_prefix"`
		Cron       string `json:"cron"`
		Retention  int    `json:"retention"`
		Enabled    bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	sched := &domain.SnapshotSchedule{OrgID: claims.OrgID, AppID: req.AppID, Volume: req.Volume, NamePrefix: req.NamePrefix, Cron: req.Cron, Retention: req.Retention, Enabled: req.Enabled}
	if err := s.core.CreateSnapshotSchedule(sched); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, sched)
}

func (s *Server) handleDeleteSnapshotSchedule(w http.ResponseWriter, r *http.Request) {
	if err := s.core.DeleteSnapshotSchedule(r.PathValue("id")); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) handleUploadZip(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeErr(w, 400, "upload muito grande (máx 64MB)")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeErr(w, 400, "arquivo 'file' é obrigatório")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		writeErr(w, 500, "falha ao ler arquivo")
		return
	}
	name := header.Filename
	up, err := s.core.SaveZipUpload(name, data)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, up)
}

func (s *Server) handleCreatePlan(w http.ResponseWriter, r *http.Request) {
	plan, err := s.core.AnalyzeApp(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, plan)
}

func (s *Server) handleGetPlan(w http.ResponseWriter, r *http.Request) {
	plan, err := s.core.GetDeploymentPlan(r.PathValue("id"))
	if err != nil {
		writeErr(w, 404, "plano não gerado — rode a análise primeiro")
		return
	}
	writeJSON(w, 200, plan)
}

func (s *Server) handlePlanPreview(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Plan *planner.Plan `json:"plan"`
		Port int           `json:"port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if req.Plan == nil {
		writeErr(w, 400, "plan obrigatório")
		return
	}
	p := req.Plan
	if req.Port > 0 {
		p.ContainerPort = req.Port
	}
	if p.AppType == planner.TypeSPA || p.AppType == planner.TypeStatic {
		p.SPAFallback = true
	}
	writeJSON(w, 200, map[string]string{
		"dockerfile": planner.GenerateDockerfile(p),
		"nginx_conf": planner.GenerateNginxConf(p),
	})
}

func (s *Server) handleAnalyzeRepo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GitURL   string `json:"git_url"`
		Branch   string `json:"git_branch"`
		UploadID string `json:"upload_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if req.GitURL == "" && req.UploadID == "" {
		writeErr(w, 400, "git_url ou upload_id é obrigatório")
		return
	}
	plan, err := s.core.AnalyzeRepo(r.Context(), req.GitURL, req.Branch, req.UploadID)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, plan)
}
