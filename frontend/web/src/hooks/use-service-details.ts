import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { ServiceSummary } from "../api/types";
import { qk } from "./query-keys";

export function useServiceDetails(id: string, enabled = true) {
  return useQuery({
    queryKey: qk.service(id),
    queryFn: () => apiGet<ServiceSummary>(`/api/v1/services/${id}`),
    enabled: enabled && !!id,
  });
}

export function useServices(projectId?: string, environmentId?: string) {
  const params = new URLSearchParams();
  if (projectId) params.set("project_id", projectId);
  if (environmentId) params.set("environment_id", environmentId);
  const suffix = params.toString() ? `?${params.toString()}` : "";
  return useQuery({
    queryKey: ["services", projectId ?? "", environmentId ?? ""],
    queryFn: () => apiGet<ServiceSummary[]>(`/api/v1/services${suffix}`),
    enabled: true,
  });
}
