package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	appsdomain "aether/internal/apps/domain"
	appsInfra "aether/internal/apps/infra"
	"aether/internal/domains/domain"
	"aether/internal/domains/infra"
)

type env struct {
	ctx       context.Context
	svc       *Domains
	appsStore *appsInfra.Store
	orgID     uuid.UUID
	appID     uuid.UUID
	projectID uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	appsStore := appsInfra.NewStore(pool)
	e := &env{ctx: context.Background(), svc: &Domains{
		Store: store, Apps: appsStore,
		Provisioner: &Provisioner{TraefikDir: t.TempDir(), FreeDomainBase: "apps.example.com"},
	}, appsStore: appsStore, orgID: uuid.New()}
	t.Cleanup(func() {
		_ = store.Close()
		_ = appsStore.Close()
	})
	if _, err := pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, e.orgID, "Dom Org", "dom-org"); err != nil {
		t.Fatalf("criar org: %v", err)
	}
	project, err := appsStore.CreateProject(e.ctx, e.orgID, "DomProj", "dom-proj", "", "")
	if err != nil {
		t.Fatalf("criar project: %v", err)
	}
	e.projectID = project.ID
	app, err := appsStore.CreateApp(e.ctx, &appsdomain.App{
		OrgID: e.orgID, ProjectID: project.ID, Name: "api", SourceType: "git",
		Image: "nginx", Port: 80, PreviewDomain: "previews.example.com",
	})
	if err != nil {
		t.Fatalf("criar app: %v", err)
	}
	e.appID = app.ID
	return e
}

func TestDomainLifecycle(t *testing.T) {
	e := newEnv(t)
	d, err := e.svc.Add(e.ctx, e.appID, e.orgID, ServiceTypeApp, AddDomainInput{Host: "Api.Example.COM", HTTPS: true})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if d.Host != "api.example.com" || !d.HTTPS {
		t.Fatalf("host deveria ser minúsculo: %+v", d)
	}

	if _, err := e.svc.Add(e.ctx, e.appID, e.orgID, ServiceTypeApp, AddDomainInput{Host: "api.example.com"}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("host duplicado deveria falhar: %v", err)
	}

	list, err := e.svc.List(e.ctx, e.appID, e.orgID, ServiceTypeApp)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}

	if err := e.svc.Remove(e.ctx, e.appID, e.orgID, ServiceTypeApp, "api.example.com"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	list, _ = e.svc.List(e.ctx, e.appID, e.orgID, ServiceTypeApp)
	if len(list) != 0 {
		t.Fatalf("domínio deveria ter sido removido")
	}
}

func TestDomainIsolation(t *testing.T) {
	e := newEnv(t)
	if _, err := e.svc.Add(e.ctx, e.appID, uuid.New(), ServiceTypeApp, AddDomainInput{Host: "x.example.com"}); !errors.Is(err, appsdomain.ErrNotFound) {
		t.Fatalf("org errada deveria falhar: %v", err)
	}
}

func TestDomainValidation(t *testing.T) {
	e := newEnv(t)
	if _, err := e.svc.Add(e.ctx, e.appID, e.orgID, ServiceTypeApp, AddDomainInput{Host: ""}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("host vazio deveria falhar: %v", err)
	}
	if err := e.svc.Provisioner.ValidateHost("localhost"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("localhost deveria ser rejeitado")
	}
	if err := e.svc.Provisioner.ValidateHost("127.0.0.1"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("IP deveria ser rejeitado")
	}
	if err := e.svc.Provisioner.ValidateHost("app.example.com"); err != nil {
		t.Fatalf("host válido deveria passar: %v", err)
	}
	host := e.svc.Provisioner.GenerateFreeDomain("My App", e.appID)
	if !strings.HasSuffix(host, ".apps.example.com") || strings.Contains(host, " ") {
		t.Fatalf("free domain inesperado: %s", host)
	}
}

func TestProvisionWorkerHTTP(t *testing.T) {
	e := newEnv(t)
	d, err := e.svc.Add(e.ctx, e.appID, e.orgID, ServiceTypeApp, AddDomainInput{Host: "http.example.com"})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	w := &ProvisionWorker{Store: e.svc.Store, Provisioner: e.svc.Provisioner}
	w.process(e.ctx)
	got, err := e.svc.Store.GetDomainByID(e.ctx, d.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != string(domain.DomainActive) || got.CertStatus != "active" {
		t.Fatalf("http domain deveria ficar ACTIVE: %+v", got)
	}
	content, err := os.ReadFile(filepath.Join(e.svc.Provisioner.TraefikDir, "dynamic", "domain-"+d.ID.String()+".yml"))
	if err != nil {
		t.Fatalf("config não escrita: %v", err)
	}
	if !strings.Contains(string(content), "app-"+e.appID.String()[:8]) {
		t.Fatalf("config sem alias do app")
	}
}

func TestProvisionWorkerRetry(t *testing.T) {
	e := newEnv(t)
	d, err := e.svc.Add(e.ctx, e.appID, e.orgID, ServiceTypeApp, AddDomainInput{Host: "https.example.com", HTTPS: true})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	w := &ProvisionWorker{Store: e.svc.Store, Provisioner: e.svc.Provisioner}
	w.process(e.ctx)
	got, _ := e.svc.Store.GetDomainByID(e.ctx, d.ID)
	if got.Status != string(domain.DomainActive) || got.CertStatus != "pending" || got.RetryCount == 0 {
		t.Fatalf("https sem cert deveria ficar ACTIVE/pending com retry: %+v", got)
	}
	next := retryIn(got.RetryCount)
	if next == nil || !next.After(time.Now()) {
		t.Fatalf("backoff inválido")
	}
}

func TestDomainProvisionIdempotent(t *testing.T) {
	e := newEnv(t)
	w := &ProvisionWorker{Store: e.svc.Store, Provisioner: e.svc.Provisioner}
	_, _ = e.svc.Add(e.ctx, e.appID, e.orgID, ServiceTypeApp, AddDomainInput{Host: "idem.example.com"})
	w.process(e.ctx)
	w.process(e.ctx)
	list, _ := e.svc.Store.ListDomains(e.ctx, e.appID)
	if len(list) != 1 {
		t.Fatalf("deveria haver apenas 1 domínio (idempotente): %d", len(list))
	}
}

func TestPreviewLifecycle(t *testing.T) {
	e := newEnv(t)
	p, err := e.svc.CreatePreview(e.ctx, e.appID, e.orgID, "feature/foo")
	if err != nil {
		t.Fatalf("create preview: %v", err)
	}
	if p.Status != "active" || p.Domain != "api-feature-foo.previews.example.com" {
		t.Fatalf("preview inesperado: %+v", p)
	}

	if _, err := e.svc.CreatePreview(e.ctx, e.appID, e.orgID, "feature/foo"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("preview duplicado deveria falhar: %v", err)
	}

	previews, err := e.svc.ListPreviews(e.ctx, e.appID, e.orgID)
	if err != nil || len(previews) != 1 {
		t.Fatalf("list previews: %v %d", err, len(previews))
	}

	if err := e.svc.DeletePreview(e.ctx, p.ID, e.orgID); err != nil {
		t.Fatalf("delete preview: %v", err)
	}
}

func TestPreviewDeleteIsolation(t *testing.T) {
	e := newEnv(t)
	p, _ := e.svc.CreatePreview(e.ctx, e.appID, e.orgID, "main")
	if err := e.svc.DeletePreview(e.ctx, p.ID, uuid.New()); !errors.Is(err, appsdomain.ErrNotFound) {
		t.Fatalf("delete de outra org deveria falhar: %v", err)
	}
}

func TestCertificatesAggregate(t *testing.T) {
	e := newEnv(t)
	app := &appsdomain.App{OrgID: e.orgID, ProjectID: e.projectID, Name: "api2", SourceType: "image", Image: "nginx", Port: 80}
	created, _ := e.appsStore.CreateApp(e.ctx, app)
	if _, err := e.svc.Add(e.ctx, created.ID, e.orgID, ServiceTypeApp, AddDomainInput{Host: "secure.example.com", HTTPS: true}); err != nil {
		t.Fatalf("add domain: %v", err)
	}
	certs, err := e.svc.Certificates(e.ctx, e.orgID)
	if err != nil || len(certs) != 1 {
		t.Fatalf("certs: %v %d", err, len(certs))
	}
	if certs[0].Host != "secure.example.com" || !certs[0].HTTPS || certs[0].AppName != "api2" {
		t.Fatalf("cert inesperado: %+v", certs[0])
	}
}
