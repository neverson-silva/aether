import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useComposeUp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost(`/api/v1/compose/${id}/up`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["compose"] }),
  });
}
