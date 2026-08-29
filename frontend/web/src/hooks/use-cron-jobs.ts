import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { CronJob } from "../api/types";

export function useCronJobs(appID: string, canonical = false) {
  return useQuery({
    queryKey: ["cron-jobs", appID],
    queryFn: () => apiGet<CronJob[]>(canonical ? `/api/v1/services/${appID}/cron-jobs` : `/api/v1/apps/${appID}/cron-jobs`),
    enabled: !!appID,
  });
}
