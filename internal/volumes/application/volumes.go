package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	appsdomain "aether/internal/apps/domain"
	settingsdomain "aether/internal/settings/domain"
	"aether/internal/volumes/domain"
)

type Volumes struct {
	Store        domain.Store
	Apps         AppStore
	Destinations DestinationStore
}

type AppStore interface {
	GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error)
}

type DestinationStore interface {
	GetS3(ctx context.Context, id, orgID uuid.UUID) (*settingsdomain.S3Destination, error)
}

func (v *Volumes) BackupVolume(ctx context.Context, appID, orgID, destinationID uuid.UUID, volumeName string) (*domain.Backup, error) {
	app, err := v.Apps.GetApp(ctx, appID, orgID)
	if err != nil {
		return nil, err
	}
	dest, err := v.Destinations.GetS3(ctx, destinationID, orgID)
	if err != nil {
		return nil, err
	}
	volume, err := v.Store.GetVolumeByApp(ctx, appID, strings.TrimSpace(volumeName))
	if err != nil {
		return nil, err
	}
	return v.Store.CreateBackup(ctx, &domain.Backup{
		OrgID: orgID, AppID: &appID, Kind: "volume", Dest: dest.Name,
		Path: fmt.Sprintf("vol-backups/%s/%s/%s-%d.tar", dest.Bucket, app.Name, volume.Name, time.Now().Unix()),
	})
}

func (v *Volumes) List(ctx context.Context, appID, orgID uuid.UUID) ([]domain.Volume, error) {
	if _, err := v.Apps.GetApp(ctx, appID, orgID); err != nil {
		return nil, err
	}
	return v.Store.ListVolumesByApp(ctx, appID)
}
