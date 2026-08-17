import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { TimelineEvent } from "../api/types";
import { qk } from "./query-keys";

export function useTimeline(appID: string) {
  return useQuery({
    queryKey: qk.timeline(appID),
    queryFn: () => apiGet<TimelineEvent[]>(`/api/v1/apps/${appID}/timeline`),

    enabled: !!appID,
  });
}
