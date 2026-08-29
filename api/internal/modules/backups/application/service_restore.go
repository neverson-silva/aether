package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/backups/domain"
	"aether/internal/platform/druntime/events"
	"aether/internal/platform/druntime/queue"
)

func (s *DatabaseBackups) RequestRestore(ctx context.Context, backupID, targetDBID, orgID uuid.UUID) (*domain.RestoreJob, error) {
	backup, err := s.GetBackup(ctx, backupID, orgID)
	if err != nil {
		return nil, err
	}
	if backup.Status != domain.BackupCompleted {
		return nil, domain.ErrValidation
	}
	target, err := s.Databases.Get(ctx, targetDBID, orgID)
	if err != nil {
		return nil, err
	}
	if string(target.Engine) != backup.Engine {
		return nil, domain.ErrValidation
	}
	if err := validateRestoreCompatibility(backup.Engine, backup.EngineVersion, target.Version); err != nil {
		return nil, err
	}
	if err := s.ensureNoActiveRestore(ctx, targetDBID); err != nil {
		return nil, err
	}
	job, err := s.Store.CreateRestoreJob(ctx, &domain.RestoreJob{
		BackupID: &backupID, TargetDatabaseID: targetDBID, ServiceID: target.ServiceID, Status: domain.RestoreQueued,
		SourceType: domain.RestoreSourceBackup, SourceSize: backup.SizeBytes,
		SourceChecksum: backup.Checksum, SourceFormat: backup.Format,
	})
	if err != nil {
		return nil, err
	}
	s.Audit.Record(ctx, orgID, "restore.started", "database", targetDBID.String(), backupID.String())
	s.notifyRestore(ctx, orgID, targetDBID, job.ID, string(job.Status))
	if err := s.enqueueRestore(ctx, job, orgID); err != nil {
		_ = s.failRestore(ctx, job, "RESTORE_ENQUEUE_FAILED", err.Error())
		return job, err
	}
	return job, nil
}

func (s *DatabaseBackups) CreateUploadRestore(ctx context.Context, dbID, orgID uuid.UUID, filename string) (*domain.RestoreJob, error) {
	target, err := s.Databases.Get(ctx, dbID, orgID)
	if err != nil {
		return nil, err
	}
	if err := validateUploadEngine(string(target.Engine)); err != nil {
		return nil, err
	}
	job, err := s.Store.CreateRestoreJob(ctx, &domain.RestoreJob{
		TargetDatabaseID: dbID, ServiceID: target.ServiceID, Status: domain.RestoreQueued,
		SourceType: domain.RestoreSourceUpload, SourceFilename: sanitizeSourceFilename(filename),
	})
	if err != nil {
		return nil, err
	}
	s.notifyRestore(ctx, orgID, dbID, job.ID, string(job.Status))
	return job, nil
}

func (s *DatabaseBackups) WriteUpload(ctx context.Context, dbID, restoreID, orgID uuid.UUID, src io.Reader, expectedSize int64) (*domain.RestoreJob, error) {
	job, err := s.uploadJob(ctx, dbID, restoreID, orgID)
	if err != nil {
		return nil, err
	}
	if job.Status != domain.RestoreQueued && job.Status != domain.RestoreUploading {
		return nil, domain.ErrValidation
	}
	maxBytes := s.maxUploadBytes()
	if expectedSize > maxBytes {
		_ = s.failUploadRestore(ctx, job, "RESTORE_UPLOAD_TOO_LARGE", fmt.Sprintf("file exceeds the maximum upload size of %d bytes", maxBytes))
		return job, domain.ErrValidation
	}
	if expectedSize > 0 {
		if err := s.ensureDiskSpace(ctx, expectedSize); err != nil {
			_ = s.failUploadRestore(ctx, job, "RESTORE_DISK_FULL", err.Error())
			return job, domain.ErrValidation
		}
	}
	if err := job.Transition(domain.RestoreUploading); err != nil {
		return nil, err
	}
	if _, err := s.Store.UpdateRestoreJob(ctx, job); err != nil {
		return nil, err
	}

	dir := filepath.Dir(s.uploadArtifactPath(job.ID))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		_ = s.failUploadRestore(ctx, job, "RESTORE_UPLOAD_STORAGE_FAILED", err.Error())
		return job, err
	}
	hash := sha256.New()
	file, err := os.Create(s.uploadArtifactPath(job.ID))
	if err != nil {
		_ = s.failUploadRestore(ctx, job, "RESTORE_UPLOAD_STORAGE_FAILED", err.Error())
		return job, err
	}
	defer file.Close()
	tracker := s.progressNotifier(ctx, orgID, dbID, job.ID)
	var uploaded int64
	buffer := make([]byte, 1<<20)
	for {
		n, readErr := src.Read(buffer)
		if n > 0 {
			if uploaded+int64(n) > maxBytes {
				s.removeUploadArtifact(job.ID)
				_ = s.failUploadRestore(ctx, job, "RESTORE_UPLOAD_TOO_LARGE", fmt.Sprintf("file exceeds the maximum upload size of %d bytes", maxBytes))
				return job, domain.ErrValidation
			}
			if _, writeErr := file.Write(buffer[:n]); writeErr != nil {
				s.removeUploadArtifact(job.ID)
				_ = s.failUploadRestore(ctx, job, "RESTORE_UPLOAD_STORAGE_FAILED", writeErr.Error())
				return job, writeErr
			}
			_, _ = hash.Write(buffer[:n])
			uploaded += int64(n)
			job.UploadedBytes = uploaded
			if uploaded%(512<<20) == 0 {
				if diskErr := s.ensureDiskSpace(ctx, 256<<20); diskErr != nil {
					s.removeUploadArtifact(job.ID)
					_ = s.failUploadRestore(ctx, job, "RESTORE_DISK_FULL", diskErr.Error())
					return job, diskErr
				}
				_, _ = s.Store.UpdateRestoreJob(ctx, job)
			}
			tracker(uploaded, expectedSize)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			s.removeUploadArtifact(job.ID)
			_ = s.failUploadRestore(ctx, job, "RESTORE_UPLOAD_INTERRUPTED", readErr.Error())
			return job, readErr
		}
	}
	if err := file.Sync(); err != nil {
		s.removeUploadArtifact(job.ID)
		_ = s.failUploadRestore(ctx, job, "RESTORE_UPLOAD_STORAGE_FAILED", err.Error())
		return job, err
	}
	if expectedSize > 0 && uploaded != expectedSize {
		s.removeUploadArtifact(job.ID)
		_ = s.failUploadRestore(ctx, job, "RESTORE_SIZE_MISMATCH", "uploaded size does not match the declared file size")
		return job, domain.ErrValidation
	}
	job.SourceSize = uploaded
	job.SourceChecksum = hex.EncodeToString(hash.Sum(nil))
	job.UploadedBytes = uploaded
	job.SourceFormat = DetectUploadFormat(s.uploadArtifactPath(job.ID), fileNameExt(job.SourceFilename))
	if err := job.Transition(domain.RestoreValidating); err != nil {
		s.removeUploadArtifact(job.ID)
		return nil, err
	}
	if _, err := s.Store.UpdateRestoreJob(ctx, job); err != nil {
		s.removeUploadArtifact(job.ID)
		return nil, err
	}
	s.notifyRestore(ctx, orgID, dbID, job.ID, string(job.Status))
	return s.ValidateUpload(ctx, dbID, restoreID, orgID)
}

func (s *DatabaseBackups) ValidateUpload(ctx context.Context, dbID, restoreID, orgID uuid.UUID) (*domain.RestoreJob, error) {
	job, err := s.uploadJob(ctx, dbID, restoreID, orgID)
	if err != nil {
		return nil, err
	}
	if job.Status != domain.RestoreValidating && job.Status != domain.RestoreReady {
		return nil, domain.ErrValidation
	}
	st, statErr := os.Stat(s.uploadArtifactPath(job.ID))
	if statErr != nil {
		_ = s.failUploadRestore(ctx, job, "RESTORE_UPLOAD_ARTIFACT_MISSING", statErr.Error())
		return job, domain.ErrValidation
	}
	if st.Size() == 0 {
		_ = s.failUploadRestore(ctx, job, "RESTORE_EMPTY_BACKUP", "uploaded file is empty")
		return job, domain.ErrValidation
	}
	target, err := s.Databases.Get(ctx, dbID, orgID)
	if err != nil {
		return nil, err
	}
	job.SourceFormat = DetectUploadFormat(s.uploadArtifactPath(job.ID), fileNameExt(job.SourceFilename))
	if err := validateFormatForEngine(string(target.Engine), job.SourceFormat, job.SourceFilename); err != nil {
		_ = s.failUploadRestore(ctx, job, "RESTORE_INVALID_FORMAT", err.Error())
		return job, domain.ErrValidation
	}
	sourceVersion, err := detectUploadVersion(s.uploadArtifactPath(job.ID), job.SourceFormat, string(target.Engine))
	if err != nil {
		_ = s.failUploadRestore(ctx, job, "RESTORE_INVALID_FORMAT", err.Error())
		return job, domain.ErrValidation
	}
	if err := validateRestoreCompatibility(string(target.Engine), sourceVersion, target.Version); err != nil {
		_ = s.failUploadRestore(ctx, job, "RESTORE_VERSION_INCOMPATIBLE", err.Error())
		return job, domain.ErrValidation
	}
	if err := job.Transition(domain.RestoreReady); err != nil {
		_ = s.failUploadRestore(ctx, job, "RESTORE_STATE_INVALID", err.Error())
		return job, err
	}
	if _, err := s.Store.UpdateRestoreJob(ctx, job); err != nil {
		return nil, err
	}
	s.notifyRestore(ctx, orgID, dbID, job.ID, string(job.Status))
	return job, nil
}

func (s *DatabaseBackups) StartUploadRestore(ctx context.Context, dbID, restoreID, orgID uuid.UUID) (*domain.RestoreJob, error) {
	job, err := s.uploadJob(ctx, dbID, restoreID, orgID)
	if err != nil {
		return nil, err
	}
	if job.Status != domain.RestoreReady {
		return nil, domain.ErrValidation
	}
	if err := s.ensureNoActiveRestoreForUpload(ctx, dbID, job.ID); err != nil {
		return nil, err
	}
	if err := job.Transition(domain.RestorePreparing); err != nil {
		return nil, err
	}
	now := time.Now()
	job.StartedAt = &now
	if _, err := s.Store.UpdateRestoreJob(ctx, job); err != nil {
		return nil, err
	}
	s.Audit.Record(ctx, orgID, "restore.started", "database", dbID.String(), job.ID.String())
	s.notifyRestore(ctx, orgID, dbID, job.ID, string(job.Status))
	if err := s.enqueueRestore(ctx, job, orgID); err != nil {
		_ = s.failUploadRestore(ctx, job, "RESTORE_ENQUEUE_FAILED", err.Error())
		return job, err
	}
	return job, nil
}

func (s *DatabaseBackups) GetRestore(ctx context.Context, dbID, restoreID, orgID uuid.UUID) (*domain.RestoreJob, error) {
	job, err := s.Store.GetRestoreJob(ctx, restoreID)
	if err != nil {
		return nil, err
	}
	if job.TargetDatabaseID != dbID {
		return nil, domain.ErrNotFound
	}
	if _, err := s.Databases.Get(ctx, dbID, orgID); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *DatabaseBackups) CancelUploadRestore(ctx context.Context, dbID, restoreID, orgID uuid.UUID) error {
	job, err := s.uploadJob(ctx, dbID, restoreID, orgID)
	if err != nil {
		return err
	}
	switch job.Status {
	case domain.RestoreQueued, domain.RestoreUploading, domain.RestoreValidating, domain.RestoreReady:
		now := time.Now()
		job.CompletedAt = &now
		if err := job.Transition(domain.RestoreCancelled); err != nil {
			return err
		}
		if _, err := s.Store.UpdateRestoreJob(ctx, job); err != nil {
			return err
		}
		s.removeUploadArtifact(job.ID)
		s.Audit.Record(ctx, orgID, "restore.cancelled", "database", dbID.String(), job.ID.String())
		return nil
	default:
		return domain.ErrValidation
	}
}

func (s *DatabaseBackups) ListRestoreJobs(ctx context.Context, targetDBID, orgID uuid.UUID, limit int) ([]domain.RestoreJob, error) {
	if _, err := s.Databases.Get(ctx, targetDBID, orgID); err != nil {
		return nil, err
	}
	return s.Store.ListRestoreJobsByTarget(ctx, targetDBID, limit)
}

func (s *DatabaseBackups) enqueueRestore(ctx context.Context, job *domain.RestoreJob, orgID uuid.UUID) error {
	jobPayload := queue.Job{ID: job.ID.String(), Type: "restore", OrgID: orgID.String(), Payload: []byte(job.ID.String())}
	if s.Outbox != nil {
		payload, err := json.Marshal(jobPayload)
		if err != nil {
			return err
		}
		return s.Outbox.Enqueue(ctx, events.Event{ID: uuid.NewString(), Type: "restore.queued", AggregateType: "restore", AggregateID: job.ID.String(), Payload: payload, TS: time.Now().UTC()}, "backups")
	}
	if s.Queue == nil {
		return domain.ErrValidation
	}
	return s.Queue.Enqueue(ctx, "backups", jobPayload)
}

func (s *DatabaseBackups) uploadJob(ctx context.Context, dbID, restoreID, orgID uuid.UUID) (*domain.RestoreJob, error) {
	job, err := s.GetRestore(ctx, dbID, restoreID, orgID)
	if err != nil {
		return nil, err
	}
	if job.SourceType != domain.RestoreSourceUpload {
		return nil, domain.ErrValidation
	}
	return job, nil
}

func (s *DatabaseBackups) ensureNoActiveRestore(ctx context.Context, dbID uuid.UUID) error {
	return s.ensureNoActiveRestoreExcept(ctx, dbID, uuid.Nil)
}

func (s *DatabaseBackups) ensureNoActiveRestoreForUpload(ctx context.Context, dbID uuid.UUID, except uuid.UUID) error {
	return s.ensureNoActiveRestoreExcept(ctx, dbID, except)
}

func (s *DatabaseBackups) ensureNoActiveRestoreExcept(ctx context.Context, dbID uuid.UUID, except uuid.UUID) error {
	activeBackups, err := s.Store.ListActiveJobsByDatabase(ctx, dbID)
	if err != nil {
		return err
	}
	if len(activeBackups) > 0 {
		return domain.ErrConflict
	}
	restores, err := s.Store.ListRestoreJobsByTarget(ctx, dbID, 50)
	if err != nil {
		return err
	}
	for _, job := range restores {
		if job.ID == except {
			continue
		}
		if restoreJobActive(job.Status) {
			return domain.ErrConflict
		}
	}
	return nil
}

func restoreJobActive(status domain.RestoreStatus) bool {
	switch status {
	case domain.RestoreQueued, domain.RestoreUploading, domain.RestoreValidating, domain.RestoreReady,
		domain.RestorePreparing, domain.RestoreDownload, domain.RestoreRunning, domain.RestoreVerifying:
		return true
	default:
		return false
	}
}

func validateUploadEngine(engine string) error {
	switch engine {
	case "postgres", "mysql", "mariadb", "mssql", "oracle":
		return nil
	default:
		return domain.ErrValidation
	}
}

func (s *DatabaseBackups) maxUploadBytes() int64 {
	if s.MaxUploadBytes <= 0 {
		return 20 << 30
	}
	return s.MaxUploadBytes
}

func (s *DatabaseBackups) ensureDiskSpace(ctx context.Context, required int64) error {
	if s.UploadRoot == "" {
		return nil
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(s.UploadRoot, &stat); err != nil {
		return nil
	}
	available := int64(stat.Bavail) * int64(stat.Bsize)
	margin := int64(256 << 20)
	if available < required+margin {
		return errors.New("there is not enough disk space to process this backup")
	}
	return nil
}

func (s *DatabaseBackups) removeUploadArtifact(restoreID uuid.UUID) {
	_ = os.RemoveAll(filepath.Dir(s.uploadArtifactPath(restoreID)))
}

func (s *DatabaseBackups) failUploadRestore(ctx context.Context, rj *domain.RestoreJob, code, msg string) error {
	s.removeUploadArtifact(rj.ID)
	return s.failRestore(ctx, rj, code, msg)
}

func (s *DatabaseBackups) progressNotifier(ctx context.Context, orgID, dbID, restoreID uuid.UUID) func(uploaded, total int64) {
	if notifier, ok := s.Notifier.(RestoreProgressNotifier); ok {
		return func(uploaded, total int64) {
			notifier.NotifyRestoreProgress(ctx, orgID, dbID, restoreID, uploaded, total)
		}
	}
	return func(int64, int64) {}
}

type RestoreProgressNotifier interface {
	NotifyRestoreProgress(ctx context.Context, orgID, dbID, restoreID uuid.UUID, uploaded, total int64)
}

func fileNameExt(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".sql.gz"):
		return "sql.gz"
	case strings.HasSuffix(lower, ".gz"):
		return "gzip"
	case strings.HasSuffix(lower, ".dump"), strings.HasSuffix(lower, ".backup"), strings.HasSuffix(lower, ".dmp"), strings.HasSuffix(lower, ".sql"), strings.HasSuffix(lower, ".bak"), strings.HasSuffix(lower, ".tar"):
		return strings.TrimPrefix(filepath.Ext(lower), ".")
	default:
		return ""
	}
}

func validateFormatForEngine(engine, format, filename string) error {
	switch engine {
	case "postgres":
		switch format {
		case "dump", "tar", "sql", "sql.gz", "gzip":
			return nil
		}
	case "mysql", "mariadb":
		switch format {
		case "sql", "sql.gz", "gzip":
			return nil
		}
	case "mssql":
		switch format {
		case "bak":
			return nil
		}
	case "oracle":
		switch format {
		case "dmp", "gzip":
			return nil
		}
	}
	return fmt.Errorf("file %q is not a valid %s restore artifact", sanitizeSourceFilename(filename), engine)
}
