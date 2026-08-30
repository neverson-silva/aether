package application

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
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

	artifact, sourceErr := s.openRestoreSource(ctx, rj, orgID)
	if sourceErr != nil {
		_ = s.failRestore(ctx, rj, sourceErr.code, sourceErr.msg)
		return
	}
	defer artifact.cleanup()

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
		Password: pass, DBName: target.DBName, Version: target.Version, Format: rj.SourceFormat,
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

	if err := adapter.Restore(ctx, desc, artifact.reader); err != nil {
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
	if s.Cache != nil {
		_ = s.Cache.Invalidate(ctx, "studio:db:"+rj.TargetDatabaseID.String()+":")
	}
	if rj.SourceType == domain.RestoreSourceUpload {
		s.removeUploadArtifact(rj.ID)
	}
	s.notifyRestore(ctx, orgID, rj.TargetDatabaseID, rj.ID, string(rj.Status))
	s.Audit.Record(ctx, orgID, "restore.completed", "database", rj.TargetDatabaseID.String(), rj.ID.String())
}

type restoreArtifact struct {
	reader  io.ReadCloser
	cleanup func()
}

type restoreSourceError struct {
	code string
	msg  string
}

func (s *DatabaseBackups) openRestoreSource(ctx context.Context, rj *domain.RestoreJob, orgID uuid.UUID) (*restoreArtifact, *restoreSourceError) {
	switch rj.SourceType {
	case domain.RestoreSourceUpload:
		return s.openUploadedSource(ctx, rj)
	case domain.RestoreSourceBackup:
		return s.openBackupSource(ctx, rj, orgID)
	default:
		return nil, &restoreSourceError{"RESTORE_SOURCE_INVALID", "unsupported restore source"}
	}
}

func (s *DatabaseBackups) openUploadedSource(ctx context.Context, rj *domain.RestoreJob) (*restoreArtifact, *restoreSourceError) {
	path := s.uploadArtifactPath(rj.ID)
	st, err := os.Stat(path)
	if err != nil {
		return nil, &restoreSourceError{"RESTORE_UPLOAD_ARTIFACT_MISSING", "uploaded restore artifact is missing"}
	}
	if st.Size() == 0 {
		return nil, &restoreSourceError{"RESTORE_EMPTY_BACKUP", "uploaded file is empty"}
	}
	if rj.SourceSize > 0 && st.Size() != rj.SourceSize {
		return nil, &restoreSourceError{"RESTORE_SIZE_MISMATCH", "uploaded file size changed after upload"}
	}
	if err := rj.Transition(domain.RestoreDownload); err != nil {
		return nil, &restoreSourceError{"RESTORE_STATE_INVALID", err.Error()}
	}
	if _, err := s.Store.UpdateRestoreJob(ctx, rj); err != nil {
		return nil, &restoreSourceError{"RESTORE_STATE_UPDATE_FAILED", err.Error()}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, &restoreSourceError{"RESTORE_DISK_UNREADABLE", err.Error()}
	}
	detected := DetectUploadFormat(path, rj.SourceFormat)
	if detected == "sql.gz" || detected == "gzip" {
		gz, gzErr := gzip.NewReader(f)
		if gzErr != nil {
			_ = f.Close()
			return nil, &restoreSourceError{"RESTORE_INVALID_FORMAT", "uploaded file is not a valid gzip archive"}
		}
		return &restoreArtifact{reader: gz, cleanup: func() { _ = gz.Close(); _ = f.Close() }}, nil
	}
	rj.SourceFormat = detected
	return &restoreArtifact{reader: f, cleanup: func() { _ = f.Close() }}, nil
}

func (s *DatabaseBackups) openBackupSource(ctx context.Context, rj *domain.RestoreJob, orgID uuid.UUID) (*restoreArtifact, *restoreSourceError) {
	if rj.BackupID == nil {
		return nil, &restoreSourceError{"RESTORE_BACKUP_NOT_FOUND", "restore has no source backup"}
	}
	backup, err := s.Store.GetJob(ctx, *rj.BackupID)
	if err != nil {
		return nil, &restoreSourceError{"RESTORE_BACKUP_NOT_FOUND", err.Error()}
	}
	provider, err := s.Destinations.GetProvider(ctx, backup.DestinationID, orgID)
	if err != nil {
		return nil, &restoreSourceError{"RESTORE_STORAGE_ACCESS_FAILED", err.Error()}
	}
	if err := rj.Transition(domain.RestoreDownload); err != nil {
		return nil, &restoreSourceError{"RESTORE_STATE_INVALID", err.Error()}
	}
	if _, err := s.Store.UpdateRestoreJob(ctx, rj); err != nil {
		return nil, &restoreSourceError{"RESTORE_STATE_UPDATE_FAILED", err.Error()}
	}
	obj, err := provider.GetObject(ctx, storage.GetObjectInput{Key: backup.StorageKey})
	if err != nil {
		return nil, &restoreSourceError{"RESTORE_BACKUP_NOT_FOUND", err.Error()}
	}

	file, err := os.CreateTemp("", "paas-restore-*")
	if err != nil {
		_ = obj.Body.Close()
		return nil, &restoreSourceError{"RESTORE_DISK_FULL", err.Error()}
	}
	path := file.Name()
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(file, hash), obj.Body)
	if err != nil {
		_ = file.Close()
		_ = obj.Body.Close()
		_ = os.Remove(path)
		return nil, &restoreSourceError{"RESTORE_DOWNLOAD_FAILED", err.Error()}
	}
	if err := file.Close(); err != nil {
		_ = obj.Body.Close()
		_ = os.Remove(path)
		return nil, &restoreSourceError{"RESTORE_DISK_WRITE_FAILED", err.Error()}
	}
	_ = obj.Body.Close()
	if written == 0 {
		_ = os.Remove(path)
		return nil, &restoreSourceError{"RESTORE_EMPTY_BACKUP", "backup object is empty"}
	}
	if (backup.SizeBytes > 0 && written != backup.SizeBytes) || (obj.ContentLength > 0 && written != obj.ContentLength) {
		_ = os.Remove(path)
		return nil, &restoreSourceError{"RESTORE_SIZE_MISMATCH", "backup object size does not match its metadata"}
	}
	if backup.Checksum != "" && hex.EncodeToString(hash.Sum(nil)) != backup.Checksum {
		_ = os.Remove(path)
		return nil, &restoreSourceError{"RESTORE_CHECKSUM_MISMATCH", "checksum verification failed"}
	}
	f, err := os.Open(path)
	if err != nil {
		_ = os.Remove(path)
		return nil, &restoreSourceError{"RESTORE_DISK_UNREADABLE", err.Error()}
	}
	return &restoreArtifact{reader: f, cleanup: func() { _ = f.Close(); _ = os.Remove(path) }}, nil
}

func (s *DatabaseBackups) uploadArtifactPath(restoreID uuid.UUID) string {
	return filepath.Join(s.UploadRoot, restoreID.String(), "payload")
}

func sanitizeSourceFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "..", "")
	if name == "." || name == "/" || name == "" {
		return "restore"
	}
	if len(name) > 255 {
		name = name[len(name)-255:]
	}
	return name
}

var gzipMagic = []byte{0x1f, 0x8b}

func DetectUploadFormat(path, current string) string {
	if current == "sql.gz" || current == "gzip" {
		return "sql.gz"
	}
	var only [8]byte
	f, err := os.Open(path)
	if err == nil {
		_, _ = f.Read(only[:])
		_ = f.Close()
	}
	if len(only) >= len(gzipMagic) && only[0] == gzipMagic[0] && only[1] == gzipMagic[1] {
		return "sql.gz"
	}
	if len(only) >= 5 && string(only[:5]) == "PGDMP" {
		return "dump"
	}
	if len(only) >= 6 && string(only[:6]) == "ustar\x00" {
		return "tar"
	}
	if current != "" {
		return current
	}
	return fileNameExt(filepath.Base(path))
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
	if rj.SourceType == domain.RestoreSourceUpload {
		s.removeUploadArtifact(rj.ID)
	}
	return err
}
