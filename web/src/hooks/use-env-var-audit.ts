import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { VariableAudit } from "./types";

export function useEnvVarAudit(projectId: string, environmentId: string | null) {
  return useQuery({
    queryKey: ["env-vars-audit", projectId, environmentId],
    queryFn: () => apiGet<VariableAudit[]>(`/api/v1/projects/${projectId}/environments/${environmentId}/variables/audit`),
    enabled: !!environmentId,
  });
}
