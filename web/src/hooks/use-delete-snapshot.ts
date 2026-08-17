import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeleteSnapshot() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/snapshots/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["snapshots"] }),
  });
}
