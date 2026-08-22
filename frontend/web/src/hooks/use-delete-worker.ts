import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeleteWorker(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/workers/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workers", appID] }),
  });
}
