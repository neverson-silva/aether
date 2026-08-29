import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { BackupJob } from "../api/types";

export function useStartServiceDatabaseBackup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (serviceId: string) => apiPost<BackupJob>(`/api/v1/services/${serviceId}/backups`, {}),
    onSuccess: (_, serviceId) => {
      qc.invalidateQueries({ queryKey: ["database-backups", "service", serviceId] });
      qc.invalidateQueries({ queryKey: ["services"] });
    },
  });
}
