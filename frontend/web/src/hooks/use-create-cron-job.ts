import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { CronJob } from "../api/types";

export function useCreateCronJob(appID: string, canonical = false) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; schedule: string; command: string }) =>
      apiPost<CronJob>(canonical ? `/api/v1/services/${appID}/cron-jobs` : `/api/v1/apps/${appID}/cron-jobs`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cron-jobs", appID] }),
  });
}
