import { useQuery } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import { qk } from "./query-keys";

export function useServiceConnection(serviceId: string, enabled = false) {
  return useQuery({
    queryKey: [...qk.service(serviceId), "connection"],
    queryFn: () => apiPost<{ dsn: string }>(`/api/v1/services/${serviceId}/connection/reveal`),
    enabled: enabled && !!serviceId,
    retry: false,
  });
}
