import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPatch } from "../api/client";
import type { Environment } from "./types";

export function useUpdateEnvironment(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { environmentID: string; name: string; description?: string; color?: string }) =>
      apiPatch<Environment>(`/api/v1/projects/${projectId}/environments/${body.environmentID}`, {
        name: body.name, description: body.description, color: body.color,
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["environments", projectId] }),
  });
}
