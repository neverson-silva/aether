package domain

import "time"

type DBEngine string

const (
	EnginePostgres DBEngine = "postgres"
	EngineMysql    DBEngine = "mysql"
	EngineMariaDB  DBEngine = "mariadb"
	EngineRedis    DBEngine = "redis"
	EngineMongoDB  DBEngine = "mongodb"
	EngineMSSQL    DBEngine = "mssql"
	EngineOracle   DBEngine = "oracle"
)

type Database struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Engine      DBEngine  `json:"engine"`
	Version     string    `json:"version"`
	Port        int       `json:"port"`
	DBName      string    `json:"db_name"`
	User        string    `json:"user"`
	StorageMB   int64     `json:"storage_mb"`
	PassEnc     string    `json:"-"`
	MemMB       int64     `json:"mem_mb"`
	Status      string    `json:"status"`
	ContainerID string    `json:"container_id"`
	CreatedAt   time.Time `json:"created_at"`
}

func (d *Database) AppIDRef() string { return "db-" + d.ID }

type CronJob struct {
	ID        string    `json:"id"`
	AppID     string    `json:"app_id"`
	Name      string    `json:"name"`
	Schedule  string    `json:"schedule"`
	Command   string    `json:"command"`
	Enabled   bool      `json:"enabled"`
	LastRun   time.Time `json:"last_run"`
	NextRun   time.Time `json:"next_run"`
	CreatedAt time.Time `json:"created_at"`
}

type Worker struct {
	ID          string    `json:"id"`
	AppID       string    `json:"app_id"`
	Name        string    `json:"name"`
	Command     string    `json:"command"`
	Replicas    int       `json:"replicas"`
	Enabled     bool      `json:"enabled"`
	Status      string    `json:"status"`
	ContainerID string    `json:"container_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type S3Destination struct {
	ID           string    `json:"id"`
	OrgID        string    `json:"org_id"`
	Name         string    `json:"name"`
	Endpoint     string    `json:"endpoint"`
	Bucket       string    `json:"bucket"`
	Region       string    `json:"region"`
	AccessKeyEnc string    `json:"-"`
	SecretKeyEnc string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type NotificationChannel struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	ConfigEnc string    `json:"-"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type Template struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Category      string    `json:"category"`
	Tags          []string  `json:"tags"`
	Icon          string    `json:"icon"`
	Version       string    `json:"version"`
	Definition    string    `json:"definition"`
	ComposeYAML   string    `json:"compose_yaml,omitempty"`
	Readme        string    `json:"readme"`
	Homepage      string    `json:"homepage"`
	GitHub        string    `json:"github"`
	License       string    `json:"license"`
	Installs      int       `json:"installs"`
	Featured      bool      `json:"featured"`
	EditorsChoice bool      `json:"editors_choice"`
	Verified      bool      `json:"verified"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Preview struct {
	ID           string    `json:"id"`
	AppID        string    `json:"app_id"`
	Branch       string    `json:"branch"`
	DeploymentID string    `json:"deployment_id"`
	ContainerID  string    `json:"container_id"`
	Domain       string    `json:"domain"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type TemplateService struct {
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	Port    int               `json:"port"`
	Env     map[string]string `json:"env"`
	Volumes []string          `json:"volumes"`
	Command string            `json:"command"`
}

type TemplateDefinition struct {
	Services []TemplateService `json:"services"`
}

type OutWebhook struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	SecretEnc string    `json:"-"`
	Events    []string  `json:"events"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type RegistrySettings struct {
	ID          string `json:"id"`
	Enabled     bool   `json:"enabled"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	ContainerID string `json:"container_id"`
	Status      string `json:"status"`
}

type RegistryImage struct {
	Repo string   `json:"repo"`
	Tags []string `json:"tags"`
	Size int64    `json:"size"`
}

type AppPolicy struct {
	AppID        string    `json:"app_id"`
	Enabled      bool      `json:"enabled"`
	CPUMin       float64   `json:"cpu_min"`
	CPUMax       float64   `json:"cpu_max"`
	MemMinMB     int64     `json:"mem_min_mb"`
	MemMaxMB     int64     `json:"mem_max_mb"`
	ScaleUpPct   int       `json:"scale_up_pct"`
	ScaleDownPct int       `json:"scale_down_pct"`
	CooldownMin  int       `json:"cooldown_min"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AutopilotEvent struct {
	ID        string    `json:"id"`
	AppID     string    `json:"app_id"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

type GitOps struct {
	ID           string    `json:"id"`
	OrgID        string    `json:"org_id"`
	Name         string    `json:"name"`
	RepoURL      string    `json:"repo_url"`
	Branch       string    `json:"branch"`
	Path         string    `json:"path"`
	TargetOrgID  string    `json:"target_org_id"`
	ApplyMode    string    `json:"apply_mode"`
	LastSHA      string    `json:"last_sha"`
	LastStatus   string    `json:"last_status"`
	DriftAdded   int       `json:"drift_added"`
	DriftChanged int       `json:"drift_changed"`
	DriftRemoved int       `json:"drift_removed"`
	LastSync     string    `json:"last_sync"`
	CreatedAt    time.Time `json:"created_at"`
}

type RegistryMirror struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Source        string    `json:"source"`
	Dest          string    `json:"dest"`
	DestTLSVerify bool      `json:"dest_tls_verify"`
	TagsFilter    string    `json:"tags_filter"`
	Schedule      string    `json:"schedule"`
	LastRun       string    `json:"last_run"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type Snapshot struct {
	ID         string    `json:"id"`
	OrgID      string    `json:"org_id"`
	AppID      string    `json:"app_id"`
	Volume     string    `json:"volume"`
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	Chunks     int       `json:"chunks"`
	DedupSaved int64     `json:"dedup_saved"`
	CreatedAt  time.Time `json:"created_at"`
}

type Branding struct {
	OrgID        string    `json:"org_id"`
	Name         string    `json:"name"`
	LogoURL      string    `json:"logo_url"`
	PrimaryColor string    `json:"primary_color"`
	AccentColor  string    `json:"accent_color"`
	DarkMode     bool      `json:"dark_mode"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PipelineStage struct {
	Name     string   `json:"name"`
	Image    string   `json:"image"`
	Commands []string `json:"commands"`
}

type Pipeline struct {
	ID        string          `json:"id"`
	OrgID     string          `json:"org_id"`
	AppID     string          `json:"app_id"`
	Name      string          `json:"name"`
	Trigger   string          `json:"trigger"`
	Stages    []PipelineStage `json:"stages"`
	Enabled   bool            `json:"enabled"`
	CreatedAt time.Time       `json:"created_at"`
}

type PipelineRun struct {
	ID         string    `json:"id"`
	PipelineID string    `json:"pipeline_id"`
	Status     string    `json:"status"`
	Trigger    string    `json:"trigger"`
	Log        string    `json:"log"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}

type OIDCProvider struct {
	ID              string    `json:"id"`
	OrgID           string    `json:"org_id"`
	Name            string    `json:"name"`
	Issuer          string    `json:"issuer"`
	ClientID        string    `json:"client_id"`
	ClientSecretEnc string    `json:"-"`
	Scopes          string    `json:"scopes"`
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
}

type Cluster struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	Labels    []string  `json:"labels"`
	CreatedAt time.Time `json:"created_at"`
}

type Environment struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type EnvironmentVariable struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"project_id"`
	EnvironmentID string    `json:"environment_id"`
	Key           string    `json:"key"`
	Value         string    `json:"value"`
	IsSecret      bool      `json:"is_secret"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type VariableAudit struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"project_id"`
	EnvironmentID string    `json:"environment_id"`
	Action        string    `json:"action"`
	Key           string    `json:"key"`
	PreviousValue string    `json:"previous_value"`
	CreatedAt     time.Time `json:"created_at"`
}

type ProjectVariable struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	IsSecret  bool      `json:"is_secret"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Notification struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Payload   string    `json:"payload"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

type ZipUpload struct {
	ID     string `json:"upload_id"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Status string `json:"status"`
}
