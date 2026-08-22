import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { ProjectVariable } from "./types";

export function useProjectVars(projectId: string) {
  return useQuery({
    queryKey: ["project-vars", projectId],
    queryFn: async () => {
      const data = await apiGet<{ variables: ProjectVariable[] }>(`/api/v1/projects/${projectId}/variables?secrets=1`);
      return data.variables ?? [];
    },
  });
}
