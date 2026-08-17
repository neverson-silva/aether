import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useSetDefaultEnvironment(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (environmentID: string) =>
      apiPost(`/api/v1/projects/${projectId}/environments/${environmentID}/default`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["environments", projectId] }),
  });
}
