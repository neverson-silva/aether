import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeleteDatabaseBackupConfig(serviceId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (configId: string) => apiDelete(`/api/v1/services/${serviceId}/backup/configurations/${configId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["database-backup-configs", "service", serviceId] }),
  });
}
