package core_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aether/internal/api"
	"aether/internal/domain"
	"aether/internal/security"
)

func TestMultiTenancyProjectScoping(t *testing.T) {
	c, org, token := bootstrapTestCore(t)
	srv := httptest.NewServer(api.New(c, "").Handler())
	defer srv.Close()

	auth := func(tok string) map[string]string {
		return map[string]string{"Authorization": "Bearer " + tok, "Content-Type": "application/json"}
	}

	// cria uma segunda organização
	body, _ := json.Marshal(map[string]string{"name": "Acme Corp"})
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/organizations", bytes.NewReader(body))
	for k, v := range auth(token) {
		req.Header.Set(k, v)
	}
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 201 {
		t.Fatalf("criar org: status %d", resp.StatusCode)
	}
	var acme domain.Org
	json.NewDecoder(resp.Body).Decode(&acme)
	resp.Body.Close()

	// cria um projeto na Acme
	body, _ = json.Marshal(map[string]string{"name": "Payments"})
	req, _ = http.NewRequest("POST", srv.URL+"/api/v1/projects", bytes.NewReader(body))
	for k, v := range auth(token) {
		req.Header.Set(k, v)
	}
	req.Header.Set("X-Aether-Org", acme.ID)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 201 {
		t.Fatalf("criar projeto: status %d", resp.StatusCode)
	}
	var proj domain.Project
	json.NewDecoder(resp.Body).Decode(&proj)
	resp.Body.Close()

	// cria um segundo usuário (membro) direto no store
	var h security.PasswordHasher
	hash, _ := h.Hash("senha123")
	member := &domain.User{ID: domain.NewID(), Email: "john@acme.com", Name: "John", PasswordHash: hash, CreatedAt: time.Now().UTC()}
	if err := c.Store.CreateUser(member); err != nil {
		t.Fatal(err)
	}
	// convida como member com atribuição apenas ao projeto Payments
	body, _ = json.Marshal(map[string]any{"email": member.Email, "role": "member", "projects": []string{proj.ID}})
	req, _ = http.NewRequest("POST", srv.URL+"/api/v1/organizations/"+acme.ID+"/members", bytes.NewReader(body))
	for k, v := range auth(token) {
		req.Header.Set(k, v)
	}
	req.Header.Set("X-Aether-Org", acme.ID)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 201 {
		t.Fatalf("convidar membro: status %d", resp.StatusCode)
	}
	resp.Body.Close()

	// cria um segundo projeto que NÃO deve ser visível ao membro
	body, _ = json.Marshal(map[string]string{"name": "Infrastructure"})
	req, _ = http.NewRequest("POST", srv.URL+"/api/v1/projects", bytes.NewReader(body))
	for k, v := range auth(token) {
		req.Header.Set(k, v)
	}
	req.Header.Set("X-Aether-Org", acme.ID)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 201 {
		t.Fatalf("criar projeto 2: status %d", resp.StatusCode)
	}
	resp.Body.Close()

	// login como membro
	login, _ := json.Marshal(map[string]string{"email": member.Email, "password": "senha123"})
	req, _ = http.NewRequest("POST", srv.URL+"/api/v1/auth/login", bytes.NewReader(login))
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		t.Fatalf("login membro: status %d", resp.StatusCode)
	}
	var lr struct{ Token string }
	json.NewDecoder(resp.Body).Decode(&lr)
	resp.Body.Close()

	// o membro vê apenas o projeto atribuído
	req, _ = http.NewRequest("GET", srv.URL+"/api/v1/projects", nil)
	for k, v := range auth(lr.Token) {
		req.Header.Set(k, v)
	}
	req.Header.Set("X-Aether-Org", acme.ID)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		t.Fatalf("listar projetos membro: status %d", resp.StatusCode)
	}
	var projects []domain.Project
	json.NewDecoder(resp.Body).Decode(&projects)
	resp.Body.Close()
	if len(projects) != 1 || projects[0].Name != "Payments" {
		t.Fatalf("membro deveria ver apenas Payments, viu: %+v", projects)
	}

	// o membro NÃO pode acessar a organização Default (sem membership)
	req, _ = http.NewRequest("GET", srv.URL+"/api/v1/projects", nil)
	for k, v := range auth(lr.Token) {
		req.Header.Set(k, v)
	}
	req.Header.Set("X-Aether-Org", org.ID)
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("membro deveria levar 403 na org estranha, status %d", resp.StatusCode)
	}
}
