import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeleteCluster() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/clusters/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["clusters"] }),
  });
}
