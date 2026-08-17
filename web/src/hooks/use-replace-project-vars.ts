import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPut } from "../api/client";

export function useReplaceProjectVars(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (entries: Record<string, { value: string; secret: boolean }>) =>
      apiPut(`/api/v1/projects/${projectId}/variables`, entries),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["project-vars", projectId] });
      qc.invalidateQueries({ queryKey: ["apps"] });
    },
  });
}
