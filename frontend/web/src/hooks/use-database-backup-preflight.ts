import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { PreflightResult } from "../api/types";

export function useDatabaseBackupPreflight(dbId: string, backupId: string, enabled: boolean) {
  return useQuery({
    queryKey: ["database-backup-preflight", dbId, backupId],
    queryFn: () => apiGet<PreflightResult>(`/api/v1/databases/${dbId}/backups/${backupId}/preflight?target_database_id=${dbId}`),
    enabled: enabled && !!backupId,
  });
}
