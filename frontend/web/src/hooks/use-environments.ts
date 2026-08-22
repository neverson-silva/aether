import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { EnvSummary } from "./types";

export function useEnvironments(projectId: string) {
  return useQuery({
    queryKey: ["environments", projectId],
    queryFn: () => apiGet<EnvSummary[]>(`/api/v1/projects/${projectId}/environments`),
  });
}
