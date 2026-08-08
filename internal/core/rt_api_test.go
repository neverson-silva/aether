package core_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aether/internal/api"
	"aether/internal/domain"
)

func TestPresenceAPI(t *testing.T) {
	c, org, token := bootstrapTestCore(t)
	srv := httptest.NewServer(api.New(c, "").Handler())
	defer srv.Close()
	base := srv.URL + "/api/v1/presence"
	do := func(method, path, body string) *http.Response {
		req, _ := http.NewRequest(method, base+path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}
	scope := "deployment:abc:logs"
	resp := do("POST", "/join", `{"scope":"`+scope+`"}`)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("join: %d", resp.StatusCode)
	}
	resp = do("GET", "/count?scope="+scope, "")
	var got map[string]any
	json.NewDecoder(resp.Body).Decode(&got)
	resp.Body.Close()
	if got["count"].(float64) != 1 {
		t.Fatalf("count esperado 1: %v", got)
	}
	resp = do("POST", "/heartbeat", `{"scope":"`+scope+`"}`)
	resp.Body.Close()
	resp = do("POST", "/leave", `{"scope":"`+scope+`"}`)
	resp.Body.Close()
	resp = do("GET", "/count?scope="+scope, "")
	json.NewDecoder(resp.Body).Decode(&got)
	resp.Body.Close()
	if got["count"].(float64) != 0 {
		t.Fatalf("count após leave esperado 0: %v", got)
	}
	_ = org
}

func TestAppStateStream(t *testing.T) {
	c, org, token := bootstrapTestCore(t)
	proj, _ := c.CreateProject(org.ID, "stproj")
	app := &domain.App{ID: domain.NewID(), OrgID: org.ID, ProjectID: proj.ID, Name: "stapp", SourceType: domain.SourceImage, Image: "nginx:alpine", Port: 80}
	if err := c.CreateApp(org.ID, app); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(api.New(c, "").Handler())
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/apps/states/stream", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("stream: %d", resp.StatusCode)
	}
	c.PublishAppState(app.ID, "running")
	c.PublishAppState(app.ID, "exited")
	sc := bufio.NewScanner(resp.Body)
	deadline := time.Now().Add(8 * time.Second)
	got := map[string]string{}
	for time.Now().Before(deadline) {
		if !sc.Scan() {
			break
		}
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var s map[string]string
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &s); err != nil {
			continue
		}
		got[s["app_id"]] = s["state"]
		if got[app.ID] == "exited" {
			break
		}
	}
	if got[app.ID] != "exited" {
		t.Fatalf("estado não propagado via SSE: %v", got)
	}
}
