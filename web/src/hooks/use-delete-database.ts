import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeleteDatabase() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/databases/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["databases"] }),
  });
}
