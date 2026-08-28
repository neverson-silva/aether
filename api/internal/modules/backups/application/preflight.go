package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"aether/internal/modules/backups/domain"
	"aether/internal/platform/storage"
)

type PreflightCheck struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type PreflightResult struct {
	Compatible bool             `json:"compatible"`
	Ready      bool             `json:"ready"`
	Checks     []PreflightCheck `json:"checks"`
}

func (s *DatabaseBackups) RestorePreflight(ctx context.Context, backupID, targetDBID, orgID uuid.UUID) (*PreflightResult, error) {
	result := &PreflightResult{Compatible: true, Ready: true}

	backup, err := s.Store.GetJob(ctx, backupID)
	if err != nil {
		return nil, err
	}
	if _, err := s.Databases.Get(ctx, backup.DatabaseID, orgID); err != nil {
		return nil, err
	}
	result.Checks = append(result.Checks, check("Backup exists", backup.Status == domain.BackupCompleted, string(backup.Status)))
	if backup.Status != domain.BackupCompleted {
		result.Ready = false
	}

	result.Checks = append(result.Checks, check("Checksum present", backup.Checksum != "", "sha256"))
	if backup.Checksum == "" {
		result.Ready = false
	}

	target, err := s.Databases.Get(ctx, targetDBID, orgID)
	if err != nil {
		result.Checks = append(result.Checks, check("Target database available", false, err.Error()))
		result.Ready = false
		result.Compatible = false
		return result, nil
	}
	result.Checks = append(result.Checks, check("Target database available", true, target.Name))

	engineOK := backup.Engine != "" && backup.Engine == string(target.Engine)
	compatMsg := backup.Engine + " → " + string(target.Engine)
	result.Checks = append(result.Checks, check("Engine compatibility", engineOK, compatMsg))
	if !engineOK {
		result.Ready = false
		result.Compatible = false
	}
	if engineOK {
		versionErr := validateRestoreCompatibility(backup.Engine, backup.EngineVersion, target.Version)
		result.Checks = append(result.Checks, check("Version compatibility", versionErr == nil, versionCheckMessage(backup.Engine, backup.EngineVersion, target.Version, versionErr)))
		if versionErr != nil {
			result.Ready = false
			result.Compatible = false
		}
	}

	provider, err := s.Destinations.GetProvider(ctx, backup.DestinationID, orgID)
	if err != nil {
		result.Checks = append(result.Checks, check("Storage accessible", false, err.Error()))
		result.Ready = false
	} else {
		head, err := provider.HeadObject(ctx, storage.HeadObjectInput{Key: backup.StorageKey})
		if err != nil {
			result.Checks = append(result.Checks, check("Backup object exists", false, err.Error()))
			result.Ready = false
		} else if head.ContentLength == 0 {
			result.Checks = append(result.Checks, check("Backup object is not empty", false, "backup object is empty"))
			result.Ready = false
		} else if backup.SizeBytes > 0 && head.ContentLength != backup.SizeBytes {
			result.Checks = append(result.Checks, check("Backup object size", false, "backup object size does not match its metadata"))
			result.Ready = false
		} else {
			result.Checks = append(result.Checks, check("Backup object exists", true, backup.StorageKey))
		}
	}

	if engineOK {
		pass, err := s.Passwords.Decrypt(target.PassEnc)
		if err != nil {
			result.Checks = append(result.Checks, check("Restore credentials available", false, "cannot decrypt credentials"))
			result.Ready = false
		} else {
			desc := DBDescriptor{
				Engine: string(target.Engine), ContainerID: target.ContainerID, User: target.User,
				Password: pass, DBName: target.DBName, Version: target.Version,
			}
			adapter, aerr := adapterForEngine(desc.Engine)
			if aerr != nil {
				result.Checks = append(result.Checks, check("Restore tool available", false, aerr.Error()))
				result.Ready = false
			} else if verr := adapter.Validate(ctx, desc); verr != nil {
				result.Checks = append(result.Checks, check("Restore tool available", false, verr.Error()))
				result.Ready = false
			} else {
				result.Checks = append(result.Checks, check("Restore tool available", true, "available"))
			}
		}
	}

	return result, nil
}

func versionCheckMessage(engine, sourceVersion, targetVersion string, err error) string {
	if err != nil {
		return err.Error()
	}
	if sourceVersion == "" {
		return "source version unavailable"
	}
	return fmt.Sprintf("%s %s → %s %s", displayEngine(engine), sourceVersion, displayEngine(engine), targetVersion)
}

func check(name string, ok bool, msg string) PreflightCheck {
	return PreflightCheck{Name: name, OK: ok, Message: msg}
}
