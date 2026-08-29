package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	appsdomain "aether/internal/modules/apps/domain"
	settingsdomain "aether/internal/modules/settings/domain"
	"aether/internal/modules/volumes/domain"
)

type Volumes struct {
	Store        domain.Store
	Apps         AppStore
	Destinations DestinationStore
}

type AppStore interface {
	GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error)
}

type AppServiceResolver interface {
	GetServiceID(ctx context.Context, appID uuid.UUID) (uuid.UUID, error)
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
	var serviceID *uuid.UUID
	if resolver, ok := v.Apps.(AppServiceResolver); ok {
		resolved, resolveErr := resolver.GetServiceID(ctx, appID)
		if resolveErr != nil {
			return nil, resolveErr
		}
		serviceID = &resolved
	}
	return v.Store.CreateBackup(ctx, &domain.Backup{
		OrgID: orgID, AppID: &appID, ServiceID: serviceID, Kind: "volume", Dest: dest.Name,
		Path: fmt.Sprintf("vol-backups/%s/%s/%s-%d.tar", dest.Bucket, app.Name, volume.Name, time.Now().Unix()),
	})
}

func (v *Volumes) List(ctx context.Context, appID, orgID uuid.UUID) ([]domain.Volume, error) {
	if _, err := v.Apps.GetApp(ctx, appID, orgID); err != nil {
		return nil, err
	}
	return v.Store.ListVolumesByApp(ctx, appID)
}

func (v *Volumes) BackupServiceVolume(ctx context.Context, serviceID, orgID, destinationID uuid.UUID, volumeName, serviceName string) (*domain.Backup, error) {
	dest, err := v.Destinations.GetS3(ctx, destinationID, orgID)
	if err != nil {
		return nil, err
	}
	volume, err := v.Store.GetVolumeByService(ctx, serviceID, strings.TrimSpace(volumeName))
	if err != nil {
		return nil, err
	}
	return v.Store.CreateBackup(ctx, &domain.Backup{
		OrgID: orgID, ServiceID: &serviceID, Kind: "volume", Dest: dest.Name,
		Path: fmt.Sprintf("vol-backups/%s/%s/%s-%d.tar", dest.Bucket, serviceName, volume.Name, time.Now().Unix()),
	})
}
