import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { Environment } from "./types";

export function useCreateEnvironment(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; description?: string; color?: string }) =>
      apiPost<Environment>(`/api/v1/projects/${projectId}/environments`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["environments", projectId] }),
  });
}
