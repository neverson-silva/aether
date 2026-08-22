import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Stats } from "../api/types";

// Live container metrics for a database, auto-loaded (mirrors useStats for
// application services). Refreshes every few seconds while the page is open.
export function useDatabaseStats(dbId: string) {
  return useQuery({
    queryKey: ["database-stats", dbId],
    queryFn: () => apiGet<Stats>(`/api/v1/databases/${dbId}/stats`),
    enabled: !!dbId,
    refetchInterval: 5000,
    refetchOnWindowFocus: false,
  });
}