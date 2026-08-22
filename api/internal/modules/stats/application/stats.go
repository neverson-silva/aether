package application

import (
	"context"
	"errors"
	"io"
	"time"

	"aether/internal/platform/worker"

	"github.com/google/uuid"

	appsdomain "aether/internal/modules/apps/domain"
	databasedomain "aether/internal/modules/databases/domain"
	deploydomain "aether/internal/modules/deployments/domain"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("invalid input")
)

type Runtime interface {
	Stats(ctx context.Context, containerID string) (worker.ContainerStats, error)
	ContainerState(ctx context.Context, containerID string) (string, error)
	FollowLogs(ctx context.Context, containerID string, writer io.Writer) error
}

type Stats struct {
	Apps        AppStore
	Deployments DeploymentStore
	Databases   DatabaseStore
	Runtime     Runtime
}

type AppStore interface {
	GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error)
}

type DeploymentStore interface {
	ListByApp(ctx context.Context, appID uuid.UUID, limit int) ([]deploydomain.Deployment, error)
}

type DatabaseStore interface {
	GetDatabase(ctx context.Context, id uuid.UUID) (*databasedomain.Database, error)
}

type ContainerInfo struct {
	State string                `json:"state"`
	Stats worker.ContainerStats `json:"stats"`
}

func (s *Stats) AppStats(ctx context.Context, appID, orgID uuid.UUID) (ContainerInfo, error) {
	if _, err := s.Apps.GetApp(ctx, appID, orgID); err != nil {
		return ContainerInfo{}, err
	}
	deps, err := s.Deployments.ListByApp(ctx, appID, 1)
	if err != nil || len(deps) == 0 || deps[0].ContainerID == "" {
		return ContainerInfo{}, ErrNotFound
	}
	return s.containerInfo(ctx, deps[0].ContainerID)
}

func (s *Stats) DatabaseStats(ctx context.Context, dbID, orgID uuid.UUID) (ContainerInfo, error) {
	db, err := s.Databases.GetDatabase(ctx, dbID)
	if err != nil {
		return ContainerInfo{}, err
	}
	if db.OrgID != orgID {
		return ContainerInfo{}, ErrNotFound
	}
	if db.ContainerID == "" {
		return ContainerInfo{}, ErrNotFound
	}
	return s.containerInfo(ctx, db.ContainerID)
}

func (s *Stats) Database(ctx context.Context, dbID, orgID uuid.UUID) (*databasedomain.Database, error) {
	db, err := s.Databases.GetDatabase(ctx, dbID)
	if err != nil {
		return nil, err
	}
	if db.OrgID != orgID {
		return nil, ErrNotFound
	}
	return db, nil
}

func (s *Stats) FollowLogs(ctx context.Context, containerID string, writer io.Writer) error {
	return s.Runtime.FollowLogs(ctx, containerID, writer)
}

func (s *Stats) containerInfo(ctx context.Context, containerID string) (ContainerInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	state, _ := s.Runtime.ContainerState(ctx, containerID)
	stats, err := s.Runtime.Stats(ctx, containerID)
	if err != nil {
		return ContainerInfo{State: state}, nil
	}
	return ContainerInfo{State: state, Stats: stats}, nil
}
