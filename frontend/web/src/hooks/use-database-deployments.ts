import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";

export interface DatabaseDeployment {
  id: string;
  number: number;
  status: string;
  trigger: string;
  triggered_by?: string;
  container_id?: string;
  error?: string;
  created_at: string;
  started_at?: string | null;
  finished_at?: string | null;
}

export function useDatabaseDeployments(dbId: string) {
  return useQuery({
    queryKey: ["database-deployments", dbId],
    queryFn: () => apiGet<DatabaseDeployment[]>(`/api/v1/databases/${dbId}/deployments`),
    enabled: !!dbId,
  });
}