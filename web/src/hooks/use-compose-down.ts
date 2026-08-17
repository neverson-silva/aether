import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useComposeDown() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost(`/api/v1/compose/${id}/down`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["compose"] }),
  });
}
