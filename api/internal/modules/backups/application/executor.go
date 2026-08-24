package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/backups/domain"
	"aether/internal/platform/storage"
)

func (s *DatabaseBackups) runBackup(ctx context.Context, orgID uuid.UUID, jobID uuid.UUID) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout())
	defer cancel()

	job, err := s.Store.GetJob(ctx, jobID)
	if err != nil || job.Terminal() {
		return
	}
	now := time.Now()
	job.StartedAt = &now
	if err := job.Transition(domain.BackupPreparing); err != nil {
		return
	}
	_, _ = s.Store.UpdateJob(ctx, job)
	s.notifyBackup(ctx, orgID, job.DatabaseID, job.ID, string(job.Status))

	if job.Configuration == nil {
		_ = s.failJob(ctx, orgID, job, "BACKUP_INVALID_CONFIGURATION", "no backup configuration")
		return
	}
	cfg, err := s.Store.GetConfiguration(ctx, *job.Configuration)
	if err != nil || cfg == nil || cfg.DatabaseID != job.DatabaseID {
		_ = s.failJob(ctx, orgID, job, "BACKUP_INVALID_CONFIGURATION", "no backup configuration")
		return
	}

	lock, ok, err := s.Locks.Acquire(ctx, "db:"+job.DatabaseID.String()+":backup", s.timeout())
	if err != nil {
		_ = s.failJob(ctx, orgID, job, "BACKUP_LOCK_FAILED", err.Error())
		return
	}
	if !ok {
		_ = s.failJob(ctx, orgID, job, "BACKUP_ALREADY_RUNNING", "another backup is running")
		return
	}
	defer func() { _ = s.Locks.Release(ctx, lock) }()

	db, err := s.Databases.Get(ctx, job.DatabaseID, orgID)
	if err != nil {
		_ = s.failJob(ctx, orgID, job, "BACKUP_DATABASE_UNAVAILABLE", err.Error())
		return
	}
	pass, err := s.Passwords.Decrypt(db.PassEnc)
	if err != nil {
		_ = s.failJob(ctx, orgID, job, "BACKUP_DATABASE_UNAVAILABLE", "cannot decrypt credentials")
		return
	}
	desc := DBDescriptor{
		Engine: string(db.Engine), ContainerID: db.ContainerID, User: db.User,
		Password: pass, DBName: db.DBName, Version: db.Version,
	}
	adapter, err := adapterForEngine(desc.Engine)
	if err != nil {
		_ = s.failJob(ctx, orgID, job, "BACKUP_UNSUPPORTED_ENGINE", err.Error())
		return
	}
	if err := adapter.Validate(ctx, desc); err != nil {
		_ = s.failJob(ctx, orgID, job, "BACKUP_TOOL_NOT_AVAILABLE", err.Error())
		return
	}
	job.Engine = desc.Engine
	job.EngineVersion = desc.Version
	job.Format = adapter.Format()
	job.DestinationID = cfg.DestinationID
	_ = job.Transition(domain.BackupRunning)
	_, _ = s.Store.UpdateJob(ctx, job)

	path, size, checksum, err := s.streamBackup(ctx, adapter, desc)
	if err != nil {
		_ = s.failJob(ctx, orgID, job, "BACKUP_COMMAND_FAILED", err.Error())
		return
	}
	defer os.Remove(path)
	if size == 0 {
		_ = s.failJob(ctx, orgID, job, "BACKUP_EMPTY_ARTIFACT", "backup command produced an empty artifact")
		return
	}
	job.SizeBytes = size
	job.Checksum = checksum

	provider, err := s.Destinations.GetProvider(ctx, cfg.DestinationID, orgID)
	if err != nil {
		_ = s.failJob(ctx, orgID, job, "BACKUP_STORAGE_UPLOAD_FAILED", err.Error())
		return
	}
	key, err := StorageKey(cfg.PathPrefix, job.DatabaseID, job.Engine, job.ID, time.Now(), job.Format)
	if err != nil {
		_ = s.failJob(ctx, orgID, job, "BACKUP_INVALID_CONFIGURATION", err.Error())
		return
	}
	job.StorageKey = key
	_ = job.Transition(domain.BackupUploading)
	_, _ = s.Store.UpdateJob(ctx, job)

	if err := s.upload(ctx, provider, key, path, job); err != nil {
		_ = s.failJob(ctx, orgID, job, "BACKUP_STORAGE_UPLOAD_FAILED", err.Error())
		return
	}
	_ = job.Transition(domain.BackupVerifying)
	head, err := provider.HeadObject(ctx, storage.HeadObjectInput{Key: key})
	if err != nil {
		_ = s.failJob(ctx, orgID, job, "BACKUP_STORAGE_VERIFY_FAILED", err.Error())
		return
	}
	if head.ContentLength != size {
		_ = s.failJob(ctx, orgID, job, "BACKUP_STORAGE_VERIFY_FAILED", "stored backup size does not match the generated artifact")
		return
	}

	completed := time.Now()
	job.CompletedAt = &completed
	_ = job.Transition(domain.BackupCompleted)
	_, _ = s.Store.UpdateJob(ctx, job)
	s.Audit.Record(ctx, orgID, "backup.completed", "database", job.DatabaseID.String(), job.ID.String()+" "+key)
	s.notifyBackup(ctx, orgID, job.DatabaseID, job.ID, string(job.Status))

	if cfg.Retention.Type == domain.RetentionLatest {
		s.applyLatestRetention(ctx, provider, job)
	}
}

func (s *DatabaseBackups) streamBackup(ctx context.Context, adapter BackupAdapter, desc DBDescriptor) (string, int64, string, error) {
	file, err := os.CreateTemp("", "paas-backup-*")
	if err != nil {
		return "", 0, "", err
	}
	path := file.Name()
	hash := sha256.New()
	sink := io.MultiWriter(file, hash)
	if err := adapter.CreateBackup(ctx, desc, sink); err != nil {
		_ = file.Close()
		return "", 0, "", err
	}
	if err := file.Close(); err != nil {
		return "", 0, "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", 0, "", err
	}
	return path, info.Size(), hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *DatabaseBackups) upload(ctx context.Context, provider storage.Provider, key, path string, job *domain.BackupJob) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = provider.PutObject(ctx, storage.PutObjectInput{
		Key: key, ContentType: "application/octet-stream", Body: f,
		Metadata: map[string]string{
			"database_id": job.DatabaseID.String(),
			"backup_id":   job.ID.String(),
			"engine":      job.Engine,
			"format":      job.Format,
			"checksum":    job.Checksum,
		},
	})
	return err
}

func (s *DatabaseBackups) applyLatestRetention(ctx context.Context, provider storage.Provider, current *domain.BackupJob) {
	older, err := s.Store.ListJobsByDatabase(ctx, current.DatabaseID, 100)
	if err != nil {
		return
	}
	for _, b := range older {
		if b.ID == current.ID || b.Status != domain.BackupCompleted || b.StorageKey == "" {
			continue
		}
		_ = provider.DeleteObject(ctx, storage.DeleteObjectInput{Key: b.StorageKey})
	}
}

func (s *DatabaseBackups) runRestore(ctx context.Context, orgID uuid.UUID, jobID uuid.UUID) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout())
	defer cancel()

	rj, err := s.Store.GetRestoreJob(ctx, jobID)
	if err != nil || rj.Terminal() {
		return
	}
	if s.Locks != nil {
		lock, ok, err := s.Locks.Acquire(ctx, "db:"+rj.TargetDatabaseID.String()+":restore", s.timeout())
		if err != nil || !ok {
			_ = s.failRestore(ctx, rj, "RESTORE_ALREADY_RUNNING", "another restore is running for this database")
			return
		}
		defer func() { _ = s.Locks.Release(ctx, lock) }()
	}
	now := time.Now()
	rj.StartedAt = &now
	_ = rj.Transition(domain.RestorePreparing)
	_, _ = s.Store.UpdateRestoreJob(ctx, rj)

	backup, err := s.Store.GetJob(ctx, rj.BackupID)
	if err != nil {
		_ = s.failRestore(ctx, rj, "RESTORE_BACKUP_NOT_FOUND", err.Error())
		return
	}
	provider, err := s.Destinations.GetProvider(ctx, backup.DestinationID, orgID)
	if err != nil {
		_ = s.failRestore(ctx, rj, "RESTORE_STORAGE_ACCESS_FAILED", err.Error())
		return
	}
	if err := rj.Transition(domain.RestoreDownload); err != nil {
		_ = s.failRestore(ctx, rj, "RESTORE_STATE_INVALID", err.Error())
		return
	}
	if _, err := s.Store.UpdateRestoreJob(ctx, rj); err != nil {
		_ = s.failRestore(ctx, rj, "RESTORE_STATE_UPDATE_FAILED", err.Error())
		return
	}
	obj, err := provider.GetObject(ctx, storage.GetObjectInput{Key: backup.StorageKey})
	if err != nil {
		_ = s.failRestore(ctx, rj, "RESTORE_BACKUP_NOT_FOUND", err.Error())
		return
	}

	file, err := os.CreateTemp("", "paas-restore-*")
	if err != nil {
		_ = obj.Body.Close()
		_ = s.failRestore(ctx, rj, "RESTORE_DISK_FULL", err.Error())
		return
	}
	path := file.Name()
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(file, hash), obj.Body)
	if err != nil {
		_ = file.Close()
		_ = obj.Body.Close()
		_ = s.failRestore(ctx, rj, "RESTORE_DOWNLOAD_FAILED", err.Error())
		return
	}
	if err := file.Close(); err != nil {
		_ = obj.Body.Close()
		_ = s.failRestore(ctx, rj, "RESTORE_DISK_WRITE_FAILED", err.Error())
		return
	}
	_ = obj.Body.Close()
	defer os.Remove(path)
	if written == 0 {
		_ = s.failRestore(ctx, rj, "RESTORE_EMPTY_BACKUP", "backup object is empty")
		return
	}
	if (backup.SizeBytes > 0 && written != backup.SizeBytes) || (obj.ContentLength > 0 && written != obj.ContentLength) {
		_ = s.failRestore(ctx, rj, "RESTORE_SIZE_MISMATCH", "backup object size does not match its metadata")
		return
	}

	if backup.Checksum != "" && hex.EncodeToString(hash.Sum(nil)) != backup.Checksum {
		_ = s.failRestore(ctx, rj, "RESTORE_CHECKSUM_MISMATCH", "checksum verification failed")
		return
	}

	target, err := s.Databases.Get(ctx, rj.TargetDatabaseID, orgID)
	if err != nil {
		_ = s.failRestore(ctx, rj, "RESTORE_DATABASE_UNAVAILABLE", err.Error())
		return
	}
	pass, err := s.Passwords.Decrypt(target.PassEnc)
	if err != nil {
		_ = s.failRestore(ctx, rj, "RESTORE_DATABASE_UNAVAILABLE", "cannot decrypt credentials")
		return
	}
	desc := DBDescriptor{
		Engine: string(target.Engine), ContainerID: target.ContainerID, User: target.User,
		Password: pass, DBName: target.DBName, Version: target.Version,
	}
	adapter, err := adapterForEngine(desc.Engine)
	if err != nil {
		_ = s.failRestore(ctx, rj, "RESTORE_UNSUPPORTED_ENGINE", err.Error())
		return
	}
	if err := rj.Transition(domain.RestoreRunning); err != nil {
		_ = s.failRestore(ctx, rj, "RESTORE_STATE_INVALID", err.Error())
		return
	}
	if _, err := s.Store.UpdateRestoreJob(ctx, rj); err != nil {
		_ = s.failRestore(ctx, rj, "RESTORE_STATE_UPDATE_FAILED", err.Error())
		return
	}

	f, err := os.Open(path)
	if err != nil {
		_ = s.failRestore(ctx, rj, "RESTORE_DATABASE_UNAVAILABLE", err.Error())
		return
	}
	defer f.Close()
	if err := adapter.Restore(ctx, desc, f); err != nil {
		_ = s.failRestore(ctx, rj, "RESTORE_FAILED", err.Error())
		return
	}
	if err := rj.Transition(domain.RestoreVerifying); err != nil {
		_ = s.failRestore(ctx, rj, "RESTORE_STATE_INVALID", err.Error())
		return
	}
	completed := time.Now()
	rj.CompletedAt = &completed
	if err := rj.Transition(domain.RestoreCompleted); err != nil {
		_ = s.failRestore(ctx, rj, "RESTORE_STATE_INVALID", err.Error())
		return
	}
	_, _ = s.Store.UpdateRestoreJob(ctx, rj)
	s.Audit.Record(ctx, orgID, "restore.completed", "database", rj.TargetDatabaseID.String(), rj.ID.String())
}

func (s *DatabaseBackups) failRestore(ctx context.Context, rj *domain.RestoreJob, code, msg string) error {
	now := time.Now()
	rj.ErrorCode = code
	rj.ErrorMessage = msg
	rj.CompletedAt = &now
	if err := rj.Transition(domain.RestoreFailed); err != nil {
		return err
	}
	_, err := s.Store.UpdateRestoreJob(ctx, rj)
	return err
}
