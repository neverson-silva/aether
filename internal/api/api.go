package api

import (
	"aether/internal/db"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"aether/internal/core"
	"aether/internal/domain"
	"aether/internal/druntime/pubsub"
	"aether/internal/druntime/ratelimit"
	"aether/internal/git"
	"aether/internal/rbac"
	"aether/internal/security"
)

type Server struct {
	core   *core.Core
	webDir string
}

func New(c *core.Core, webDir string) *Server {
	return &Server{core: c, webDir: webDir}
}

type ctxKey string

const claimsKey ctxKey = "claims"

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		health := db.Health(s.core.DB)
		if health["db"] != "up" {
			writeJSON(w, 503, health)
			return
		}
		writeJSON(w, 200, health)
	})

	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)

	mux.HandleFunc("GET /api/v1/me", s.auth(s.handleMe))
	mux.HandleFunc("POST /api/v1/upload/zip", s.auth(s.perm("app.write", s.handleUploadZip)))

	mux.HandleFunc("GET /api/v1/members", s.auth(s.perm("org.read", s.handleListMembers)))
	mux.HandleFunc("POST /api/v1/members", s.auth(s.perm("member.write", s.handleAddMember)))
	mux.HandleFunc("PATCH /api/v1/members/{userID}", s.auth(s.perm("member.write", s.handleUpdateMember)))

	mux.HandleFunc("GET /api/v1/api-keys", s.auth(s.perm("org.read", s.handleListKeys)))
	mux.HandleFunc("POST /api/v1/api-keys", s.auth(s.perm("key.write", s.handleCreateKey)))
	mux.HandleFunc("DELETE /api/v1/api-keys/{id}", s.auth(s.perm("key.write", s.handleDeleteKey)))

	mux.HandleFunc("GET /api/v1/projects", s.auth(s.perm("app.read", s.handleListProjects)))
	mux.HandleFunc("POST /api/v1/projects", s.auth(s.perm("app.write", s.handleCreateProject)))
	mux.HandleFunc("GET /api/v1/organizations", s.auth(s.perm("org.read", s.handleListMyOrgs)))
	mux.HandleFunc("POST /api/v1/organizations", s.auth(s.perm("org.write", s.handleCreateOrg)))
	mux.HandleFunc("GET /api/v1/organizations/{id}", s.auth(s.perm("org.read", s.handleGetOrg)))
	mux.HandleFunc("PATCH /api/v1/organizations/{id}", s.auth(s.perm("org.write", s.handleUpdateOrg)))
	mux.HandleFunc("DELETE /api/v1/organizations/{id}", s.auth(s.perm("org.write", s.handleDeleteOrg)))
	mux.HandleFunc("GET /api/v1/organizations/{id}/members", s.auth(s.perm("member.read", s.handleListOrgMembers)))
	mux.HandleFunc("POST /api/v1/organizations/{id}/members", s.auth(s.perm("member.write", s.handleAddOrgMember)))
	mux.HandleFunc("PATCH /api/v1/organizations/{id}/members/{userId}", s.auth(s.perm("member.write", s.handleUpdateOrgMember)))
	mux.HandleFunc("DELETE /api/v1/organizations/{id}/members/{userId}", s.auth(s.perm("member.write", s.handleRemoveOrgMember)))
	mux.HandleFunc("POST /api/v1/organizations/{id}/members/{userId}/projects/{projectId}", s.auth(s.perm("member.write", s.handleSetProjectAssignment)))
	mux.HandleFunc("DELETE /api/v1/organizations/{id}/members/{userId}/projects/{projectId}", s.auth(s.perm("member.write", s.handleRemoveProjectAssignment)))
	mux.HandleFunc("GET /api/v1/organizations/{id}/audit", s.auth(s.perm("org.read", s.handleOrgAudit)))
	mux.HandleFunc("GET /api/v1/apps", s.auth(s.perm("app.read", s.handleListApps)))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/apps", s.auth(s.perm("app.write", s.handleCreateApp)))
	mux.HandleFunc("GET /api/v1/apps/{id}", s.auth(s.perm("app.read", s.handleGetApp)))
	mux.HandleFunc("PATCH /api/v1/apps/{id}", s.auth(s.perm("app.write", s.handleUpdateApp)))
	mux.HandleFunc("DELETE /api/v1/apps/{id}", s.auth(s.perm("app.write", s.handleDeleteApp)))
	mux.HandleFunc("POST /api/v1/apps/{id}/deploy", s.auth(s.perm("app.deploy", s.handleDeploy)))
	mux.HandleFunc("POST /api/v1/apps/{id}/rollback", s.auth(s.perm("app.deploy", s.handleRollback)))
	mux.HandleFunc("GET /api/v1/apps/{id}/deployments", s.auth(s.perm("app.read", s.handleListDeployments)))
	mux.HandleFunc("GET /api/v1/apps/{id}/deployments/compare", s.auth(s.perm("app.read", s.handleCompareDeployments)))
	mux.HandleFunc("GET /api/v1/apps/{id}/deployments/{depID}/log", s.auth(s.perm("app.read", s.handleDeploymentLog)))
	mux.HandleFunc("GET /api/v1/apps/{id}/compose", s.auth(s.perm("app.read", s.handleAppCompose)))
	mux.HandleFunc("GET /api/v1/apps/{id}/spec", s.auth(s.perm("app.read", s.handleAppSpec)))
	mux.HandleFunc("GET /api/v1/apps/{id}/export", s.auth(s.perm("app.read", s.handleExportRuntime)))
	mux.HandleFunc("GET /api/v1/apps/{id}/deployments/{depID}/compose", s.auth(s.perm("app.read", s.handleDeploymentCompose)))
	mux.HandleFunc("POST /api/v1/apps/{id}/compose/import", s.auth(s.perm("app.write", s.handleImportCompose)))
	mux.HandleFunc("GET /api/v1/deployments/{id}", s.auth(s.perm("app.read", s.handleGetDeployment)))
	mux.HandleFunc("GET /api/v1/apps/{id}/logs", s.auth(s.perm("app.read", s.handleLogs)))
	mux.HandleFunc("GET /api/v1/apps/{id}/logs/history", s.auth(s.perm("app.read", s.handleLogHistory)))
	mux.HandleFunc("GET /api/v1/apps/{id}/stats", s.auth(s.perm("app.read", s.handleStats)))
	mux.HandleFunc("GET /api/v1/apps/{id}/timeline", s.auth(s.perm("app.read", s.handleTimeline)))
	mux.HandleFunc("PUT /api/v1/apps/{id}/env", s.auth(s.perm("app.env", s.handleSetEnv)))
	mux.HandleFunc("DELETE /api/v1/apps/{id}/env/{name}", s.auth(s.perm("app.env", s.handleDeleteEnv)))
	mux.HandleFunc("POST /api/v1/apps/{id}/domains", s.auth(s.perm("app.domain", s.handleAddDomain)))
	mux.HandleFunc("GET /api/v1/apps/{id}/domains", s.auth(s.perm("app.read", s.handleListDomains)))
	mux.HandleFunc("DELETE /api/v1/apps/{id}/domains/{host}", s.auth(s.perm("app.domain", s.handleRemoveDomain)))
	mux.HandleFunc("PUT /api/v1/apps/{id}/webhook", s.auth(s.perm("app.write", s.handleSetWebhook)))

	mux.HandleFunc("GET /api/v1/backups", s.auth(s.perm("backup.read", s.handleListBackups)))
	mux.HandleFunc("POST /api/v1/backups", s.auth(s.perm("backup.write", s.handleCreateBackup)))
	mux.HandleFunc("POST /api/v1/backups/{id}/restore", s.auth(s.perm("backup.write", s.handleRestoreBackup)))

	mux.HandleFunc("GET /api/v1/events", s.auth(s.perm("org.read", s.handleEvents)))
	mux.HandleFunc("POST /api/v1/webhooks/github/{appID}", s.handleGitHubWebhook)

	s.registerF2Routes(mux)

	mux.HandleFunc("/", s.handleWeb)
	return s.rateLimit(mux)
}

func (s *Server) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if !strings.HasPrefix(path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		ctx := r.Context()
		ip := clientIP(r)
		if path == "/api/v1/auth/login" {
			if d, err := s.core.RT.RateLimit.Allow(ctx, ratelimit.TipIP, ip, 1, 0.5, 10); err == nil && !d.Allowed {
				w.Header().Set("Retry-After", "60")
				writeErr(w, 429, "muitas tentativas de login; aguarde e tente novamente")
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if strings.Contains(path, "/events/stream") || strings.Contains(path, "/states/stream") ||
			strings.Contains(path, "/stats") || strings.Contains(path, "/logs") || strings.Contains(path, "/ws/") {
			next.ServeHTTP(w, r)
			return
		}
		if strings.Contains(path, "/webhooks/") {
			if d, err := s.core.RT.RateLimit.Allow(ctx, ratelimit.TipWebhook, ip, 1, 1, 60); err == nil && !d.Allowed {
				w.Header().Set("Retry-After", "60")
				writeErr(w, 429, "rate limit de webhook excedido")
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if d, err := s.core.RT.RateLimit.Allow(ctx, ratelimit.TipRoute, path, 1, 5, 300); err == nil && !d.Allowed {
			w.Header().Set("Retry-After", "60")
			writeErr(w, 429, "rate limit da rota excedido")
			return
		}
		d, err := s.core.RT.RateLimit.Allow(ctx, ratelimit.TipIP, ip, 1, 10, 600)
		if err == nil {
			w.Header().Set("X-RateLimit-Limit", "600")
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(d.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(d.ResetIn).Unix(), 10))
			if !d.Allowed {
				w.Header().Set("Retry-After", strconv.Itoa(int(d.ResetIn.Seconds())+1))
				writeErr(w, 429, "rate limit excedido: 600 requests/min")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.TrimSpace(strings.Split(ip, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := s.authCookieValue(r)
		if token == "" {
			header := r.Header.Get("Authorization")
			token = strings.TrimPrefix(header, "Bearer ")
			token = strings.TrimSpace(token)
		}
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			writeErr(w, http.StatusUnauthorized, "token ausente")
			return
		}
		var claims *security.Claims
		if strings.HasPrefix(token, "ak_") {
			c, err := s.core.VerifyAPIKey(token)
			if err != nil {
				writeErr(w, http.StatusUnauthorized, "chave inválida")
				return
			}
			claims = c
		} else {
			c, err := s.core.VerifyToken(token)
			if err != nil {
				writeErr(w, http.StatusUnauthorized, "token inválido ou expirado")
				return
			}
			claims = c
		}
		// Resolve a organização ativa (multi-tenancy): header X-Aether-Org
		// troca de organização sem novo login; padrão é a org do token.
		reqOrg := r.Header.Get("X-Aether-Org")
		if reqOrg == "" {
			reqOrg = claims.OrgID
		}
		// Global ADMIN pode acessar qualquer organização; demais precisam ser membros.
		if claims.GlobalRole != domain.GlobalAdmin {
			if _, err := s.core.Store.GetMember(reqOrg, claims.Subject); err != nil {
				writeErr(w, http.StatusForbidden, "fora do escopo da organização")
				return
			}
			if m, err := s.core.Store.GetMember(reqOrg, claims.Subject); err == nil {
				claims.Role = string(m.Role)
			}
		} else {
			claims.Role = string(domain.RoleOwner)
		}
		claims.OrgID = reqOrg
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next(w, r.WithContext(ctx))
	}
}

func (s *Server) perm(permission string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFrom(r)
		role := domain.Role(claims.Role)
		if !rbac.Can(role, permission) {
			writeErr(w, http.StatusForbidden, "permissão negada: "+permission)
			return
		}
		next(w, r)
	}
}

func claimsFrom(r *http.Request) *security.Claims {
	claims, _ := r.Context().Value(claimsKey).(*security.Claims)
	return claims
}

// canAccessProject centraliza a autorização por projeto:
// Owner/Admin veem tudo; Member/Viewer apenas projetos atribuídos.
func (s *Server) canAccessProject(orgID, userID, projectID string, role domain.Role) bool {
	if role.AtLeast(domain.RoleAdmin) {
		return true
	}
	ok, _ := s.core.Store.ProjectAssigned(orgID, userID, projectID)
	return ok
}

// projectForOrg valida que o projeto pertence à organização ativa e que o
// usuário tem acesso; retorna o projeto.
func (s *Server) projectForOrg(w http.ResponseWriter, r *http.Request, projectID string) (*domain.Project, bool) {
	claims := claimsFrom(r)
	project, err := s.core.Store.GetProject(projectID)
	if err != nil {
		writeErr(w, http.StatusNotFound, "projeto não encontrado")
		return nil, false
	}
	if project.OrgID != claims.OrgID {
		writeErr(w, http.StatusForbidden, "fora do escopo da organização")
		return nil, false
	}
	if !s.canAccessProject(claims.OrgID, claims.Subject, project.ID, domain.Role(claims.Role)) {
		writeErr(w, http.StatusForbidden, "projeto não atribuído a você")
		return nil, false
	}
	return project, true
}

func (s *Server) appForOrg(w http.ResponseWriter, r *http.Request, id string) (*domain.App, bool) {
	app, err := s.core.Store.GetApp(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "aplicação não encontrada")
		return nil, false
	}
	claims := claimsFrom(r)
	if app.OrgID != claims.OrgID {
		writeErr(w, http.StatusForbidden, "fora do escopo da organização")
		return nil, false
	}
	if !s.canAccessProject(claims.OrgID, claims.Subject, app.ProjectID, domain.Role(claims.Role)) {
		writeErr(w, http.StatusForbidden, "projeto não atribuído a você")
		return nil, false
	}
	return app, true
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	user, token, err := s.core.Login(req.Email, req.Password, req.Code)
	if err != nil {
		if errors.Is(err, core.ErrMFARequired) {
			writeJSON(w, http.StatusOK, map[string]string{"mfa_required": "true"})
			return
		}
		writeErr(w, http.StatusUnauthorized, err.Error())
		return
	}
	s.setAuthCookie(w, token)
	writeJSON(w, 200, map[string]any{
		"token": token,
		"user": map[string]any{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.clearAuthCookie(w)
	writeJSON(w, 200, map[string]string{"status": "logged_out"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	user, err := s.core.Store.GetUser(claims.Subject)
	if err != nil {
		writeErr(w, http.StatusNotFound, "usuário não encontrado")
		return
	}
	org, err := s.core.Store.GetOrg(claims.OrgID)
	if err != nil {
		org = nil
	}
	orgName := ""
	if org != nil {
		orgName = org.Name
	}
	// organizações do usuário (multi-tenancy)
	memberships, _ := s.core.Store.ListOrgsForUser(claims.Subject)
	orgs := make([]map[string]any, 0, len(memberships))
	for _, m := range memberships {
		o, err := s.core.Store.GetOrg(m.OrgID)
		if err != nil {
			continue
		}
		orgs = append(orgs, map[string]any{
			"id":    o.ID,
			"slug":  o.Slug,
			"name":  o.Name,
			"color": o.Color,
			"role":  m.Role,
		})
	}
	writeJSON(w, 200, map[string]any{
		"id":          user.ID,
		"email":       user.Email,
		"name":        user.Name,
		"global_role": user.GlobalRole,
		"org": map[string]any{
			"id":   claims.OrgID,
			"name": orgName,
			"role": claims.Role,
		},
		"organizations": orgs,
	})
}

func (s *Server) handleListMembers(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	members, err := s.core.Store.ListMembers(claims.OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	var out []map[string]any
	for _, m := range members {
		u, err := s.core.Store.GetUser(m.UserID)
		if err != nil {
			continue
		}
		out = append(out, map[string]any{
			"user_id": m.UserID,
			"email":   u.Email,
			"name":    u.Name,
			"role":    m.Role,
		})
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleAddMember(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		writeErr(w, 400, "email obrigatório")
		return
	}
	role := domain.Role(req.Role)
	switch role {
	case domain.RoleOwner, domain.RoleAdmin, domain.RoleDeveloper, domain.RoleViewer:
	default:
		writeErr(w, 400, "papel inválido")
		return
	}
	user, err := s.core.Store.GetUserByEmail(req.Email)
	created := false
	if err != nil {
		if len(req.Password) < 8 {
			writeErr(w, 400, "senha obrigatória (mín. 8) para novo usuário")
			return
		}
		if req.Name == "" {
			req.Name = req.Email
		}
		user, err = s.core.CreateMemberUser(claims.OrgID, req.Email, req.Name, req.Password, string(role))
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		created = true
	} else {
		if err := s.core.Store.CreateMember(&domain.Member{OrgID: claims.OrgID, UserID: user.ID, Role: role}); err != nil {
			writeErr(w, 409, "membro já existe")
			return
		}
	}
	s.core.Audit(claims.OrgID, claims.Subject, "member.created", "member", user.ID, req.Email)
	writeJSON(w, 201, map[string]any{"status": "added", "created": created, "user_id": user.ID})
}

func (s *Server) handleUpdateMember(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	role := domain.Role(req.Role)
	switch role {
	case domain.RoleOwner, domain.RoleAdmin, domain.RoleDeveloper, domain.RoleViewer:
	default:
		writeErr(w, 400, "papel inválido")
		return
	}
	userID := r.PathValue("userID")
	if userID == claims.Subject && role != domain.RoleOwner {
		writeErr(w, 400, "não é possível rebaixar o próprio papel")
		return
	}
	if err := s.core.Store.SetMemberRole(claims.OrgID, userID, role); err != nil {
		writeErr(w, 404, "membro não encontrado")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "updated"})
}

func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	keys, err := s.core.Store.ListApiKeys(claims.OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	var out []map[string]any
	for _, k := range keys {
		out = append(out, map[string]any{
			"id":         k.ID,
			"name":       k.Name,
			"scopes":     k.Scopes,
			"created_at": k.CreatedAt,
		})
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleCreateKey(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	var req struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if req.Name == "" {
		req.Name = "default"
	}
	key, err := s.core.CreateAPIKey(claims.OrgID, claims.Subject, req.Name, req.Scopes)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, map[string]string{"key": key})
}

func (s *Server) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	id := r.PathValue("id")
	res, err := s.core.DB.Exec(`DELETE FROM api_keys WHERE id=? AND org_id=?`, id, claims.OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeErr(w, 404, "chave não encontrada")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	projects, err := s.core.Store.ListProjects(claims.OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	// Member/Viewer veem apenas projetos atribuídos.
	if role := domain.Role(claims.Role); !role.AtLeast(domain.RoleAdmin) {
		assigned, aerr := s.core.Store.ListProjectAssignments(claims.OrgID, claims.Subject)
		if aerr != nil {
			writeErr(w, 500, aerr.Error())
			return
		}
		set := make(map[string]bool, len(assigned))
		for _, pid := range assigned {
			set[pid] = true
		}
		filtered := projects[:0]
		for _, p := range projects {
			if set[p.ID] {
				filtered = append(filtered, p)
			}
		}
		projects = filtered
	}
	if projects == nil {
		projects = []domain.Project{}
	}
	writeJSON(w, 200, projects)
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Color       string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeErr(w, 400, "nome obrigatório")
		return
	}
	project, err := s.core.CreateProject(claims.OrgID, req.Name)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if req.Description != "" || req.Color != "" {
		s.core.Store.UpdateProjectDetails(project.ID, req.Description, req.Color)
	}
	s.core.Audit(claims.OrgID, claims.Subject, "project.created", "project", project.ID, project.Name)
	writeJSON(w, 201, project)
}

func (s *Server) handleListApps(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	var cached []domain.App
	if s.cacheGetJSON(appsCacheKey(claims.OrgID), &cached) {
		writeJSON(w, 200, cached)
		return
	}
	apps, err := s.core.Store.ListApps(claims.OrgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if apps == nil {
		apps = []domain.App{}
	}
	s.cacheSetJSON(appsCacheKey(claims.OrgID), apps, cacheAppsTTL)
	writeJSON(w, 200, apps)
}

func (s *Server) handleCreateApp(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	projectID := r.PathValue("projectID")
	var req struct {
		Name           string                 `json:"name"`
		SourceType     string                 `json:"source_type"`
		Image          string                 `json:"image"`
		GitURL         string                 `json:"git_url"`
		GitBranch      string                 `json:"git_branch"`
		Dockerfile     string                 `json:"dockerfile"`
		Port           int                    `json:"port"`
		Resources      domain.Resources       `json:"resources"`
		HealthCheck    domain.HealthCheck     `json:"health_check"`
		Volumes        []domain.Volume        `json:"volumes"`
		EnvironmentID  string                 `json:"environment_id"`
		UploadID       string                 `json:"upload_id"`
		InstallCommand string                 `json:"install_command"`
		BuildCommand   string                 `json:"build_command"`
		StartCommand   string                 `json:"start_command"`
		RootFolder     string                 `json:"root_folder"`
		DistFolder     string                 `json:"dist_folder"`
		WatchPaths     string                 `json:"watch_paths"`
		Plan           *domain.DeploymentPlan `json:"plan"`
		Env            []struct {
			Name   string `json:"name"`
			Value  string `json:"value"`
			Secret bool   `json:"secret"`
		} `json:"env"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeErr(w, 400, "nome obrigatório")
		return
	}
	envID := req.EnvironmentID
	if envID == "" {
		if def, err := s.core.DefaultEnvironment(projectID); err == nil {
			envID = def.ID
		}
	}
	app := &domain.App{
		ID:            domain.NewID(),
		ProjectID:     projectID,
		EnvironmentID: envID,
		Name:          req.Name,
		SourceType:    domain.SourceType(req.SourceType),
		Image:         req.Image,
		GitURL:        req.GitURL,
		GitBranch:     req.GitBranch,
		Dockerfile:    req.Dockerfile,
		Port:          req.Port,
		Resources:     req.Resources,
		HealthCheck:   req.HealthCheck,
		Volumes:       req.Volumes,
	}
	if app.SourceType == "" {
		app.SourceType = domain.SourceImage
	}
	if app.GitBranch == "" {
		app.GitBranch = "main"
	}
	if app.SourceType == domain.SourceImage && app.Image == "" {
		writeErr(w, 400, "imagem obrigatória para source_type=image")
		return
	}
	if app.SourceType == domain.SourceGit && app.GitURL == "" && req.UploadID == "" {
		writeErr(w, 400, "git_url ou upload necessário para source_type=git")
		return
	}
	app.UploadID = req.UploadID
	app.InstallCommand = req.InstallCommand
	app.BuildCommand = req.BuildCommand
	app.StartCommand = req.StartCommand
	app.RootFolder = req.RootFolder
	app.DistFolder = req.DistFolder
	app.WatchPaths = req.WatchPaths
	if err := s.core.CreateApp(claims.OrgID, app); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if req.Plan != nil {
		plan := *req.Plan
		plan.ID = domain.NewID()
		plan.AppID = app.ID
		if plan.DetectedAt.IsZero() {
			plan.DetectedAt = time.Now().UTC()
		}
		if plan.CreatedAt.IsZero() {
			plan.CreatedAt = time.Now().UTC()
		}
		_ = s.core.Store.SaveDeploymentPlan(&plan)
	}
	for _, e := range req.Env {
		if strings.TrimSpace(e.Name) == "" {
			continue
		}
		if err := s.core.Store.SetEnvVar(app.ID, e.Name, e.Value, e.Secret); err != nil {
			continue
		}
	}
	s.cacheInvalidate(appsCacheKey(claimsFrom(r).OrgID))
	s.core.EmitOrg(claimsFrom(r).OrgID, "service.created", "🆕 "+app.Name+" created", map[string]any{
		"service_id": app.ID, "service_name": app.Name, "project_id": app.ProjectID,
	})
	writeJSON(w, 201, app)
}

func (s *Server) handleGetApp(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	env, err := s.core.Store.ListEnvVars(app.ID)
	if err == nil {
		includeSecrets := r.URL.Query().Get("secrets") == "1"
		appEnv := make([]map[string]any, 0, len(env))
		for _, e := range env {
			val := ""
			if !e.Secret {
				val = string(e.Value)
			} else if includeSecrets {
				if dec, derr := s.core.Secrets.DecryptString(string(e.Value)); derr == nil {
					val = dec
				}
			}
			appEnv = append(appEnv, map[string]any{"name": e.Name, "value": val, "secret": e.Secret})
		}
		writeJSON(w, 200, map[string]any{
			"app":              app,
			"env":              appEnv,
			"internal_host":    s.core.InternalHost(app),
			"internal_network": s.core.InternalNetwork(app),
		})
		return
	}
	writeJSON(w, 200, app)
}

func (s *Server) handleUpdateApp(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	var req domain.App
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if req.Image != "" {
		app.Image = req.Image
	}
	if req.Name != "" {
		app.Name = req.Name
	}
	if req.GitURL != "" {
		app.GitURL = req.GitURL
	}
	if req.GitBranch != "" {
		app.GitBranch = req.GitBranch
	}
	if req.Dockerfile != "" {
		app.Dockerfile = req.Dockerfile
	}
	if req.BuildType != "" {
		app.BuildType = req.BuildType
	}
	if req.PreviewDomain != "" {
		app.PreviewDomain = req.PreviewDomain
	}
	if req.Port != 0 {
		app.Port = req.Port
	}
	if req.Resources.CPUs != "" || req.Resources.MemMB != 0 {
		app.Resources = req.Resources
	}
	if req.HealthCheck.Enabled || req.HealthCheck.Path != "" {
		app.HealthCheck = req.HealthCheck
	}
	if req.ImageRetention > 0 {
		app.ImageRetention = req.ImageRetention
	}
	if err := s.core.Store.UpdateApp(app); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	s.cacheInvalidate(appsCacheKey(claimsFrom(r).OrgID))
	writeJSON(w, 200, app)
}

func (s *Server) handleDeleteApp(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	s.core.StopAppLogCollector(app.ID)
	s.cacheInvalidate(appsCacheKey(claimsFrom(r).OrgID))
	deploys, err := s.core.Store.ListDeployments(app.ID, 1)
	if err == nil && len(deploys) > 0 && deploys[0].ContainerID != "" {
		s.core.Driver.Remove(ctx, deploys[0].ContainerID, true)
	}
	if len(deploys) > 0 {
		_ = s.core.ComposeDownFor(app, &deploys[0])
	}
	volumes := s.core.Store.ListAppVolumes(app.ID)
	if err := s.core.Store.DeleteApp(app.ID); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	for _, v := range volumes {
		if v != "" {
			_ = s.core.Driver.VolumeRemove(ctx, v)
		}
	}
	if app.SourceType == domain.SourceGit {
		_ = s.core.RemoveAppImages(ctx, app)
	}
	for _, d := range appDomains(s.core, app.ID) {
		s.core.Net.RemoveRoute(d)
	}
	s.core.EmitOrg(claimsFrom(r).OrgID, "service.deleted", "🗑 "+app.Name+" deleted", map[string]any{
		"service_id": app.ID, "service_name": app.Name, "project_id": app.ProjectID,
	})
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func appDomains(c *core.Core, appID string) []string {
	domains, err := c.Store.ListDomains(appID)
	if err != nil {
		return nil
	}
	var out []string
	for _, d := range domains {
		out = append(out, d.Host)
	}
	return out
}

func (s *Server) handleDeploy(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	dep, err := s.core.Deploy(app.ID, core.DeployOpts{Trigger: "api", TriggeredBy: s.triggeredBy(r)})
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 202, dep)
}

func (s *Server) handleDeploymentLog(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	dep, err := s.core.Store.GetDeployment(r.PathValue("depID"))
	if err != nil || dep.AppID != app.ID {
		writeErr(w, 404, "deployment não encontrado")
		return
	}
	data, err := s.core.ReadDeploymentLog(app.Name, dep.Number)
	if err != nil {
		writeErr(w, 404, "log indisponível")
		return
	}
	writeJSON(w, 200, map[string]any{
		"number":  dep.Number,
		"status":  dep.Status,
		"error":   dep.Error,
		"content": string(data),
	})
}

func (s *Server) handleRollback(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	dep, err := s.core.RollbackBy(app.ID, s.triggeredBy(r))
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 202, dep)
}

func (s *Server) handleListDeployments(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	limit := 25
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	deploys, err := s.core.Store.ListDeployments(app.ID, limit)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, deploys)
}

func (s *Server) handleGetDeployment(w http.ResponseWriter, r *http.Request) {
	dep, err := s.core.Store.GetDeployment(r.PathValue("id"))
	if err != nil {
		writeErr(w, 404, "deployment não encontrado")
		return
	}
	app, err := s.core.Store.GetApp(dep.AppID)
	if err != nil {
		writeErr(w, 404, "aplicação não encontrada")
		return
	}
	if app.OrgID != claimsFrom(r).OrgID {
		writeErr(w, 403, "fora do escopo da organização")
		return
	}
	writeJSON(w, 200, dep)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}

	flusher, wok := w.(http.Flusher)
	if !wok {
		writeErr(w, 500, "stream não suportado")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(200)

	deploys, derr := s.core.Store.ListDeployments(app.ID, 1)
	if derr == nil && len(deploys) > 0 {
		dep := &deploys[0]
		if data, rerr := s.core.ReadDeploymentLog(app.Name, dep.Number); rerr == nil && len(data) > 0 {
			writeSSE(w, flusher, data)
		}
	}

	ctx := r.Context()
	channel := "logs:app:" + app.ID
	msgs := make(chan []byte, 256)
	sub, err := s.core.RT.PubSub.Subscribe(ctx, channel, func(_ context.Context, m pubsub.Message) {
		select {
		case msgs <- m.Data:
		default:
		}
	}, pubsub.WithBuffer(256))
	if err == nil {
		defer sub.Unsubscribe()
	}

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case chunk := <-msgs:
			writeSSE(w, flusher, chunk)
		}
	}
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, data []byte) {
	for _, line := range strings.Split(string(data), "\n") {
		fmt.Fprintf(w, "data: %s\n", line)
	}
	fmt.Fprintf(w, "\n")
	flusher.Flush()
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	dep, err := s.core.Store.LastReadyDeployment(app.ID, 1<<62)
	if err != nil || dep.ContainerID == "" {
		writeErr(w, 404, "sem container ativo")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	stats, err := s.core.Driver.Stats(ctx, dep.ContainerID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	info, _ := s.core.Driver.Inspect(ctx, dep.ContainerID)
	writeJSON(w, 200, map[string]any{
		"state": info.State,
		"stats": stats,
	})
}

func (s *Server) handleTimeline(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	appEvents, err := s.core.Bus.Timeline(r.Context(), "app", app.ID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, appEvents)
}

func (s *Server) handleSetEnv(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	var req struct {
		Name   string `json:"name"`
		Value  string `json:"value"`
		Secret bool   `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if req.Name == "" {
		writeErr(w, 400, "nome obrigatório")
		return
	}
	if err := s.core.SetAppEnv(app.ID, req.Name, req.Value, req.Secret); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "saved"})
}

func (s *Server) handleDeleteEnv(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	if err := s.core.Store.DeleteEnvVar(app.ID, r.PathValue("name")); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleAddDomain(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	var req struct {
		Host  string `json:"host"`
		HTTPS bool   `json:"https"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if strings.TrimSpace(req.Host) == "" {
		writeErr(w, 400, "host obrigatório")
		return
	}
	if err := s.core.CreateDomain(app.ID, strings.ToLower(req.Host), req.HTTPS); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, map[string]string{"status": "added", "host": req.Host})
}

func (s *Server) handleListDomains(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	domains, err := s.core.Store.ListDomains(app.ID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, domains)
}

func (s *Server) handleRemoveDomain(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	host := r.PathValue("host")
	domains, err := s.core.Store.ListDomains(app.ID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	for _, d := range domains {
		if d.Host == host {
			s.core.Net.RemoveRoute(host)
			if err := s.core.Store.DeleteDomain(d.ID); err != nil {
				writeErr(w, 500, err.Error())
				return
			}
			writeJSON(w, 200, map[string]string{"status": "deleted"})
			return
		}
	}
	writeErr(w, 404, "domínio não encontrado")
}

func (s *Server) handleSetWebhook(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	var req struct {
		Secret string `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	enc, err := s.core.Secrets.EncryptString(req.Secret)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if _, err := s.core.DB.Exec(`UPDATE apps SET webhook_secret=? WHERE id=?`, enc, app.ID); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "saved"})
}

func (s *Server) handleListBackups(w http.ResponseWriter, r *http.Request) {
	backups, err := s.core.Store.ListBackups(50)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, backups)
}

func (s *Server) handleCreateBackup(w http.ResponseWriter, r *http.Request) {
	b, err := s.core.BackupCreate()
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, b)
}

func (s *Server) handleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	if err := s.core.RestoreBackup(r.PathValue("id")); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "restored"})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.core.Bus.Recent(r.Context(), 100)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, events)
}

func (s *Server) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	app, err := s.core.Store.GetApp(r.PathValue("appID"))
	if err != nil {
		writeErr(w, 404, "aplicação não encontrada")
		return
	}
	if app.SourceType != domain.SourceGit {
		writeErr(w, 400, "aplicação não é de fonte git")
		return
	}
	if app.WebhookSecret == "" {
		writeErr(w, 403, "webhook não configurado")
		return
	}
	secret, err := s.core.Secrets.DecryptString(app.WebhookSecret)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	sig := r.Header.Get("X-Hub-Signature-256")
	if err := git.VerifyGitHubSignature(body, sig, []byte(secret)); err != nil {
		writeErr(w, 401, err.Error())
		return
	}
	ev, err := git.ParsePushEvent(body)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	if ev.Branch() != app.GitBranch {
		writeJSON(w, 200, map[string]string{"status": "ignored", "reason": "branch diferente"})
		return
	}
	delivery := r.Header.Get("X-GitHub-Delivery")
	if delivery == "" {
		delivery = ev.Ref
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	ran, err := s.core.RT.RunOnce(ctx, "idem:webhook:"+app.ID+":"+delivery, 15*time.Second, func() error {
		_, derr := s.core.Deploy(app.ID, core.DeployOpts{Trigger: "webhook", Commit: ""})
		s.core.TriggerDeployPipelines(context.Background(), app.ID, "deploy")
		return derr
	})
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if !ran {
		writeJSON(w, 202, map[string]string{"status": "duplicate", "delivery": delivery})
		return
	}
	deploys, derr := s.core.Store.ListDeployments(app.ID, 1)
	if derr != nil || len(deploys) == 0 {
		writeJSON(w, 202, map[string]string{"status": "queued"})
		return
	}
	writeJSON(w, 202, deploys[0])
}

func (s *Server) handleWeb(w http.ResponseWriter, r *http.Request) {
	index := filepath.Join(s.webDir, "index.html")
	if _, err := os.Stat(index); err != nil {
		writeJSON(w, 200, map[string]string{"name": "aether", "api": "/api/v1"})
		return
	}
	if r.URL.Path == "/" {
		http.ServeFile(w, r, index)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/assets/") || strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/healthz") {
		http.FileServer(http.Dir(s.webDir)).ServeHTTP(w, r)
		return
	}
	// SPA fallback: qualquer rota client-side serve o index.html
	http.ServeFile(w, r, index)
}

func (s *Server) handleLogHistory(w http.ResponseWriter, r *http.Request) {
	app, ok := s.appForOrg(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	deploys, err := s.core.Store.ListDeployments(app.ID, 1)
	if err != nil || len(deploys) == 0 {
		writeErr(w, 404, "sem deployments")
		return
	}
	dep := &deploys[0]
	path := filepath.Join(s.core.Cfg.LogsDir, "apps", app.Name, fmt.Sprintf("%d.log", dep.Number))
	data, err := os.ReadFile(path)
	if err != nil {
		writeJSON(w, 200, map[string]any{"lines": []string{}, "has_more": false})
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	before, _ := strconv.Atoi(r.URL.Query().Get("before"))
	if before <= 0 {
		before = len(data)
	}
	end := before
	start := end - limit*4096
	if start < 0 {
		start = 0
	}
	// ajusta start para quebra de linha
	if start > 0 {
		for start < end && data[start] != '\n' {
			start++
		}
		if start < end {
			start++
		}
	}
	chunk := string(data[start:end])
	hasMore := start > 0
	writeJSON(w, 200, map[string]any{"text": chunk, "offset": start, "has_more": hasMore})
}
