import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { BackupJob } from "../api/types";

export function useDatabaseBackups(dbId: string, limit = 50) {
  return useQuery({
    queryKey: ["database-backups", dbId, limit],
    queryFn: () => apiGet<BackupJob[]>(`/api/v1/databases/${dbId}/backups?limit=${limit}`),
    enabled: !!dbId,
    refetchInterval: (query) => {
      const jobs = query.state.data as BackupJob[] | undefined;
      const active = jobs?.some((j) => !["completed", "failed", "cancelled"].includes(j.status));
      return active ? 4000 : false;
    },
  });
}
