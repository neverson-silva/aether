import { useMutation } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useRestoreDatabase() {
  return useMutation({
    mutationFn: ({ dbId, backupId }: { dbId: string; backupId: string }) =>
      apiPost(`/api/v1/databases/${dbId}/restore`, { backup_id: backupId }),
  });
}
