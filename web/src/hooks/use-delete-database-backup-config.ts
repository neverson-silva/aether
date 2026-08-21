import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeleteDatabaseBackupConfig(dbId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiDelete(`/api/v1/databases/${dbId}/backup/configuration`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["database-backup-config", dbId] }),
  });
}
