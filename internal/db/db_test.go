package db

import (
	"testing"
	"time"

	"aether/internal/domain"
)

func testStore(t *testing.T) *Store {
	sqldb := OpenTest(t)
	return NewStore(sqldb)
}

func TestMigrations(t *testing.T) {
	sqldb := OpenTest(t)
	v, err := Version(sqldb)
	if err != nil {
		t.Fatal(err)
	}
	if v != 22 {
		t.Fatalf("versão esperada 21, obtida %d", v)
	}
}

func TestUserOrgMemberFlow(t *testing.T) {
	s := testStore(t)
	user := &domain.User{ID: domain.NewID(), Email: "a@b.c", Name: "A", PasswordHash: "h", CreatedAt: time.Now()}
	if err := s.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetUserByEmail("a@b.c")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != user.ID {
		t.Fatal("id divergiu")
	}
	if _, err := s.GetUserByEmail("n@o.p"); err != ErrNotFound {
		t.Fatal("deveria retornar ErrNotFound")
	}
	org := &domain.Org{ID: domain.NewID(), Name: "O", OwnerUserID: user.ID, CreatedAt: time.Now()}
	if err := s.CreateOrg(org); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateMember(&domain.Member{OrgID: org.ID, UserID: user.ID, Role: domain.RoleOwner}); err != nil {
		t.Fatal(err)
	}
	m, err := s.GetMember(org.ID, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if m.Role != domain.RoleOwner {
		t.Fatal("papel divergiu")
	}
	if err := s.SetMemberRole(org.ID, user.ID, domain.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	m, _ = s.GetMember(org.ID, user.ID)
	if m.Role != domain.RoleAdmin {
		t.Fatal("papel não atualizou")
	}
}

func TestAppWithVolumesAndDeployments(t *testing.T) {
	s := testStore(t)
	orgID := domain.NewID()
	project := &domain.Project{ID: domain.NewID(), OrgID: orgID, Name: "p", CreatedAt: time.Now()}
	if err := s.CreateProject(project); err != nil {
		t.Fatal(err)
	}
	app := &domain.App{
		ID:         domain.NewID(),
		OrgID:      orgID,
		ProjectID:  project.ID,
		Name:       "web",
		SourceType: domain.SourceImage,
		Image:      "nginx:alpine",
		Port:       80,
		Volumes:    []domain.Volume{{Name: "data", MountPath: "/data"}},
	}
	if err := s.CreateApp(app); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetApp(app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Volumes) != 1 || got.Volumes[0].MountPath != "/data" {
		t.Fatalf("volumes divergiram: %+v", got.Volumes)
	}
	if err := s.SetEnvVar(app.ID, "FOO", "bar", false); err != nil {
		t.Fatal(err)
	}
	if err := s.SetEnvVar(app.ID, "TOKEN", "cifrado", true); err != nil {
		t.Fatal(err)
	}
	envs, err := s.ListEnvVars(app.ID)
	if err != nil || len(envs) != 2 {
		t.Fatalf("env divergiu: %v %v", envs, err)
	}
	n, err := s.NextDeploymentNumber(app.ID)
	if err != nil || n != 1 {
		t.Fatalf("numero: %v %v", n, err)
	}
	dep := &domain.Deployment{
		ID: domain.NewID(), AppID: app.ID, Number: 1,
		Status: domain.DeploymentReady, CreatedAt: time.Now(), StartedAt: time.Now(),
	}
	if err := s.CreateDeployment(dep); err != nil {
		t.Fatal(err)
	}
	n, _ = s.NextDeploymentNumber(app.ID)
	if n != 2 {
		t.Fatalf("numero deveria ser 2, obtido %d", n)
	}
	last, err := s.LastReadyDeployment(app.ID, 10)
	if err != nil || last.ID != dep.ID {
		t.Fatalf("last ready divergiu: %v %v", last, err)
	}
	if err := s.DeleteApp(app.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetApp(app.ID); err != ErrNotFound {
		t.Fatal("app deveria ter sido removido")
	}
}

func TestDomainsAndBackups(t *testing.T) {
	s := testStore(t)
	d := &domain.Domain{ID: domain.NewID(), AppID: "a1", Host: "ex.com", HTTPS: true, CertStatus: "pending", CreatedAt: time.Now()}
	if err := s.CreateDomain(d); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetDomainByHost("ex.com")
	if err != nil {
		t.Fatal(err)
	}
	if !got.HTTPS {
		t.Fatal("https deveria ser true")
	}
	if err := s.UpdateDomainCert("ex.com", "issued"); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetDomainByHost("ex.com")
	if got.CertStatus != "issued" {
		t.Fatal("cert status não atualizou")
	}
	b := &domain.Backup{ID: domain.NewID(), Path: "/x", Size: 10, CreatedAt: time.Now()}
	if err := s.CreateBackup(b); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListBackups(10)
	if err != nil || len(list) != 1 {
		t.Fatalf("backups divergiram: %v %v", list, err)
	}
}
