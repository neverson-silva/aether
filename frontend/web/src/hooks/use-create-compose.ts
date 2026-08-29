import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import { qk } from "./query-keys";

export function useCreateCompose() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { project_id: string; name: string; compose: string }) => apiPost("/api/v1/compose", body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.composes });
    },
  });
}
