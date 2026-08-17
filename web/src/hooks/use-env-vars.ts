import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { EnvironmentVariable } from "./types";

export function useEnvVars(projectId: string, environmentId: string | null) {
  return useQuery({
    queryKey: ["env-vars", projectId, environmentId],
    enabled: !!projectId && !!environmentId,
    queryFn: async () => {
      const data = await apiGet<{ variables: EnvironmentVariable[] }>(`/api/v1/projects/${projectId}/environments/${environmentId}/variables?secrets=1`);
      return data.variables ?? [];
    },
  });
}
