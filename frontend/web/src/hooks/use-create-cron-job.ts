import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { CronJob } from "../api/types";

export function useCreateCronJob(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; schedule: string; command: string }) =>
      apiPost<CronJob>(`/api/v1/apps/${appID}/cron-jobs`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cron-jobs", appID] }),
  });
}
