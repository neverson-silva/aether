import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import { qk } from "./query-keys";

export interface ServiceEnvironmentVariable {
  name: string;
  value: string;
  secret: boolean;
}

export function useServiceEnvironment(serviceId: string, enabled = true) {
  return useQuery({
    queryKey: qk.serviceEnvironment(serviceId),
    queryFn: () => apiGet<{ env: ServiceEnvironmentVariable[] }>(`/api/v1/services/${serviceId}/environment`),
    enabled: enabled && !!serviceId,
  });
}
