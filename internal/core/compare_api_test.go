package core_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aether/internal/api"
	"aether/internal/core"
	"aether/internal/domain"
)

func TestCompareDeploymentsAPI(t *testing.T) {
	c, org, token := bootstrapTestCore(t)
	proj, _ := c.CreateProject(org.ID, "cmp")
	envs, _ := c.ListEnvironments(proj.ID)
	app := &domain.App{ID: domain.NewID(), OrgID: org.ID, ProjectID: proj.ID, EnvironmentID: envs[0].ID, Name: "cmpapp", SourceType: domain.SourceImage, Image: "nginx:alpine", Port: 18080}
	if err := c.CreateApp(org.ID, app); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Deploy(app.ID, core.DeployOpts{Trigger: "api", TriggeredBy: "t", ImageOverride: "nginx:alpine"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Deploy(app.ID, core.DeployOpts{Trigger: "api", TriggeredBy: "t", ImageOverride: "nginx:1.25"}); err != nil {
		t.Fatal(err)
	}
	// os deployments são processados de forma assíncrona; aguarda os dois
	// terem image_ref persistido para um compare determinístico.
	var deploys []domain.Deployment
	for i := 0; i < 50; i++ {
		deploys, _ = c.Store.ListDeployments(app.ID, 2)
		if len(deploys) == 2 && deploys[0].ImageRef != "" && deploys[1].ImageRef != "" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(deploys) < 2 {
		t.Fatal("precisamos de 2 deploys")
	}
	srv := httptest.NewServer(api.New(c, "").Handler())
	defer srv.Close()
	url := srv.URL + "/api/v1/apps/" + app.ID + "/deployments/compare?a=" + deploys[1].ID + "&b=" + deploys[0].ID
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var diff map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&diff); err != nil {
		t.Fatal(err)
	}
	img := diff["image"].(map[string]any)
	if img["from"] != "nginx:alpine" || img["to"] != "nginx:1.25" {
		t.Fatalf("image diff: %v", img)
	}
	if _, ok := diff["env_added"]; !ok {
		t.Fatal("env_added ausente")
	}
	t.Logf("compare ok: %+v", diff)
}
