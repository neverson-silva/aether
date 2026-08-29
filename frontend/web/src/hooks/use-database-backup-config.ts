import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { BackupConfig } from "../api/types";

export function useDatabaseBackupConfig(serviceId: string) {
  return useQuery({
    queryKey: ["database-backup-configs", "service", serviceId],
    queryFn: () => apiGet<BackupConfig[]>(`/api/v1/services/${serviceId}/backup/configurations`),
    enabled: !!serviceId,
  });
}
