import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeleteCronJob(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/cron-jobs/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cron-jobs", appID] }),
  });
}
