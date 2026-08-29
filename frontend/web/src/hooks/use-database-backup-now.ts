import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { BackupJob } from "../api/types";

export function useDatabaseBackupNow(serviceId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (configurationId: string) => apiPost<BackupJob>(`/api/v1/services/${serviceId}/backups`, { configuration_id: configurationId }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["database-backups", "service", serviceId] });
    },
  });
}
