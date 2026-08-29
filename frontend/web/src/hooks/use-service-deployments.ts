import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Deployment } from "../api/types";
import { qk } from "./query-keys";

export function useServiceDeployments(serviceId: string, enabled = true) {
  return useQuery({
    queryKey: qk.serviceDeployments(serviceId),
    queryFn: () => apiGet<Deployment[]>(`/api/v1/services/${serviceId}/deployments`),
    enabled: enabled && !!serviceId,
  });
}
