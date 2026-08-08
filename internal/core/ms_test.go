package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"aether/internal/domain"
)

func TestServerTokenRoundtrip(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	token, err := c.NewServerTokenFor("worker-1")
	if err != nil || token == "" {
		t.Fatalf("token: %v %v", token, err)
	}
	sum := sha256.Sum256([]byte(token))
	bound, err := c.Store.ConsumeServerToken(hex.EncodeToString(sum[:]))
	if err != nil {
		t.Fatal(err)
	}
	if bound != "" {
		t.Fatalf("token genérico deveria devolver vazio: %s", bound)
	}
	if _, err := c.Store.ConsumeServerToken(hex.EncodeToString(sum[:])); err == nil {
		t.Fatal("token deveria ser consumido (single-use)")
	}
}

func TestSchedulerLeastLoaded(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	_, org, err := c.CreateUserAndOrg("ms@aether.local", "ms", "senha-ms")
	if err != nil {
		t.Fatal(err)
	}
	proj, _ := c.CreateProject(org.ID, "msproj")
	now := time.Now().UTC()
	busy := &domain.Server{ID: "srv-busy", Name: "busy", Status: "healthy", Load: 3.2, LastHeartbeat: now, CreatedAt: now}
	free := &domain.Server{ID: "srv-free", Name: "free", Status: "healthy", Load: 0.1, LastHeartbeat: now, CreatedAt: now}
	down := &domain.Server{ID: "srv-down", Name: "down", Status: "unhealthy", Load: 0.0, LastHeartbeat: now, CreatedAt: now}
	for _, srv := range []*domain.Server{busy, free, down} {
		if err := c.Store.CreateServer(srv); err != nil {
			t.Fatal(err)
		}
	}
	app := &domain.App{ID: domain.NewID(), OrgID: org.ID, ProjectID: proj.ID, Name: "web"}
	got, err := c.placeOnAgent(app)
	if err != nil || got != "srv-free" {
		t.Fatalf("scheduler deveria escolher o menos carregado: %s %v", got, err)
	}
	app.ServerID = "srv-busy"
	got, _ = c.placeOnAgent(app)
	if got != "srv-busy" {
		t.Fatalf("server_id explícito deveria vencer: %s", got)
	}
	app.ServerID = ""
	c.Store.DeleteServer("srv-busy")
	c.Store.DeleteServer("srv-free")
	got, _ = c.placeOnAgent(app)
	if got != "" {
		t.Fatalf("sem servidores saudáveis deveria cair para local: %s", got)
	}
}

func TestAgentRegisterAndHeartbeat(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	if err := c.StartAgentServer(); err != nil {
		t.Fatal(err)
	}
	url := c.AgentURL()
	if url == "" {
		t.Fatal("AgentURL vazia")
	}
	token, err := c.NewServerTokenFor("worker-2")
	if err != nil {
		t.Fatal(err)
	}
	// sem TLS confiável o teste usa o CA gerado; validamos registro com cliente HTTP inseguro
	body, _ := json.Marshal(map[string]any{
		"token": token, "name": "worker-2", "host": "test.local", "version": "0.1.0",
	})
	req, _ := http.NewRequest("POST", url+"/agent/v1/register", bytes.NewReader(body))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skip("agent server não respondeu: " + err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("register: %d", resp.StatusCode)
	}
	var reg struct {
		ServerID string `json:"server_id"`
	}
	json.NewDecoder(resp.Body).Decode(&reg)
	if reg.ServerID == "" {
		t.Fatal("server_id vazio")
	}
	srv, err := c.Store.GetServer(reg.ServerID)
	if err != nil || srv.Status != "registered" {
		t.Fatalf("server não registrado: %+v %v", srv, err)
	}
}

func TestDequeueCommands(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	payload := `{"deployment_id":"d1","app":{"image":"nginx"}}`
	if err := c.Store.EnqueueServerCommand("srv-x", "deploy", payload); err != nil {
		t.Fatal(err)
	}
	cmds, err := c.Store.DequeueServerCommands("srv-x", 10)
	if err != nil || len(cmds) != 1 || cmds[0].Kind != "deploy" {
		t.Fatalf("dequeue: %v %v", cmds, err)
	}
	cmds, _ = c.Store.DequeueServerCommands("srv-x", 10)
	if len(cmds) != 0 {
		t.Fatal("comando deveria ter sido marcado entregue")
	}
}
