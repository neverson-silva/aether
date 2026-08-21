import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { RestoreJob } from "../api/types";

export function useDatabaseBackupRestore(dbId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (backupId: string) =>
      apiPost<RestoreJob>(`/api/v1/databases/${dbId}/backups/${backupId}/restore`, { target_database_id: dbId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["database-restore-jobs", dbId] }),
  });
}
