import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { CronJob } from "../api/types";

export function useCronJobs(appID: string) {
  return useQuery({
    queryKey: ["cron-jobs", appID],
    queryFn: () => apiGet<CronJob[]>(`/api/v1/apps/${appID}/cron-jobs`),
    enabled: !!appID,
  });
}
