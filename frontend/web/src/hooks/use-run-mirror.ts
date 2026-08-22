import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useRunMirror() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost(`/api/v1/mirrors/${id}/run`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["mirrors"] }),
  });
}
