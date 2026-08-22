import { useQuery } from "@tanstack/react-query";
import { apiGet, ApiError } from "../api/client";
import type { Stats } from "../api/types";
import { qk } from "./query-keys";

export function useStats(appID: string, enabled = true) {
  return useQuery({
    queryKey: qk.stats(appID),
    queryFn: async () => {
      try {
        return await apiGet<Stats>(`/api/v1/apps/${appID}/stats`);
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) return null;
        throw error;
      }
    },

    enabled: !!appID && enabled,
    retry: false,
  });
}
