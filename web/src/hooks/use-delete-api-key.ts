import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";
import { qk } from "./query-keys";

export function useDeleteApiKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/api-keys/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.keys }),
  });
}
