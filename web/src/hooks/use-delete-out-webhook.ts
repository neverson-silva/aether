import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeleteOutWebhook() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/webhooks/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["out-webhooks"] }),
  });
}
