import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Stats } from "../api/types";
import { qk } from "./query-keys";

export function useServiceStats(serviceId: string, enabled = true) {
  return useQuery({
    queryKey: qk.serviceStats(serviceId),
    queryFn: () => apiGet<Stats>(`/api/v1/services/${serviceId}/stats`),
    enabled: enabled && !!serviceId,
  });
}
