import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useDatabaseBackupCancel(serviceId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (backupId: string) => apiPost(`/api/v1/services/${serviceId}/backups/${backupId}/cancel`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["database-backups", "service", serviceId] }),
  });
}
