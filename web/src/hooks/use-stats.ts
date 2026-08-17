import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Stats } from "../api/types";
import { qk } from "./query-keys";

export function useStats(appID: string) {
  return useQuery({
    queryKey: qk.stats(appID),
    queryFn: () => apiGet<Stats>(`/api/v1/apps/${appID}/stats`),

    enabled: !!appID,
  });
}
