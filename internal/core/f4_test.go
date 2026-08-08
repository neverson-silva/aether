package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"aether/internal/domain"
	"aether/internal/runtime"
)

func TestAutopilotPolicy(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	_, org, err := c.CreateUserAndOrg("ap@aether.local", "ap", "senha-ap")
	if err != nil {
		t.Fatal(err)
	}
	proj, _ := c.CreateProject(org.ID, "aproj")
	a := &domain.App{ID: domain.NewID(), OrgID: org.ID, ProjectID: proj.ID, Name: "apapp"}
	c.CreateApp(org.ID, a)
	p, err := c.Store.GetPolicy(a.ID)
	if err != nil || p.Enabled {
		t.Fatalf("default policy: %+v %v", p, err)
	}
	p.Enabled = true
	p.CPUMax = 2
	p.MemMaxMB = 512
	p.ScaleUpPct = 60
	p.ScaleDownPct = 10
	if err := c.SavePolicy(&p); err != nil {
		t.Fatal(err)
	}
	got, _ := c.Store.GetPolicy(a.ID)
	if !got.Enabled || got.MemMaxMB != 512 {
		t.Fatalf("roundtrip: %+v", got)
	}
	events, _ := c.PolicyEvents(a.ID)
	if len(events) != 0 {
		t.Fatal("sem eventos deveria estar vazio")
	}
}

func TestGitOpsDriftAndApply(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	_, org, err := c.CreateUserAndOrg("go@aether.local", "go", "senha-go")
	if err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	if err := gitInit(repo); err != nil {
		t.Fatal(err)
	}
	yml := `version: 1
projects:
  - name: demo
    apps:
      - name: web
        source_type: image
        image: nginx:alpine
        port: 80
`
	if err := os.WriteFile(filepath.Join(repo, "aether.yml"), []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitCommitAll(repo, "init"); err != nil {
		t.Fatal(err)
	}
	g, err := c.CreateGitOps(org.ID, "prod", repo, "main", "aether.yml", "auto")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.SyncGitOps(g); err != nil {
		t.Fatal(err)
	}
	if g.LastStatus != "applied" {
		t.Fatalf("status: %s", g.LastStatus)
	}
	if g.DriftAdded != 0 {
		t.Fatalf("drift: %+v", g)
	}
	target, err := c.Store.GetOrg(g.TargetOrgID)
	if err != nil || target.Name != "gitops-prod" {
		t.Fatalf("org alvo: %+v %v", target, err)
	}
	projects, _ := c.Store.ListProjects(target.ID)
	if len(projects) != 1 {
		t.Fatalf("projeto não importado: %d", len(projects))
	}
}

func TestMirrorRunAgainstMock(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/_catalog" {
			w.Write([]byte(`{"repositories":["nginx"]}`))
			return
		}
		if r.URL.Path == "/v2/nginx/tags/list" {
			w.Write([]byte(`{"tags":["alpine","1.25"]}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer mock.Close()
	m, err := c.CreateMirror("test", mock.URL, mock.URL+"/dest", true, "", "")
	if err != nil {
		t.Fatal(err)
	}
	// sem skopeo o Run falha com erro claro, mas catalog/tags devem funcionar
	repos, err := c.registryCatalog(context.Background(), mock.URL)
	if err != nil || len(repos) != 1 || repos[0] != "nginx" {
		t.Fatalf("catalog: %v %v", repos, err)
	}
	tags := c.registryTags(context.Background(), mock.URL, "nginx")
	if len(tags) != 2 {
		t.Fatalf("tags: %v", tags)
	}
	_ = m
}

func TestNetQPercentiles(t *testing.T) {
	lats := []float64{10, 20, 30, 40, 50}
	if percentile(lats, 50) != 30 || percentile(lats, 95) != 50 {
		t.Fatalf("percentis errados: %v %v", percentile(lats, 50), percentile(lats, 95))
	}
	if percentile(nil, 50) != 0 {
		t.Fatal("vazio deveria ser 0")
	}
}

func TestK8sDriverRun(t *testing.T) {
	var created []map[string]any
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			var body map[string]any
			if err := jsonDecode(r, &body); err == nil {
				created = append(created, body)
			}
			w.WriteHeader(201)
			w.Write([]byte(`{}`))
		case "GET":
			w.WriteHeader(200)
			w.Write([]byte(`{"status":{"conditions":[{"type":"Available","status":"True"}]}}`))
		case "DELETE":
			w.WriteHeader(200)
		case "PATCH":
			w.WriteHeader(200)
		}
	}))
	defer mock.Close()
	d := runtime.NewK8sDriver(runtime.K8sConfig{API: mock.URL, Token: "t", Namespace: "default"})
	id, err := d.Run(context.Background(), runtime.RunSpec{
		Name:   "my_app-1",
		Image:  "nginx:alpine",
		Env:    []string{"A=1"},
		MemMB:  256,
		CPUs:   "0.5",
		Labels: map[string]string{"aether.app": "abc"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "my-app-1" {
		t.Fatalf("nome sanitizado: %s", id)
	}
	if len(created) != 2 {
		t.Fatalf("deveria criar deployment+service: %d", len(created))
	}
	if created[0]["kind"] != "Deployment" || created[1]["kind"] != "Service" {
		t.Fatalf("kinds: %v %v", created[0]["kind"], created[1]["kind"])
	}
	dep, _ := created[0]["spec"].(map[string]any)
	tpl, _ := dep["template"].(map[string]any)
	sp, _ := tpl["spec"].(map[string]any)
	containersRaw, _ := sp["containers"].([]any)
	if len(containersRaw) != 1 {
		t.Fatalf("container: %+v", containersRaw)
	}
	container := containersRaw[0].(map[string]any)
	if container["image"] != "nginx:alpine" {
		t.Fatalf("container: %+v", container)
	}
	res, _ := container["resources"].(map[string]any)
	limits, _ := res["limits"].(map[string]any)
	if limits["memory"] != "256Mi" || limits["cpu"] != "0.5" {
		t.Fatalf("resources: %+v", limits)
	}
}

func TestMigrateDiscover(t *testing.T) {
	root := t.TempDir()
	for _, svc := range []string{"app-a", "app-b"} {
		dir := filepath.Join(root, svc)
		os.MkdirAll(dir, 0o750)
		os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("services:\n  web:\n    image: nginx\n"), 0o644)
		os.WriteFile(filepath.Join(dir, ".env"), []byte("PORT=3000\nAPI_KEY=segredo\nMODE=prod\n"), 0o644)
	}
	res, err := DiscoverCoolify(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Services) != 2 {
		t.Fatalf("serviços: %d", len(res.Services))
	}
	a := res.Services[0]
	if a.Name != "app-a" || a.Env["PORT"] != "3000" {
		t.Fatalf("env: %+v", a)
	}
	if len(a.Secrets) != 1 || a.Secrets[0] != "API_KEY" {
		t.Fatalf("secrets: %v", a.Secrets)
	}
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	_, org, err := c.CreateUserAndOrg("mi@aether.local", "mi", "senha-mi")
	if err != nil {
		t.Fatal(err)
	}
	n, err := c.ImportDiscovered(org.ID, res)
	if err != nil || n != 2 {
		t.Fatalf("import: %d %v", n, err)
	}
	compose, err := c.ListCompose(org.ID)
	if err != nil || len(compose) != 2 {
		t.Fatalf("compose: %d %v", len(compose), err)
	}
}

func jsonDecode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func gitInit(dir string) error {
	cmd := exec.Command("git", "init", "-q", "-b", "main", dir)
	return cmd.Run()
}

func gitCommitAll(dir, msg string) error {
	if err := exec.Command("git", "-C", dir, "add", "-A").Run(); err != nil {
		return err
	}
	exec.Command("git", "-C", dir, "config", "user.email", "test@aether.local").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "test").Run()
	return exec.Command("git", "-C", dir, "commit", "-q", "-m", msg).Run()
}
