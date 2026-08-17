import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeleteCompose() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/compose/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["compose"] }),
  });
}
