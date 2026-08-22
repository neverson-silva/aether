package domain

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Provider string

const ProviderGitHub Provider = "github"

type Connection struct {
	ID                  uuid.UUID
	OrganizationID      uuid.UUID
	Provider            Provider
	ExternalAccountID   string
	ExternalAccountName string
	InstallationID      string
	Status              string
	Metadata            []byte
	CredentialsEnc      string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type GitHubAppCredentials struct {
	AppID         int64  `json:"app_id"`
	Slug          string `json:"slug"`
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	PrivateKey    string `json:"private_key"`
	WebhookSecret string `json:"webhook_secret"`
}

type ServiceSource struct {
	ID                 uuid.UUID
	ServiceID          uuid.UUID
	ConnectionID       uuid.UUID
	OrganizationID     uuid.UUID
	RepositoryID       string
	RepositoryOwner    string
	RepositoryName     string
	RepositoryFullName string
	DefaultBranch      string
	Branch             string
	AutoDeploy         bool
	RootDirectory      string
	WatchPaths         []string
	IgnorePaths        []string
	WatchRootFiles     bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Repository struct {
	ID            string
	Owner         string
	Name          string
	FullName      string
	DefaultBranch string
}

type Installation struct {
	ID          string
	AccountID   string
	AccountName string
}

type Branch struct {
	Name string
}

type Commit struct {
	SHA     string
	Message string
	Author  string
}

type CloneCredential struct {
	Username  string
	Secret    string
	ExpiresAt time.Time
}

type SourceEvent struct {
	Provider       Provider
	DeliveryID     string
	EventType      string
	InstallationID string
	Repository     Repository
	OccurredAt     time.Time
}

type PushEvent struct {
	SourceEvent
	Ref       string
	Branch    string
	BeforeSHA string
	AfterSHA  string
	Commit    Commit
}

type TriggerReason string

const (
	ReasonBranchNotMatched    TriggerReason = "branch_not_matched"
	ReasonAutodeployDisabled  TriggerReason = "autodeploy_disabled"
	ReasonNoRelevantChanges   TriggerReason = "no_relevant_changes"
	ReasonWatchPathMatched    TriggerReason = "watch_path_matched"
	ReasonRootFileMatched     TriggerReason = "root_file_matched"
	ReasonChangedFilesUnknown TriggerReason = "changed_files_unknown_fallback"
)

type TriggerDecision struct {
	Trigger bool
	Reason  TriggerReason
	Matches []string
}

type BuildTriggerRules struct {
	Branch         string
	AutoDeploy     bool
	RootDirectory  string
	WatchPaths     []string
	IgnorePaths    []string
	WatchRootFiles bool
}

type WebhookDelivery struct {
	ID             uuid.UUID
	Provider       Provider
	DeliveryID     string
	EventType      string
	InstallationID string
	RepositoryID   string
	Status         string
	Error          string
	ReceivedAt     time.Time
	ProcessedAt    *time.Time
}

type SourceProvider interface {
	ListRepositories(ctx context.Context, installationID string) ([]Repository, error)
	GetRepository(ctx context.Context, repoID string) (Repository, error)
	GetBranches(ctx context.Context, repoID string) ([]Branch, error)
	GetCommit(ctx context.Context, repoID string, sha string) (Commit, error)
	GetChangedFiles(ctx context.Context, repoID string, before string, after string) ([]string, error)
	CreateCloneCredential(ctx context.Context, repoID string) (CloneCredential, error)
}

type SourceEventProvider interface {
	VerifyWebhook(headers http.Header, body []byte) error
	ParseWebhook(headers http.Header, body []byte) ([]SourceEvent, error)
}
