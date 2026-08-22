import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPatch, apiPost } from "../api/client";
import type { BackupConfig } from "../api/types";

export function useUpsertDatabaseBackupConfig(dbId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cfg: Partial<BackupConfig>) =>
      cfg.id
        ? apiPatch<BackupConfig>(`/api/v1/databases/${dbId}/backup/configurations/${cfg.id}`, cfg)
        : apiPost<BackupConfig>(`/api/v1/databases/${dbId}/backup/configurations`, cfg),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["database-backup-configs", dbId] }),
  });
}
