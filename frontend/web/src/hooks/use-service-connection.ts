import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import { qk } from "./query-keys";

export function useServiceConnection(serviceId: string, enabled = false) {
  return useQuery({
    queryKey: [...qk.service(serviceId), "connection"],
    queryFn: () => apiGet<{ dsn: string }>(`/api/v1/services/${serviceId}/connection`),
    enabled: enabled && !!serviceId,
    retry: false,
  });
}
