import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { BackupConfig } from "../api/types";

export function useDatabaseBackupConfig(dbId: string) {
  return useQuery({
    queryKey: ["database-backup-configs", dbId],
    queryFn: () => apiGet<BackupConfig[]>(`/api/v1/databases/${dbId}/backup/configurations`),
    enabled: !!dbId,
  });
}
