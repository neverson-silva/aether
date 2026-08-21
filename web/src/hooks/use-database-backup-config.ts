import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { BackupConfig } from "../api/types";

export function useDatabaseBackupConfig(dbId: string) {
  return useQuery({
    queryKey: ["database-backup-config", dbId],
    queryFn: () => apiGet<BackupConfig | null>(`/api/v1/databases/${dbId}/backup/configuration`),
    enabled: !!dbId,
  });
}
