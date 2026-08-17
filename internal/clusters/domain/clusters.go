package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict")
	ErrValidation = errors.New("invalid input")
	ErrForbidden  = errors.New("access denied")
)

type Cluster struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	Name      string
	Labels    []string
	CreatedAt time.Time
}

type Server struct {
	ID            uuid.UUID
	Name          string
	Host          string
	Role          string
	Status        string
	Version       string
	Labels        []string
	CPUCores      int
	MemTotalBytes int64
	Load          float64
	LastHeartbeat *time.Time
	ClusterID     *uuid.UUID
	CreatedAt     time.Time
}

type Registry struct {
	Enabled     bool
	Host        string
	Port        int
	ContainerID string
	Status      string
}

type RegistryImage struct {
	Repo    string
	Tag     string
	ID      string
	Size    int64
	Created int64
}

type Store interface {
	CreateCluster(ctx context.Context, cluster *Cluster) (*Cluster, error)
	GetCluster(ctx context.Context, id uuid.UUID) (*Cluster, error)
	ListClustersByOrg(ctx context.Context, orgID uuid.UUID) ([]Cluster, error)
	DeleteCluster(ctx context.Context, id, orgID uuid.UUID) error

	ListServers(ctx context.Context) ([]Server, error)
	GetServer(ctx context.Context, id uuid.UUID) (*Server, error)
	SetServerCluster(ctx context.Context, serverID uuid.UUID, clusterID *uuid.UUID) error
	DeleteServer(ctx context.Context, id uuid.UUID) error

	GetRegistry(ctx context.Context) (*Registry, error)
	SetRegistryEnabled(ctx context.Context, enabled bool) (*Registry, error)

	CreateServerToken(ctx context.Context, tokenHash string) error
}
