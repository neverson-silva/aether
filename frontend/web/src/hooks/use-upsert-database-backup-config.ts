import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPatch, apiPost } from "../api/client";
import type { BackupConfig } from "../api/types";

export function useUpsertDatabaseBackupConfig(serviceId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cfg: Partial<BackupConfig>) =>
      cfg.id
        ? apiPatch<BackupConfig>(`/api/v1/services/${serviceId}/backup/configurations/${cfg.id}`, cfg)
        : apiPost<BackupConfig>(`/api/v1/services/${serviceId}/backup/configurations`, cfg),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["database-backup-configs", "service", serviceId] }),
  });
}
