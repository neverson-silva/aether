import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeleteSSO() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/sso/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["sso"] }),
  });
}
