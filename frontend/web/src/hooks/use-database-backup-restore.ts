import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { RestoreJob } from "../api/types";

export function useDatabaseBackupRestore(serviceId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (backupId: string) =>
      apiPost<RestoreJob>(`/api/v1/services/${serviceId}/backups/${backupId}/restore`, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["database-restore-jobs", "service", serviceId] }),
  });
}
