import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useDatabaseBackupCancel(dbId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (backupId: string) => apiPost(`/api/v1/databases/${dbId}/backups/${backupId}/cancel`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["database-backups", dbId] }),
  });
}
