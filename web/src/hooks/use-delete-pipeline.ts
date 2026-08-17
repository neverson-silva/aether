import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeletePipeline() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/pipelines/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["pipelines"] }),
  });
}
