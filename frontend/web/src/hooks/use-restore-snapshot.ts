import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useRestoreSnapshot() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { id: string; volume?: string }) => apiPost(`/api/v1/snapshots/${body.id}/restore`, { volume: body.volume }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["snapshots"] }),
  });
}
