import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeleteDatabaseBackupConfig(dbId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (configId: string) => apiDelete(`/api/v1/databases/${dbId}/backup/configurations/${configId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["database-backup-configs", dbId] }),
  });
}
