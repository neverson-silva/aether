import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeleteCronJob(appID: string, canonical = false) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(canonical ? `/api/v1/services/${appID}/cron-jobs/${id}` : `/api/v1/cron-jobs/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cron-jobs", appID] }),
  });
}
