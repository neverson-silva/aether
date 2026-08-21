import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { BackupJob } from "../api/types";

export function useDatabaseBackupNow(dbId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiPost<BackupJob>(`/api/v1/databases/${dbId}/backups`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["database-backups", dbId] });
    },
  });
}
