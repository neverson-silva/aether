import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { App } from "../api/types";

export function useProjectApps(projectId: string) {
  return useQuery({
    queryKey: ["apps", "project", projectId],
    queryFn: () => apiGet<App[]>(`/api/v1/apps?project_id=${projectId}`),
    enabled: !!projectId,
  });
}
