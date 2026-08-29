import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import { qk } from "./query-keys";

export type ServiceContainer = {
  id: string;
  name: string;
  status: string;
};

export function useServiceContainers(id: string, enabled = true) {
  return useQuery({
    queryKey: qk.serviceContainers(id),
    queryFn: () => apiGet<ServiceContainer[]>(`/api/v1/services/${id}/containers`),
    enabled: enabled && !!id,
  });
}
