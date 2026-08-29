package domain

import (
	"time"

	"github.com/google/uuid"
)

type TriggerType string

const (
	TriggerManual    TriggerType = "manual"
	TriggerScheduled TriggerType = "scheduled"
)

type BackupStatus string

const (
	BackupQueued     BackupStatus = "queued"
	BackupPreparing  BackupStatus = "preparing"
	BackupRunning    BackupStatus = "running"
	BackupUploading  BackupStatus = "uploading"
	BackupVerifying  BackupStatus = "verifying"
	BackupCompleted  BackupStatus = "completed"
	BackupFailed     BackupStatus = "failed"
	BackupCancelling BackupStatus = "cancelling"
	BackupCancelled  BackupStatus = "cancelled"
)

type BackupJob struct {
	ID            uuid.UUID
	DatabaseID    uuid.UUID
	ServiceID     uuid.UUID
	Configuration *uuid.UUID
	Trigger       TriggerType
	Status        BackupStatus
	Engine        string
	EngineVersion string
	Format        string
	DestinationID uuid.UUID
	StorageKey    string
	SizeBytes     int64
	Checksum      string
	ErrorCode     string
	ErrorMessage  string
	StartedAt     *time.Time
	CompletedAt   *time.Time
	CreatedAt     time.Time
}

func (j *BackupJob) Transition(to BackupStatus) error {
	if !validBackupTransition(j.Status, to) {
		return ErrValidation
	}
	j.Status = to
	return nil
}

func validBackupTransition(from, to BackupStatus) bool {
	switch from {
	case BackupQueued:
		return to == BackupPreparing || to == BackupCancelled || to == BackupFailed
	case BackupPreparing:
		return to == BackupRunning || to == BackupFailed || to == BackupCancelled
	case BackupRunning:
		return to == BackupUploading || to == BackupFailed || to == BackupCancelling || to == BackupCancelled
	case BackupUploading:
		return to == BackupVerifying || to == BackupFailed || to == BackupCancelling
	case BackupVerifying:
		return to == BackupCompleted || to == BackupFailed
	case BackupCancelling:
		return to == BackupCancelled || to == BackupFailed
	default:
		return false
	}
}

func (j *BackupJob) Terminal() bool {
	return j.Status == BackupCompleted || j.Status == BackupFailed || j.Status == BackupCancelled
}

type RestoreStatus string

const (
	RestoreQueued     RestoreStatus = "queued"
	RestoreUploading  RestoreStatus = "uploading"
	RestoreValidating RestoreStatus = "validating"
	RestoreReady      RestoreStatus = "ready"
	RestorePreparing  RestoreStatus = "preparing"
	RestoreDownload   RestoreStatus = "downloading"
	RestoreRunning    RestoreStatus = "restoring"
	RestoreVerifying  RestoreStatus = "verifying"
	RestoreCompleted  RestoreStatus = "completed"
	RestoreFailed     RestoreStatus = "failed"
	RestoreCancelling RestoreStatus = "cancelling"
	RestoreCancelled  RestoreStatus = "cancelled"
)

type RestoreSourceType string

const (
	RestoreSourceBackup RestoreSourceType = "backup"
	RestoreSourceUpload RestoreSourceType = "upload"
)

type RestoreJob struct {
	ID               uuid.UUID
	BackupID         *uuid.UUID
	TargetDatabaseID uuid.UUID
	ServiceID        uuid.UUID
	Status           RestoreStatus
	ErrorCode        string
	ErrorMessage     string
	SourceType       RestoreSourceType
	SourceFilename   string
	SourceSize       int64
	SourceChecksum   string
	SourceFormat     string
	UploadedBytes    int64
	StartedAt        *time.Time
	CompletedAt      *time.Time
	CreatedAt        time.Time
}

func (j *RestoreJob) Transition(to RestoreStatus) error {
	if !validRestoreTransition(j.Status, to) {
		return ErrValidation
	}
	j.Status = to
	return nil
}

func validRestoreTransition(from, to RestoreStatus) bool {
	switch from {
	case RestoreQueued:
		return to == RestoreUploading || to == RestorePreparing || to == RestoreCancelled || to == RestoreFailed
	case RestoreUploading:
		return to == RestoreValidating || to == RestoreCancelled || to == RestoreFailed
	case RestoreValidating:
		return to == RestoreReady || to == RestoreFailed || to == RestoreCancelled
	case RestoreReady:
		return to == RestorePreparing || to == RestoreCancelled || to == RestoreFailed
	case RestorePreparing:
		return to == RestoreDownload || to == RestoreFailed || to == RestoreCancelled
	case RestoreDownload:
		return to == RestoreRunning || to == RestoreFailed || to == RestoreCancelling
	case RestoreRunning:
		return to == RestoreVerifying || to == RestoreFailed || to == RestoreCancelling
	case RestoreVerifying:
		return to == RestoreCompleted || to == RestoreFailed
	case RestoreCancelling:
		return to == RestoreCancelled || to == RestoreFailed
	default:
		return false
	}
}

func (j *RestoreJob) Terminal() bool {
	return j.Status == RestoreCompleted || j.Status == RestoreFailed || j.Status == RestoreCancelled
}
