package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"aether/internal/domain"
	"aether/internal/security"
)

func (s *Server) handleListMyOrgs(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	memberships, err := s.core.Store.ListOrgsForUser(claims.Subject)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	orgs := make([]map[string]any, 0, len(memberships))
	for _, m := range memberships {
		org, err := s.core.Store.GetOrg(m.OrgID)
		if err != nil {
			continue
		}
		orgs = append(orgs, map[string]any{
			"id":             org.ID,
			"slug":           org.Slug,
			"name":           org.Name,
			"description":    org.Description,
			"avatar":         org.Avatar,
			"color":          org.Color,
			"owner_user_id":  org.OwnerUserID,
			"role":           m.Role,
			"projects_count": s.orgProjectCount(org.ID),
		})
	}
	writeJSON(w, 200, orgs)
}

func slugify(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	out := strings.ToLower(re.ReplaceAllString(strings.TrimSpace(s), "-"))
	return strings.Trim(out, "-")
}

func (s *Server) orgProjectCount(orgID string) int {
	projects, err := s.core.Store.ListProjects(orgID)
	if err != nil {
		return 0
	}
	return len(projects)
}

func (s *Server) handleGetOrg(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	org, ok := s.orgForMember(w, r, claims, r.PathValue("id"))
	if !ok {
		return
	}
	writeJSON(w, 200, org)
}

// orgForMember valida que o usuário é membro da organização (ou Global Admin).
func (s *Server) orgForMember(w http.ResponseWriter, r *http.Request, claims *security.Claims, id string) (*domain.Org, bool) {
	org, err := s.core.Store.GetOrg(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "organização não encontrada")
		return nil, false
	}
	if claims.GlobalRole != domain.GlobalAdmin {
		if _, err := s.core.Store.GetMember(id, claims.Subject); err != nil {
			writeErr(w, http.StatusForbidden, "fora do escopo da organização")
			return nil, false
		}
	}
	return org, true
}

func (s *Server) handleCreateOrg(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	var req struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
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
	now := time.Now().UTC()
	org := &domain.Org{
		ID:          domain.NewID(),
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		OwnerUserID: claims.Subject,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if req.Slug != "" {
		org.Slug = slugify(req.Slug)
	} else {
		org.Slug = slugify(req.Name)
	}
	// garante slug único
	if _, err := s.core.Store.GetOrgBySlug(org.Slug); err == nil {
		org.Slug = org.Slug + "-" + strconv.Itoa(int(time.Now().UnixNano()%100000))
	}
	if err := s.core.Store.CreateOrg(org); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if err := s.core.Store.CreateMember(&domain.Member{OrgID: org.ID, UserID: claims.Subject, Role: domain.RoleOwner}); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	s.core.Store.CreateAudit(&domain.AuditLog{ID: domain.NewID(), OrgID: org.ID, UserID: claims.Subject, Action: "org.created", ResourceType: "organization", ResourceID: org.ID, CreatedAt: now})
	writeJSON(w, 201, org)
}

func (s *Server) handleUpdateOrg(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	org, ok := s.orgForMember(w, r, claims, r.PathValue("id"))
	if !ok {
		return
	}
	role := domain.Role(claims.Role)
	if claims.GlobalRole != domain.GlobalAdmin && !role.AtLeast(domain.RoleAdmin) {
		writeErr(w, http.StatusForbidden, "somente owner/admin podem editar a organização")
		return
	}
	var req struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
		Avatar      string `json:"avatar"`
		Color       string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if req.Name != "" {
		org.Name = req.Name
	}
	if req.Slug != "" {
		org.Slug = slugify(req.Slug)
	}
	if req.Description != "" {
		org.Description = req.Description
	}
	if req.Avatar != "" {
		org.Avatar = req.Avatar
	}
	if req.Color != "" {
		org.Color = req.Color
	}
	org.UpdatedAt = time.Now().UTC()
	if err := s.core.Store.UpdateOrg(org); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	s.core.Store.CreateAudit(&domain.AuditLog{ID: domain.NewID(), OrgID: org.ID, UserID: claims.Subject, Action: "org.updated", ResourceType: "organization", ResourceID: org.ID, CreatedAt: time.Now().UTC()})
	writeJSON(w, 200, org)
}

func (s *Server) handleDeleteOrg(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	org, ok := s.orgForMember(w, r, claims, r.PathValue("id"))
	if !ok {
		return
	}
	if claims.GlobalRole != domain.GlobalAdmin && org.OwnerUserID != claims.Subject {
		writeErr(w, http.StatusForbidden, "somente o owner pode deletar a organização")
		return
	}
	if err := s.core.Store.DeleteOrg(org.ID); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	s.core.Store.CreateAudit(&domain.AuditLog{ID: domain.NewID(), OrgID: org.ID, UserID: claims.Subject, Action: "org.deleted", ResourceType: "organization", ResourceID: org.ID, CreatedAt: time.Now().UTC()})
	writeJSON(w, 200, map[string]bool{"deleted": true})
}

// ---- Members ----

type orgMemberView struct {
	OrgID     string   `json:"org_id"`
	UserID    string   `json:"user_id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Role      string   `json:"role"`
	Projects  []string `json:"projects"`
	CreatedAt string   `json:"created_at"`
}

func (s *Server) handleListOrgMembers(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	orgID := r.PathValue("id")
	if _, ok := s.orgForMember(w, r, claims, orgID); !ok {
		return
	}
	members, err := s.core.Store.ListMembers(orgID)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	out := make([]orgMemberView, 0, len(members))
	for _, m := range members {
		user, err := s.core.Store.GetUser(m.UserID)
		if err != nil {
			continue
		}
		projs, _ := s.core.Store.ListProjectAssignments(orgID, m.UserID)
		out = append(out, orgMemberView{
			OrgID: m.OrgID, UserID: m.UserID, Email: user.Email, Name: user.Name,
			Role: string(m.Role), Projects: projs,
		})
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleAddOrgMember(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	orgID := r.PathValue("id")
	org, ok := s.orgForMember(w, r, claims, orgID)
	if !ok {
		return
	}
	role := domain.Role(claims.Role)
	if claims.GlobalRole != domain.GlobalAdmin && !role.AtLeast(domain.RoleAdmin) {
		writeErr(w, http.StatusForbidden, "somente owner/admin podem convidar membros")
		return
	}
	var req struct {
		Email    string   `json:"email"`
		Role     string   `json:"role"`
		Projects []string `json:"projects"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	if req.Email == "" {
		writeErr(w, 400, "email obrigatório")
		return
	}
	newRole := domain.Role(req.Role).Normalize()
	if !newRole.IsOrgRole() || newRole == domain.RoleOwner {
		writeErr(w, 400, "role inválido (use admin, member ou viewer)")
		return
	}
	user, err := s.core.Store.GetUserByEmail(req.Email)
	if err != nil {
		writeErr(w, 404, "usuário não encontrado")
		return
	}
	if err := s.core.Store.CreateMember(&domain.Member{OrgID: orgID, UserID: user.ID, Role: newRole}); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	for _, pid := range req.Projects {
		if _, err := s.core.Store.GetProject(pid); err == nil {
			s.core.Store.AddProjectAssignment(&domain.ProjectAssignment{OrgID: orgID, UserID: user.ID, ProjectID: pid, CreatedAt: time.Now().UTC()})
		}
	}
	s.core.Store.CreateAudit(&domain.AuditLog{ID: domain.NewID(), OrgID: orgID, UserID: claims.Subject, Action: "member.invited", ResourceType: "member", ResourceID: user.ID, Details: req.Email, CreatedAt: time.Now().UTC()})
	writeJSON(w, 201, map[string]any{"org_id": org.ID, "user_id": user.ID, "role": newRole, "projects": req.Projects})
}

func (s *Server) handleUpdateOrgMember(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	orgID := r.PathValue("id")
	userID := r.PathValue("userId")
	if _, ok := s.orgForMember(w, r, claims, orgID); !ok {
		return
	}
	role := domain.Role(claims.Role)
	if claims.GlobalRole != domain.GlobalAdmin && !role.AtLeast(domain.RoleAdmin) {
		writeErr(w, http.StatusForbidden, "somente owner/admin podem alterar membros")
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "corpo inválido")
		return
	}
	newRole := domain.Role(req.Role).Normalize()
	if !newRole.IsOrgRole() {
		writeErr(w, 400, "role inválido")
		return
	}
	org, _ := s.core.Store.GetOrg(orgID)
	if userID == org.OwnerUserID && claims.GlobalRole != domain.GlobalAdmin {
		writeErr(w, http.StatusForbidden, "o owner não pode ser rebaixado")
		return
	}
	if err := s.core.Store.SetMemberRole(orgID, userID, newRole); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	s.core.Store.CreateAudit(&domain.AuditLog{ID: domain.NewID(), OrgID: orgID, UserID: claims.Subject, Action: "member.role_changed", ResourceType: "member", ResourceID: userID, Details: string(newRole), CreatedAt: time.Now().UTC()})
	writeJSON(w, 200, map[string]string{"role": string(newRole)})
}

func (s *Server) handleRemoveOrgMember(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	orgID := r.PathValue("id")
	userID := r.PathValue("userId")
	org, ok := s.orgForMember(w, r, claims, orgID)
	if !ok {
		return
	}
	role := domain.Role(claims.Role)
	if claims.GlobalRole != domain.GlobalAdmin && !role.AtLeast(domain.RoleAdmin) {
		writeErr(w, http.StatusForbidden, "somente owner/admin podem remover membros")
		return
	}
	if userID == org.OwnerUserID && claims.GlobalRole != domain.GlobalAdmin {
		writeErr(w, http.StatusForbidden, "o owner não pode ser removido")
		return
	}
	s.core.Store.RemoveProjectAssignmentsForUser(orgID, userID)
	if err := s.core.Store.DeleteMember(orgID, userID); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	s.core.Store.CreateAudit(&domain.AuditLog{ID: domain.NewID(), OrgID: orgID, UserID: claims.Subject, Action: "member.removed", ResourceType: "member", ResourceID: userID, CreatedAt: time.Now().UTC()})
	writeJSON(w, 200, map[string]bool{"removed": true})
}

// ---- Project assignments ----

func (s *Server) handleSetProjectAssignment(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	orgID := r.PathValue("id")
	userID := r.PathValue("userId")
	projectID := r.PathValue("projectId")
	if _, ok := s.orgForMember(w, r, claims, orgID); !ok {
		return
	}
	role := domain.Role(claims.Role)
	if claims.GlobalRole != domain.GlobalAdmin && !role.AtLeast(domain.RoleAdmin) {
		writeErr(w, http.StatusForbidden, "somente owner/admin podem atribuir projetos")
		return
	}
	project, err := s.core.Store.GetProject(projectID)
	if err != nil || project.OrgID != orgID {
		writeErr(w, 404, "projeto não encontrado")
		return
	}
	if err := s.core.Store.AddProjectAssignment(&domain.ProjectAssignment{OrgID: orgID, UserID: userID, ProjectID: projectID, CreatedAt: time.Now().UTC()}); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"assigned": true})
}

func (s *Server) handleRemoveProjectAssignment(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	orgID := r.PathValue("id")
	userID := r.PathValue("userId")
	projectID := r.PathValue("projectId")
	if _, ok := s.orgForMember(w, r, claims, orgID); !ok {
		return
	}
	role := domain.Role(claims.Role)
	if claims.GlobalRole != domain.GlobalAdmin && !role.AtLeast(domain.RoleAdmin) {
		writeErr(w, http.StatusForbidden, "somente owner/admin podem alterar atribuições")
		return
	}
	if err := s.core.Store.RemoveProjectAssignment(orgID, userID, projectID); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"removed": true})
}

// ---- Audit log ----

func (s *Server) handleOrgAudit(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	orgID := r.PathValue("id")
	if _, ok := s.orgForMember(w, r, claims, orgID); !ok {
		return
	}
	role := domain.Role(claims.Role)
	if claims.GlobalRole != domain.GlobalAdmin && !role.AtLeast(domain.RoleAdmin) {
		writeErr(w, http.StatusForbidden, "audit log restrito a owner/admin")
		return
	}
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	logs, err := s.core.Store.ListAudit(orgID, limit)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if logs == nil {
		logs = []domain.AuditLog{}
	}
	writeJSON(w, 200, logs)
}
