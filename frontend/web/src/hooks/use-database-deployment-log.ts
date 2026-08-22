import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";

export interface DatabaseDeploymentLog {
  content: string;
}

export function useDatabaseDeploymentLog(dbId: string, depId: string | null) {
  return useQuery({
    queryKey: ["database-deploy-log", dbId, depId],
    enabled: !!depId,
    queryFn: () => apiGet<DatabaseDeploymentLog>(`/api/v1/databases/${dbId}/deployments/${depId}/logs`),
  });
}