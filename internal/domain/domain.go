package domain

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type Role string

const (
	RoleOwner     Role = "owner"
	RoleAdmin     Role = "admin"
	RoleMember    Role = "member"
	RoleDeveloper Role = "developer" // alias histórico de member
	RoleViewer    Role = "viewer"

	GlobalAdmin = "global:admin"
)

// IsOrgRole reports whether r is a known organization role.
func (r Role) IsOrgRole() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember, RoleDeveloper, RoleViewer:
		return true
	}
	return false
}

// Normalize maps legacy roles (developer) to the current role set (member).
func (r Role) Normalize() Role {
	if r == RoleDeveloper {
		return RoleMember
	}
	return r
}

// AtLeast reports whether r grants at least the privileges of min.
// Hierarchy: viewer < member < admin < owner.
func (r Role) AtLeast(min Role) bool {
	rank := map[Role]int{
		RoleViewer: 0, RoleMember: 1, RoleDeveloper: 1,
		RoleAdmin: 2, RoleOwner: 3,
	}
	return rank[r.Normalize()] >= rank[min.Normalize()]
}

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	GlobalRole   string    `json:"global_role"`
	CreatedAt    time.Time `json:"created_at"`
}

type Org struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Avatar      string    `json:"avatar"`
	Color       string    `json:"color"`
	OwnerUserID string    `json:"owner_user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   time.Time `json:"deleted_at"`
}

type Member struct {
	OrgID  string `json:"org_id"`
	UserID string `json:"user_id"`
	Role   Role   `json:"role"`
}

// ProjectAssignment scopes a member to specific projects of an organization.
// Owner/Admin see everything; Member/Viewer only see assigned projects.
type ProjectAssignment struct {
	OrgID     string    `json:"org_id"`
	UserID    string    `json:"user_id"`
	ProjectID string    `json:"project_id"`
	CreatedAt time.Time `json:"created_at"`
}

type AuditLog struct {
	ID           string    `json:"id"`
	OrgID        string    `json:"org_id"`
	UserID       string    `json:"user_id"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	Details      string    `json:"details"`
	CreatedAt    time.Time `json:"created_at"`
}

type ApiKey struct {
	ID         string    `json:"id"`
	OrgID      string    `json:"org_id"`
	UserID     string    `json:"user_id"`
	Name       string    `json:"name"`
	KeyHash    string    `json:"-"`
	Scopes     []string  `json:"scopes"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type Project struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   time.Time `json:"deleted_at"`
}

type SourceType string

const (
	SourceImage SourceType = "image"
	SourceGit   SourceType = "git"
)

type Resources struct {
	CPUs      string `json:"cpus"`
	MemMB     int64  `json:"mem_mb"`
	StorageMB int64  `json:"storage_mb"`
}

type HealthCheck struct {
	Enabled    bool   `json:"enabled"`
	Path       string `json:"path"`
	IntervalMS int    `json:"interval_ms"`
	TimeoutMS  int    `json:"timeout_ms"`
	Retries    int    `json:"retries"`
}

type Volume struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
}

type App struct {
	ID             string      `json:"id"`
	OrgID          string      `json:"org_id"`
	ProjectID      string      `json:"project_id"`
	Name           string      `json:"name"`
	SourceType     SourceType  `json:"source_type"`
	Image          string      `json:"image"`
	GitURL         string      `json:"git_url"`
	GitBranch      string      `json:"git_branch"`
	Dockerfile     string      `json:"dockerfile"`
	Port           int         `json:"port"`
	Resources      Resources   `json:"resources"`
	HealthCheck    HealthCheck `json:"health_check"`
	Volumes        []Volume    `json:"volumes"`
	BuildType      string      `json:"build_type"`
	PreviewDomain  string      `json:"preview_domain"`
	ServerID       string      `json:"server_id"`
	ClusterID      string      `json:"cluster_id"`
	EnvironmentID  string      `json:"environment_id"`
	ImageRetention int         `json:"image_retention"`
	UploadID       string      `json:"upload_id"`
	InstallCommand string      `json:"install_command"`
	BuildCommand   string      `json:"build_command"`
	StartCommand   string      `json:"start_command"`
	RootFolder     string      `json:"root_folder"`
	DistFolder     string      `json:"dist_folder"`
	WatchPaths     string      `json:"watch_paths"`
	WebhookSecret  string      `json:"-"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type EnvVar struct {
	AppID  string `json:"app_id"`
	Name   string `json:"name"`
	Value  []byte `json:"value"`
	Secret bool   `json:"secret"`
}

type DeploymentStatus string

const (
	DeploymentQueued         DeploymentStatus = "queued"
	DeploymentBuilding       DeploymentStatus = "building"
	DeploymentStarting       DeploymentStatus = "starting"
	DeploymentHealthChecking DeploymentStatus = "health_checking"
	DeploymentReady          DeploymentStatus = "ready"
	DeploymentFailed         DeploymentStatus = "failed"
	DeploymentRolledBack     DeploymentStatus = "rolled_back"
	DeploymentCancelled      DeploymentStatus = "cancelled"
)

type Deployment struct {
	ID          string           `json:"id"`
	AppID       string           `json:"app_id"`
	Number      int64            `json:"number"`
	Status      DeploymentStatus `json:"status"`
	Trigger     string           `json:"trigger"`
	TriggeredBy string           `json:"triggered_by"`
	EnvSnapshot string           `json:"env_snapshot"`
	Commit      string           `json:"commit"`
	ImageRef    string           `json:"image_ref"`
	ContainerID string           `json:"container_id"`
	ServerID    string           `json:"server_id"`
	Error       string           `json:"error"`
	ComposeYAML string           `json:"compose_yaml"`
	DeploySpec  string           `json:"deploy_spec"`
	ComposeHash string           `json:"compose_hash"`
	CreatedAt   time.Time        `json:"created_at"`
	StartedAt   time.Time        `json:"started_at"`
	FinishedAt  time.Time        `json:"finished_at"`
}

type Domain struct {
	ID         string    `json:"id"`
	AppID      string    `json:"app_id"`
	Host       string    `json:"host"`
	HTTPS      bool      `json:"https"`
	CertStatus string    `json:"cert_status"`
	CreatedAt  time.Time `json:"created_at"`
}

type Backup struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	Kind      string    `json:"kind"`
	Dest      string    `json:"dest"`
	AppID     string    `json:"app_id"`
}

var (
	idMu    sync.Mutex
	lastMS  int64
	lastSeq uint16
)

func randByte() byte {
	b := make([]byte, 1)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b[0]
}

// NewID gera um UUID v7 (time-ordered): 48 bits de timestamp unix_ms +
// contador monotônico de 12 bits (ordenação estrita dentro do mesmo ms) +
// variante + 62 bits aleatórios. Formato: 32 hex sem hífens (compatível com
// o esquema text existente), mas com localidade de índice e ordenação por
// tempo — ideal para B-tree no Postgres.
func NewID() string {
	idMu.Lock()
	defer idMu.Unlock()
	now := time.Now().UnixMilli()
	if now <= lastMS {
		now = lastMS
		lastSeq++
	} else {
		lastSeq = 0
	}
	lastMS = now

	var b [16]byte
	// 0..5  = timestamp (48 bits, big-endian)
	b[0] = byte(now >> 40)
	b[1] = byte(now >> 32)
	b[2] = byte(now >> 24)
	b[3] = byte(now >> 16)
	b[4] = byte(now >> 8)
	b[5] = byte(now)
	// 6..7  = versão 7 (alto nibble) + rand_a 12 bits (contador monotônico)
	b[6] = 0x70 | byte((lastSeq>>8)&0x0f)
	b[7] = byte(lastSeq & 0xff)
	// 8     = variante (0b10) + 6 bits aleatórios
	b[8] = 0x80 | (randByte() & 0x3f)
	// 9..15 = rand_b (62 bits aleatórios)
	b[9] = randByte()
	if _, err := rand.Read(b[10:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

type Server struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Host          string    `json:"host"`
	Role          string    `json:"role"`
	Status        string    `json:"status"`
	Version       string    `json:"version"`
	Labels        []string  `json:"labels"`
	CPUCores      int       `json:"cpu_cores"`
	MemTotalBytes int64     `json:"mem_total_bytes"`
	Load          float64   `json:"load"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	ClusterID     string    `json:"cluster_id"`
	CreatedAt     time.Time `json:"created_at"`
}
