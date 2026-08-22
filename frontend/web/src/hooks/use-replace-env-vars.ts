import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPut } from "../api/client";

export function useReplaceEnvVars(projectId: string, environmentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (entries: Record<string, { value: string; secret: boolean }>) =>
      apiPut(`/api/v1/projects/${projectId}/environments/${environmentId}/variables`, entries),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["env-vars", projectId, environmentId] });
      qc.invalidateQueries({ queryKey: ["apps"] });
    },
  });
}
