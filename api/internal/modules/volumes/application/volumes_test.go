package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	appsdomain "aether/internal/modules/apps/domain"
	appsInfra "aether/internal/modules/apps/infra"
	settingsdomain "aether/internal/modules/settings/domain"
	settingsInfra "aether/internal/modules/settings/infra"
	"aether/internal/modules/volumes/domain"
	"aether/internal/modules/volumes/infra"
)

type env struct {
	ctx    context.Context
	pool   *pgxpool.Pool
	svc    *Volumes
	orgID  uuid.UUID
	appID  uuid.UUID
	destID uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	appsStore := appsInfra.NewStore(pool)
	settingsStore := settingsInfra.NewStore(pool)
	_, _ = settingsInfra.NewPasswordCipher([]byte("0123456789abcdef0123456789abcdef"))
	e := &env{ctx: context.Background(), pool: pool, svc: &Volumes{Store: store, Apps: appsStore, Destinations: settingsStore}, orgID: uuid.New()}
	t.Cleanup(func() {
		_ = store.Close()
		_ = appsStore.Close()
		_ = settingsStore.Close()
	})
	if _, err := pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, e.orgID, "Vol Org", "vol-org"); err != nil {
		t.Fatalf("criar org: %v", err)
	}
	project, err := appsStore.CreateProject(e.ctx, e.orgID, "VolProj", "vol-proj", "", "")
	if err != nil {
		t.Fatalf("criar project: %v", err)
	}
	app, err := appsStore.CreateApp(e.ctx, &appsdomain.App{OrgID: e.orgID, ProjectID: project.ID, Name: "api", SourceType: "image", Image: "nginx", Port: 80})
	if err != nil {
		t.Fatalf("criar app: %v", err)
	}
	e.appID = app.ID
	dest, err := settingsStore.CreateS3(e.ctx, &settingsdomain.S3Destination{
		OrgID: e.orgID, Name: "backups", Endpoint: "s3.amazonaws.com", Bucket: "bkt",
		AccessKeyEnc: "a", SecretKeyEnc: "b",
	})
	if err != nil {
		t.Fatalf("criar s3: %v", err)
	}
	e.destID = dest.ID
	if _, err := store.CreateVolume(e.ctx, &domain.Volume{AppID: e.appID, Name: "data", MountPath: "/data"}); err != nil {
		t.Fatalf("criar volume: %v", err)
	}
	return e
}

func TestVolumeBackup(t *testing.T) {
	e := newEnv(t)
	backup, err := e.svc.BackupVolume(e.ctx, e.appID, e.orgID, e.destID, "data")
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if backup.Kind != "volume" || backup.Dest != "backups" || backup.AppID == nil {
		t.Fatalf("backup inesperado: %+v", backup)
	}
	if backup.Path == "" {
		t.Fatalf("path deveria estar presente")
	}
	if _, err := e.svc.BackupVolume(e.ctx, e.appID, e.orgID, e.destID, "nao-existe"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("volume inexistente deveria falhar: %v", err)
	}
}

func TestVolumeList(t *testing.T) {
	e := newEnv(t)
	volumes, err := e.svc.List(e.ctx, e.appID, e.orgID)
	if err != nil || len(volumes) != 1 || volumes[0].Name != "data" {
		t.Fatalf("list: %v %d", err, len(volumes))
	}
}
