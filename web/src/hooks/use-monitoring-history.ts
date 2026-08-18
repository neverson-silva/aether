import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { MonitoringHistoryPoint } from "./types";

export type MonitoringWindow = "5m" | "15m" | "1h" | "6h" | "24h" | "7d";

// Low-frequency telemetry (window-level trend), refetched sparingly.
export function useMonitoringHistory(window: MonitoringWindow) {
  return useQuery({
    queryKey: ["monitoring-history", window],
    queryFn: () =>
      apiGet<{ window: string; points: MonitoringHistoryPoint[] }>(`/api/v1/monitoring/history?window=${window}`).then((r) => r.points),
    refetchInterval: 15_000,
    refetchOnWindowFocus: false,
  });
}
