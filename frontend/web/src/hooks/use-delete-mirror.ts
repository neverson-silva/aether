import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeleteMirror() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/mirrors/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["mirrors"] }),
  });
}
