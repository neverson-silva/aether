import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { PreflightResult } from "../api/types";

export function useDatabaseBackupPreflight(serviceId: string, backupId: string, enabled: boolean) {
  return useQuery({
    queryKey: ["database-backup-preflight", "service", serviceId, backupId],
    queryFn: () => apiGet<PreflightResult>(`/api/v1/services/${serviceId}/backups/${backupId}/preflight`),
    enabled: enabled && !!backupId,
  });
}
