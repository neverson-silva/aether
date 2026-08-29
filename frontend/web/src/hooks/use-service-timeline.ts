import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { TimelineEvent } from "../api/types";
import { qk } from "./query-keys";

export function useServiceTimeline(serviceId: string, enabled = true) {
  return useQuery({
    queryKey: qk.serviceTimeline(serviceId),
    queryFn: () => apiGet<TimelineEvent[]>(`/api/v1/services/${serviceId}/timeline`),
    enabled: enabled && !!serviceId,
  });
}
