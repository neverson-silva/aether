import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { BackupJob } from "../api/types";

export function useDatabaseBackups(dbId: string, limit = 50) {
  return useQuery({
    queryKey: ["database-backups", dbId, limit],
    queryFn: () => apiGet<BackupJob[]>(`/api/v1/databases/${dbId}/backups?limit=${limit}`),
    enabled: !!dbId,
  });
}
