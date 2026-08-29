import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { BackupJob } from "../api/types";

export function useDatabaseBackups(serviceId: string, limit = 50) {
  return useQuery({
    queryKey: ["database-backups", "service", serviceId, limit],
    queryFn: () => apiGet<BackupJob[]>(`/api/v1/services/${serviceId}/backups?limit=${limit}`),
    enabled: !!serviceId,
  });
}
