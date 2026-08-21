import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPut } from "../api/client";
import type { BackupConfig } from "../api/types";

export function useUpsertDatabaseBackupConfig(dbId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cfg: Partial<BackupConfig>) => apiPut<BackupConfig>(`/api/v1/databases/${dbId}/backup/configuration`, cfg),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["database-backup-config", dbId] }),
  });
}
