import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { MonitoringResourcePoint } from "./types";
import type { MonitoringWindow } from "./use-monitoring-history";

export function useResourceHistory(resourceId: string | null, window: MonitoringWindow) {
  return useQuery({
    queryKey: ["monitoring-resource-history", resourceId, window],
    queryFn: () =>
      apiGet<{ resource_id: string; window: string; points: MonitoringResourcePoint[] }>(
        `/api/v1/monitoring/resources/${resourceId}/history?window=${window}`,
      ).then((r) => r.points),
    enabled: !!resourceId,
    refetchInterval: 15_000,
    refetchOnWindowFocus: false,
  });
}
