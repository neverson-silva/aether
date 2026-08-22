import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeleteEnvironment(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (environmentID: string) =>
      apiDelete(`/api/v1/projects/${projectId}/environments/${environmentID}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["environments", projectId] }),
  });
}
